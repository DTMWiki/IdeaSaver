package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	dogeTmpTokenPath      = "/auth/tmp_token.json"
	dogeTmpTokenChannel   = "OSS_FULL"
	dogeDefaultAPIServer  = "https://api.dogecloud.com"
	dogeDefaultRegion     = "ap-shanghai"
	dogeDefaultCredTTL    = 15 * time.Minute
	dogeCredRefreshWindow = 1 * time.Minute
)

// OSSClient wraps AWS S3 SDK for DogeCloud OSS operations.
type OSSClient struct {
	client       *s3.Client
	bucket       string
	endpoint     string
	usePathStyle bool
}

type dogeTmpTokenProvider struct {
	apiBase         string
	accessID        string
	secretKey       string
	preferredBucket string
	client          *http.Client
}

type dogeTmpTokenResult struct {
	Credentials aws.Credentials
	S3Endpoint  string
	S3Bucket    string
	Region      string
}

type dogeTmpTokenResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Credentials struct {
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
			SessionToken    string `json:"sessionToken"`
			ExpiredTime     any    `json:"expiredTime"`
			Expiration      any    `json:"expiration"`
			ExpiresAt       any    `json:"expiresAt"`
		} `json:"Credentials"`
		ExpiredTime any    `json:"expiredTime"`
		Expiration  any    `json:"expiration"`
		ExpiresAt   any    `json:"expiresAt"`
		S3Endpoint  string `json:"s3Endpoint"`
		S3Bucket    string `json:"s3Bucket"`
		S3Region    string `json:"s3Region"`
		Endpoint    string `json:"endpoint"`
		Bucket      string `json:"bucket"`
		Region      string `json:"region"`
		Buckets     []struct {
			Name           string `json:"name"`
			S3Bucket       string `json:"s3Bucket"`
			S3Endpoint     string `json:"s3Endpoint"`
			S3EndpointHost string `json:"s3EndpointHost"`
		} `json:"Buckets"`
	} `json:"data"`
}

// NewOSSClient creates a new DogeCloud OSS client using S3-compatible API.
func NewOSSClient(cfg *config.Config) (*OSSClient, error) {
	if cfg.DogeAccessKey == "" || cfg.DogeSecretKey == "" {
		return nil, fmt.Errorf("DogeCloud access credentials are required")
	}

	apiBase := cfg.DogeVCloudAPI
	if apiBase == "" {
		apiBase = dogeDefaultAPIServer
	}

	provider := &dogeTmpTokenProvider{
		apiBase:         strings.TrimRight(apiBase, "/"),
		accessID:        cfg.DogeAccessKey,
		secretKey:       cfg.DogeSecretKey,
		preferredBucket: strings.TrimSpace(cfg.DogeBucket),
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}

	// Resolve OSS endpoint/bucket from temp-token response first, then fallback to env.
	initial, err := provider.fetchToken(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DogeCloud temp token metadata: %w", err)
	}

	endpoint := firstNonEmpty(initial.S3Endpoint, cfg.DogeEndpoint)
	endpoint = normalizeEndpointURL(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("missing OSS endpoint: set IDEASAVER_DOGE_ENDPOINT or ensure tmp-token API returns s3Endpoint")
	}

	bucket := firstNonEmpty(initial.S3Bucket, cfg.DogeBucket)
	if bucket == "" {
		return nil, fmt.Errorf("missing OSS bucket: set IDEASAVER_DOGE_BUCKET or ensure tmp-token API returns s3Bucket")
	}

	region := firstNonEmpty(initial.Region, cfg.DogeRegion)
	if region == "" {
		if parsed, ok := extractRegionFromEndpoint(endpoint); ok {
			region = parsed
		} else {
			region = dogeDefaultRegion
		}
	}

	resolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...any) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: endpoint,
			}, nil
		},
	)

	cachedProvider := aws.NewCredentialsCache(provider, func(o *aws.CredentialsCacheOptions) {
		o.ExpiryWindow = dogeCredRefreshWindow
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithEndpointResolverWithOptions(resolver),
		awsconfig.WithCredentialsProvider(cachedProvider),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	usePathStyle := shouldUsePathStyle(endpoint)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
	})

	return &OSSClient{
		client:       client,
		bucket:       bucket,
		endpoint:     endpoint,
		usePathStyle: usePathStyle,
	}, nil
}

func (p *dogeTmpTokenProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	token, err := p.fetchToken(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	return token.Credentials, nil
}

func (p *dogeTmpTokenProvider) fetchToken(ctx context.Context) (*dogeTmpTokenResult, error) {
	body := `{"channel":"` + dogeTmpTokenChannel + `","scopes":["*"]}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+dogeTmpTokenPath, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	token := signDogeRequest(p.secretKey, dogeTmpTokenPath, body)
	req.Header.Set("Authorization", "TOKEN "+p.accessID+":"+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	creds, err := parseDogeTmpTokenResponse(raw, p.preferredBucket)
	if err != nil {
		return nil, err
	}

	return creds, nil
}

func signDogeRequest(secretKey, requestURI, body string) string {
	signStr := requestURI + "\n" + body
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(signStr))
	return hex.EncodeToString(mac.Sum(nil))
}

func parseDogeTmpTokenResponse(raw []byte, preferredBucket string) (*dogeTmpTokenResult, error) {
	var payload dogeTmpTokenResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse DogeCloud tmp token response: %w", err)
	}
	if payload.Code != 200 {
		return nil, fmt.Errorf("DogeCloud tmp token failed, code=%d msg=%s", payload.Code, payload.Msg)
	}

	accessKeyID := strings.TrimSpace(payload.Data.Credentials.AccessKeyID)
	secretAccessKey := strings.TrimSpace(payload.Data.Credentials.SecretAccessKey)
	sessionToken := strings.TrimSpace(payload.Data.Credentials.SessionToken)
	if accessKeyID == "" || secretAccessKey == "" || sessionToken == "" {
		return nil, fmt.Errorf("DogeCloud tmp token response missing credentials")
	}

	expireAt := time.Now().Add(dogeDefaultCredTTL)
	if parsed, ok := parseDogeExpiration(
		payload.Data.Credentials.ExpiredTime,
		payload.Data.Credentials.Expiration,
		payload.Data.Credentials.ExpiresAt,
		payload.Data.ExpiredTime,
		payload.Data.Expiration,
		payload.Data.ExpiresAt,
	); ok {
		expireAt = parsed
	}

	endpoint := firstNonEmpty(payload.Data.S3Endpoint, payload.Data.Endpoint)
	bucket := firstNonEmpty(payload.Data.S3Bucket, payload.Data.Bucket)
	selected, selectedByPreference := selectBucket(payload.Data.Buckets, preferredBucket)
	if selected != nil {
		if selectedByPreference {
			// Respect explicit bucket selection from config when available in API response.
			bucket = firstNonEmpty(selected.S3Bucket, bucket)
			endpoint = firstNonEmpty(selected.S3Endpoint, endpoint)
		} else {
			// Fill missing fields from first available bucket entry.
			bucket = firstNonEmpty(bucket, selected.S3Bucket)
			endpoint = firstNonEmpty(endpoint, selected.S3Endpoint)
		}
	}
	endpoint = normalizeEndpointURL(endpoint)
	region := firstNonEmpty(payload.Data.S3Region, payload.Data.Region)
	if region == "" && endpoint != "" {
		if parsed, ok := extractRegionFromEndpoint(endpoint); ok {
			region = parsed
		}
	}

	return &dogeTmpTokenResult{
		Credentials: aws.Credentials{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			SessionToken:    sessionToken,
			Source:          "dogecloud/tmp_token",
			CanExpire:       true,
			Expires:         expireAt,
		},
		S3Endpoint: endpoint,
		S3Bucket:   strings.TrimSpace(bucket),
		Region:     strings.TrimSpace(region),
	}, nil
}

type dogeBucketInfo struct {
	Name       string
	S3Bucket   string
	S3Endpoint string
}

func selectBucket(buckets []struct {
	Name           string `json:"name"`
	S3Bucket       string `json:"s3Bucket"`
	S3Endpoint     string `json:"s3Endpoint"`
	S3EndpointHost string `json:"s3EndpointHost"`
}, preferred string) (*dogeBucketInfo, bool) {
	if len(buckets) == 0 {
		return nil, false
	}
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		for _, b := range buckets {
			if strings.EqualFold(strings.TrimSpace(b.Name), preferred) || strings.EqualFold(strings.TrimSpace(b.S3Bucket), preferred) {
				return &dogeBucketInfo{
					Name:       strings.TrimSpace(b.Name),
					S3Bucket:   strings.TrimSpace(b.S3Bucket),
					S3Endpoint: strings.TrimSpace(b.S3Endpoint),
				}, true
			}
		}
	}
	b := buckets[0]
	return &dogeBucketInfo{
		Name:       strings.TrimSpace(b.Name),
		S3Bucket:   strings.TrimSpace(b.S3Bucket),
		S3Endpoint: strings.TrimSpace(b.S3Endpoint),
	}, false
}

func parseDogeExpiration(values ...any) (time.Time, bool) {
	for _, raw := range values {
		if raw == nil {
			continue
		}
		switch v := raw.(type) {
		case float64:
			if ts, ok := unixTimestampToTime(int64(v)); ok {
				return ts, true
			}
		case string:
			if t, ok := parseExpirationString(v); ok {
				return t, true
			}
		case json.Number:
			n, err := v.Int64()
			if err == nil {
				if ts, ok := unixTimestampToTime(n); ok {
					return ts, true
				}
			}
		}
	}
	return time.Time{}, false
}

func parseExpirationString(raw string) (time.Time, bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return time.Time{}, false
	}

	if unix, err := strconv.ParseInt(text, 10, 64); err == nil {
		return unixTimestampToTime(unix)
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func unixTimestampToTime(ts int64) (time.Time, bool) {
	if ts <= 0 {
		return time.Time{}, false
	}
	// Heuristic: values above this threshold are likely in milliseconds.
	if ts > 1_000_000_000_000 {
		return time.UnixMilli(ts), true
	}
	return time.Unix(ts, 0), true
}

func normalizeEndpointURL(endpoint string) string {
	ep := strings.TrimSpace(endpoint)
	if ep == "" {
		return ""
	}
	if !strings.Contains(ep, "://") {
		ep = "https://" + ep
	}
	return strings.TrimRight(ep, "/")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func extractRegionFromEndpoint(endpoint string) (string, bool) {
	host := strings.TrimSpace(endpoint)
	if host == "" {
		return "", false
	}

	if strings.Contains(host, "://") {
		if parsed, err := url.Parse(host); err == nil {
			host = parsed.Hostname()
		}
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return "", false
	}

	parts := strings.Split(host, ".")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "ap-") || strings.HasPrefix(part, "cn-") || strings.HasPrefix(part, "us-") || strings.HasPrefix(part, "eu-") {
			return part, true
		}
	}

	if len(parts) >= 2 && (parts[0] == "cos" || parts[0] == "s3") {
		return parts[1], true
	}
	return "", false
}

func shouldUsePathStyle(endpoint string) bool {
	host := strings.TrimSpace(endpoint)
	if host == "" {
		return true
	}
	if strings.Contains(host, "://") {
		if parsed, err := url.Parse(host); err == nil {
			host = parsed.Hostname()
		}
	}
	host = strings.ToLower(strings.TrimSpace(host))

	// Tencent COS / DogeCloud-backed COS endpoints require bucket in host.
	if strings.Contains(host, ".myqcloud.com") || strings.Contains(host, ".cos.") {
		return false
	}
	return true
}

// PutObject uploads an object to OSS.
func (c *OSSClient) PutObject(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}

	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	_, err := c.client.PutObject(ctx, input)
	return err
}

// GetObject retrieves an object from OSS.
func (c *OSSClient) GetObject(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", 0, err
	}

	contentType := ""
	if output.ContentType != nil {
		contentType = *output.ContentType
	}

	contentLength := int64(0)
	if output.ContentLength != nil {
		contentLength = *output.ContentLength
	}

	return output.Body, contentType, contentLength, nil
}

func (c *OSSClient) GetStyledObject(ctx context.Context, key, style string) (io.ReadCloser, string, int64, error) {
	style = strings.Trim(strings.TrimSpace(style), "/")
	if style == "" {
		return c.GetObject(ctx, key)
	}

	styledURL, err := c.buildStyledObjectURL(key, style)
	if err != nil {
		return nil, "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, styledURL, nil)
	if err != nil {
		return nil, "", 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		return nil, "", 0, fmt.Errorf("styled object fetch failed with status %d", resp.StatusCode)
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	contentLength := int64(0)
	if value := strings.TrimSpace(resp.Header.Get("Content-Length")); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			contentLength = parsed
		}
	}

	return resp.Body, contentType, contentLength, nil
}

func (c *OSSClient) buildStyledObjectURL(key, style string) (string, error) {
	base, err := c.buildObjectURL(key)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(base, "/") + "/" + style, nil
}

func (c *OSSClient) buildObjectURL(key string) (string, error) {
	if strings.TrimSpace(c.endpoint) == "" || strings.TrimSpace(c.bucket) == "" {
		return "", fmt.Errorf("oss endpoint or bucket is empty")
	}

	u, err := url.Parse(c.endpoint)
	if err != nil {
		return "", err
	}

	segments := strings.Split(strings.TrimLeft(key, "/"), "/")
	escaped := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(segment))
	}
	objectPath := strings.Join(escaped, "/")

	if c.usePathStyle {
		u.Path = "/" + strings.Trim(c.bucket, "/")
		if objectPath != "" {
			u.Path += "/" + objectPath
		}
		return u.String(), nil
	}

	u.Host = c.bucket + "." + u.Host
	u.Path = "/"
	if objectPath != "" {
		u.Path += objectPath
	}
	return u.String(), nil
}

// DeleteObject removes an object from OSS.
func (c *OSSClient) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

// CreateMultipartUpload initiates a multipart upload.
func (c *OSSClient) CreateMultipartUpload(ctx context.Context, key string, contentType string) (string, error) {
	output, err := c.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}
	return *output.UploadId, nil
}

// UploadPart uploads a single part of a multipart upload.
func (c *OSSClient) UploadPart(ctx context.Context, key, uploadID string, partNumber int32, body io.Reader, size int64) (string, error) {
	output, err := c.client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		UploadId:      aws.String(uploadID),
		PartNumber:    aws.Int32(partNumber),
		Body:          body,
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", err
	}
	return *output.ETag, nil
}

// CompleteMultipartUpload finalizes a multipart upload.
func (c *OSSClient) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []types.CompletedPart) error {
	_, err := c.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	return err
}

// AbortMultipartUpload cancels a multipart upload.
func (c *OSSClient) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	_, err := c.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	return err
}

// CopyObject copies an object within OSS.
func (c *OSSClient) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	_, err := c.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(c.bucket),
		CopySource: aws.String(c.bucket + "/" + srcKey),
		Key:        aws.String(dstKey),
	})
	return err
}
