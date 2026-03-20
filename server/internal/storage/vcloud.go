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
	"strings"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// VCloudClient wraps DogeCloud Video Cloud REST API.
type VCloudClient struct {
	apiBase    string
	accessKey  string
	secretKey  string
	httpClient *http.Client
}

type vodUploadInfo struct {
	DID        string `json:"did"`
	Key        string `json:"key"`
	S3Bucket   string `json:"s3Bucket"`
	S3Endpoint string `json:"s3Endpoint"`
}

type VideoInfo struct {
	VCode             string
	PlayerUserID      string
	PlayURL           string
	ThumbnailURL      string
	ThumbnailSmallURL string
	PlayCount         int64
	Status            int
}

// NewVCloudClient creates a new DogeCloud VCloud API client.
func NewVCloudClient(cfg *config.Config) *VCloudClient {
	apiBase := cfg.DogeVCloudAPI
	if strings.TrimSpace(apiBase) == "" {
		apiBase = dogeDefaultAPIServer
	}
	return &VCloudClient{
		apiBase:    strings.TrimRight(apiBase, "/"),
		accessKey:  cfg.DogeAccessKey,
		secretKey:  cfg.DogeSecretKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// sign generates the DogeCloud API signature.
func (c *VCloudClient) sign(requestURI, body string) string {
	signStr := requestURI + "\n" + body
	mac := hmac.New(sha1.New, []byte(c.secretKey))
	mac.Write([]byte(signStr))
	signature := hex.EncodeToString(mac.Sum(nil))
	return c.accessKey + ":" + signature
}

// doRequest performs a signed API request to DogeCloud.
func (c *VCloudClient) doRequest(method, path string, body string) (map[string]any, error) {
	url := c.apiBase + path

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	token := c.sign(path, body)
	req.Header.Set("Authorization", "TOKEN "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(respBody))
	}

	if code, ok := result["code"].(float64); ok && code != 200 {
		msg := ""
		if m, ok := result["msg"].(string); ok {
			msg = m
		}
		return nil, fmt.Errorf("DogeCloud API error %d: %s", int(code), msg)
	}

	return result, nil
}

// UploadVideo uploads a video file to DogeCloud VCloud.
// It first gets temporary credentials + VodUploadInfo, uploads via S3, then reports completion.
func (c *VCloudClient) UploadVideo(title string, fileReader io.Reader, filename string, fileSize int64, callbackString string) (string, error) {
	// Step 1: Get temporary upload credentials and VOD upload info.
	bodyJSON, _ := json.Marshal(map[string]any{
		"channel": "VOD_UPLOAD",
		"vodConfig": map[string]any{
			"filename":       filename,
			"vn":             title,
			"callbackString": callbackString,
		},
	})

	result, err := c.doRequest("POST", "/auth/tmp_token.json", string(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("failed to get temp token: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	credMap, ok := data["Credentials"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing credentials in response")
	}

	accessKeyID, _ := credMap["accessKeyId"].(string)
	secretAccessKey, _ := credMap["secretAccessKey"].(string)
	sessionToken, _ := credMap["sessionToken"].(string)
	if accessKeyID == "" || secretAccessKey == "" || sessionToken == "" {
		return "", fmt.Errorf("missing temporary credentials fields")
	}

	vodInfoRaw, ok := data["VodUploadInfo"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing VodUploadInfo in response")
	}
	vodInfo := vodUploadInfo{
		DID:        asString(vodInfoRaw["did"]),
		Key:        asString(vodInfoRaw["key"]),
		S3Bucket:   asString(vodInfoRaw["s3Bucket"]),
		S3Endpoint: normalizeEndpointURL(asString(vodInfoRaw["s3Endpoint"])),
	}
	if vodInfo.DID == "" || vodInfo.Key == "" || vodInfo.S3Bucket == "" || vodInfo.S3Endpoint == "" {
		return "", fmt.Errorf("incomplete VodUploadInfo in response")
	}

	// Step 2: Upload video content to S3 endpoint with temporary credentials.
	region := "automatic"
	if parsed, ok := extractRegionFromEndpoint(vodInfo.S3Endpoint); ok {
		region = parsed
	}

	resolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...any) (aws.Endpoint, error) {
			return aws.Endpoint{URL: vodInfo.S3Endpoint}, nil
		},
	)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithEndpointResolverWithOptions(resolver),
		awsconfig.WithCredentialsProvider(
			awscreds.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken),
		),
	)
	if err != nil {
		return "", fmt.Errorf("failed to init S3 config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = shouldUsePathStyle(vodInfo.S3Endpoint)
	})

	putInput := &s3.PutObjectInput{
		Bucket: aws.String(vodInfo.S3Bucket),
		Key:    aws.String(vodInfo.Key),
		Body:   fileReader,
	}
	if fileSize > 0 {
		putInput.ContentLength = aws.Int64(fileSize)
	}

	if _, err := s3Client.PutObject(context.Background(), putInput); err != nil {
		return "", fmt.Errorf("failed to upload video to S3: %w", err)
	}

	// Step 3: Report upload completion and get video ID.
	vid, err := c.completeUpload(vodInfo.DID)
	if err != nil {
		return "", err
	}
	return vid, nil
}

// GetVideoStreams gets the video playback URLs.
func (c *VCloudClient) GetVideoStreams(vcode, clientIP, userAgent string) (map[string]any, error) {
	params := url.Values{}
	params.Set("vcode", strings.TrimSpace(vcode))
	params.Set("platform", "pch5")
	if ip := strings.TrimSpace(clientIP); ip != "" {
		params.Set("ip", ip)
	}
	if ua := strings.TrimSpace(userAgent); ua != "" {
		params.Set("ua", ua)
	}
	path := "/video/streams.json?" + params.Encode()
	return c.doRequest("GET", path, "")
}

// GetBestPlayURL tries to extract a direct playable URL from streams response.
func (c *VCloudClient) GetBestPlayURL(vcode, clientIP, userAgent string) (string, error) {
	streams, err := c.GetVideoStreams(vcode, clientIP, userAgent)
	if err != nil {
		return "", err
	}

	data, ok := streams["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("video/streams missing data")
	}

	return extractPlayURL(data), nil
}

// GetVideoInfo fetches video metadata such as vcode and player userId by vid.
func (c *VCloudClient) GetVideoInfo(vid string) (*VideoInfo, error) {
	path := fmt.Sprintf("/video/info.json?vid=%s", url.QueryEscape(strings.TrimSpace(vid)))
	result, err := c.doRequest("GET", path, "")
	if err != nil {
		return nil, err
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("video/info missing data")
	}

	info := &VideoInfo{
		VCode:             asString(data["vcode"]),
		PlayURL:           normalizeRemoteURL(extractPlayURL(data)),
		ThumbnailURL:      normalizeRemoteURL(asString(data["thumbnail"])),
		ThumbnailSmallURL: normalizeRemoteURL(asString(data["thumbnail_small"])),
		PlayCount:         extractPlayCount(data),
		Status:            asInt(data["status"]),
	}
	info.PlayerUserID = firstNonEmptyString(
		asString(data["userId"]),
		asString(data["userid"]),
		asString(data["uid"]),
		playerUserIDFromURL(info.PlayURL),
	)
	return info, nil
}

// ListVideos lists videos from DogeCloud.
func (c *VCloudClient) ListVideos(start, count int) (map[string]any, error) {
	path := fmt.Sprintf("/vod/video/list.json?start=%d&count=%d", start, count)
	return c.doRequest("GET", path, "")
}

// SetVideoStatus enables or disables videos.
func (c *VCloudClient) SetVideoStatus(vids []string, status int) error {
	bodyJSON, _ := json.Marshal(map[string]any{
		"vids":   vids,
		"status": status,
	})
	_, err := c.doRequest("POST", "/vod/video/status.json", string(bodyJSON))
	return err
}

// DeleteVideos deletes videos from DogeCloud.
func (c *VCloudClient) DeleteVideos(vids []string) error {
	bodyJSON, _ := json.Marshal(map[string]any{
		"vids": vids,
	})
	_, err := c.doRequest("POST", "/vod/video/delete.json", string(bodyJSON))
	return err
}

func (c *VCloudClient) completeUpload(did string) (string, error) {
	if strings.TrimSpace(did) == "" {
		return "", fmt.Errorf("missing did")
	}

	u := c.apiBase + "/callback/upload.json?did=" + url.QueryEscape(did)
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return "", fmt.Errorf("failed to report upload callback: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("failed to parse callback response: %w (body: %s)", err, string(raw))
	}
	if code, ok := payload["code"].(float64); ok && int(code) != 200 {
		return "", fmt.Errorf("callback/upload failed: %v", payload["msg"])
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("callback/upload missing data")
	}
	vid := asString(data["vid"])
	if vid == "" {
		return "", fmt.Errorf("callback/upload missing vid")
	}
	return vid, nil
}

func (c *VCloudClient) BuildPlayerMP4URL(vcode, playerUserID string) string {
	vcode = strings.TrimSpace(vcode)
	playerUserID = strings.TrimSpace(playerUserID)
	if vcode == "" || playerUserID == "" {
		return ""
	}
	base := strings.TrimRight(c.apiBase, "/")
	return fmt.Sprintf("%s/player/get.mp4?vcode=%s&userId=%s", base, url.QueryEscape(vcode), url.QueryEscape(playerUserID))
}

func extractPlayURL(data map[string]any) string {
	if data == nil {
		return ""
	}

	for _, key := range []string{"play_url", "playUrl", "url"} {
		if v := strings.TrimSpace(asString(data[key])); v != "" {
			return normalizeRemoteURL(v)
		}
	}

	for _, groupKey := range []string{"stream", "streams"} {
		if streams, ok := data[groupKey].([]any); ok {
			for _, item := range streams {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				for _, key := range []string{"play_url", "playUrl", "url"} {
					if v := strings.TrimSpace(asString(m[key])); v != "" {
						return normalizeRemoteURL(v)
					}
				}
				if backups, ok := m["backup_urls"].([]any); ok {
					for _, backup := range backups {
						if v := strings.TrimSpace(asString(backup)); v != "" {
							return normalizeRemoteURL(v)
						}
					}
				}
			}
		}
	}

	return ""
}

func playerUserIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	q := u.Query()
	return firstNonEmptyString(q.Get("userId"), q.Get("userid"), q.Get("uid"))
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return fmt.Sprintf("%.0f", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case int:
		return fmt.Sprintf("%d", t)
	default:
		return ""
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return 0
		}
		var out int
		_, _ = fmt.Sscanf(t, "%d", &out)
		return out
	default:
		return 0
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return 0
		}
		var out int64
		_, _ = fmt.Sscanf(t, "%d", &out)
		return out
	default:
		return 0
	}
}

func extractPlayCount(data map[string]any) int64 {
	if data == nil {
		return 0
	}

	for _, key := range []string{"play_count", "playCount", "view_count", "viewCount", "pv", "plays"} {
		if value, ok := data[key]; ok {
			if count := asInt64(value); count > 0 {
				return count
			}
		}
	}

	if stats, ok := data["stats"].(map[string]any); ok {
		for _, key := range []string{"play_count", "playCount", "view_count", "viewCount", "pv", "plays"} {
			if value, ok := stats[key]; ok {
				if count := asInt64(value); count > 0 {
					return count
				}
			}
		}
	}

	return 0
}

func normalizeRemoteURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "//") {
		return "https:" + value
	}
	return value
}
