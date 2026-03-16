package storage

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
)

// VCloudClient wraps DogeCloud Video Cloud REST API.
type VCloudClient struct {
	apiBase    string
	accessKey  string
	secretKey  string
	httpClient *http.Client
}

// NewVCloudClient creates a new DogeCloud VCloud API client.
func NewVCloudClient(cfg *config.Config) *VCloudClient {
	return &VCloudClient{
		apiBase:    cfg.DogeVCloudAPI,
		accessKey:  cfg.DogeAccessKey,
		secretKey:  cfg.DogeSecretKey,
		httpClient: &http.Client{},
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
// It first gets temporary credentials, then uploads using multipart.
func (c *VCloudClient) UploadVideo(title string, fileReader io.Reader, filename string, fileSize int64, callbackString string) (string, error) {
	// Step 1: Get temporary upload credentials
	bodyJSON, _ := json.Marshal(map[string]any{
		"channel": "VOD_UPLOAD",
		"scopes":  []string{"*"},
	})

	result, err := c.doRequest("POST", "/auth/tmp_token.json", string(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("failed to get temp token: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	credentials, ok := data["Credentials"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing credentials in response")
	}

	accessKeyID := credentials["accessKeyId"].(string)
	secretAccessKey := credentials["secretAccessKey"].(string)
	sessionToken := credentials["sessionToken"].(string)

	// Step 2: Upload the video using DogeCloud upload API
	// For server-side uploads, we use the direct upload endpoint
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("accessKeyId", accessKeyID)
	_ = writer.WriteField("secretAccessKey", secretAccessKey)
	_ = writer.WriteField("sessionToken", sessionToken)
	_ = writer.WriteField("callbackString", callbackString)
	_ = writer.WriteField("vn", title)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, fileReader); err != nil {
		return "", err
	}
	writer.Close()

	uploadURL := "https://vod-api.dogecloud.com/upload/put.json"
	req, err := http.NewRequest("POST", uploadURL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var uploadResult map[string]any
	if err := json.Unmarshal(respBody, &uploadResult); err != nil {
		return "", fmt.Errorf("failed to parse upload response: %w", err)
	}

	if code, ok := uploadResult["code"].(float64); ok && code != 200 {
		return "", fmt.Errorf("upload failed with code %d", int(code))
	}

	// Extract video ID
	uploadData, ok := uploadResult["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected upload response format")
	}

	vid, ok := uploadData["vid"].(string)
	if !ok {
		// Try float64 format
		if vidFloat, ok := uploadData["vid"].(float64); ok {
			vid = fmt.Sprintf("%.0f", vidFloat)
		} else {
			return "", fmt.Errorf("missing vid in upload response")
		}
	}

	return vid, nil
}

// GetVideoStreams gets the video playback URLs.
func (c *VCloudClient) GetVideoStreams(vcode string) (map[string]any, error) {
	path := fmt.Sprintf("/video/streams.json?vcode=%s&platform=pch5", vcode)
	return c.doRequest("GET", path, "")
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
