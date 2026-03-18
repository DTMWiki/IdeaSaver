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
	client *s3.Client
	bucket string
}

type dogeTmpTokenProvider struct {
	apiBase   string
	accessID  string
	secretKey string
	client    *http.Client
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
		ExpiredTime any `json:"expiredTime"`
		Expiration  any `json:"expiration"`
		ExpiresAt   any `json:"expiresAt"`
	} `json:"data"`
}

// NewOSSClient creates a new DogeCloud OSS client using S3-compatible API.
func NewOSSClient(cfg *config.Config) (*OSSClient, error) {
	if cfg.DogeAccessKey == "" || cfg.DogeSecretKey == "" {
		return nil, fmt.Errorf("DogeCloud access credentials are required")
	}
	if cfg.DogeEndpoint == "" {
		return nil, fmt.Errorf("IDEASAVER_DOGE_ENDPOINT is required")
	}
	if cfg.DogeBucket == "" {
		return nil, fmt.Errorf("IDEASAVER_DOGE_BUCKET is required")
	}

	region := cfg.DogeRegion
	if region == "" {
		region = dogeDefaultRegion
	}

	apiBase := cfg.DogeVCloudAPI
	if apiBase == "" {
		apiBase = dogeDefaultAPIServer
	}

	resolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...any) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: cfg.DogeEndpoint,
			}, nil
		},
	)

	provider := &dogeTmpTokenProvider{
		apiBase:   strings.TrimRight(apiBase, "/"),
		accessID:  cfg.DogeAccessKey,
		secretKey: cfg.DogeSecretKey,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
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

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &OSSClient{
		client: client,
		bucket: cfg.DogeBucket,
	}, nil
}

func (p *dogeTmpTokenProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	body := `{"channel":"` + dogeTmpTokenChannel + `","scopes":["*"]}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+dogeTmpTokenPath, strings.NewReader(body))
	if err != nil {
		return aws.Credentials{}, err
	}

	token := signDogeRequest(p.secretKey, dogeTmpTokenPath, body)
	req.Header.Set("Authorization", "TOKEN "+p.accessID+":"+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return aws.Credentials{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return aws.Credentials{}, err
	}

	creds, err := parseDogeTmpTokenResponse(raw)
	if err != nil {
		return aws.Credentials{}, err
	}

	return creds, nil
}

func signDogeRequest(secretKey, requestURI, body string) string {
	signStr := requestURI + "\n" + body
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(signStr))
	return hex.EncodeToString(mac.Sum(nil))
}

func parseDogeTmpTokenResponse(raw []byte) (aws.Credentials, error) {
	var payload dogeTmpTokenResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to parse DogeCloud tmp token response: %w", err)
	}
	if payload.Code != 200 {
		return aws.Credentials{}, fmt.Errorf("DogeCloud tmp token failed, code=%d msg=%s", payload.Code, payload.Msg)
	}

	accessKeyID := strings.TrimSpace(payload.Data.Credentials.AccessKeyID)
	secretAccessKey := strings.TrimSpace(payload.Data.Credentials.SecretAccessKey)
	sessionToken := strings.TrimSpace(payload.Data.Credentials.SessionToken)
	if accessKeyID == "" || secretAccessKey == "" || sessionToken == "" {
		return aws.Credentials{}, fmt.Errorf("DogeCloud tmp token response missing credentials")
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

	return aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
		Source:          "dogecloud/tmp_token",
		CanExpire:       true,
		Expires:         expireAt,
	}, nil
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
