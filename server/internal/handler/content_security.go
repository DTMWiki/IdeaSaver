package handler

import (
	"path/filepath"
	"strings"
)

// dangerousContentTypes must never be served inline on the app origin.
var dangerousContentTypes = map[string]bool{
	"text/html":              true,
	"application/xhtml+xml":  true,
	"image/svg+xml":          true,
	"text/javascript":        true,
	"application/javascript": true,
	"application/x-javascript": true,
	"text/css":               true,
	"text/xml":               true,
	"application/xml":        true,
	"application/xhtml":      true,
}

// safeServeContentType returns a content type safe for same-origin delivery.
// Executable or scriptable types are forced to application/octet-stream.
func safeServeContentType(contentType, filename string) string {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if ct == "" {
		ct = strings.ToLower(contentTypeFromFilename(filename))
	}
	if dangerousContentTypes[ct] {
		return "application/octet-stream"
	}
	// Also catch typedown by extension if storage type lied.
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".html", ".htm", ".svg", ".js", ".mjs", ".css", ".xml", ".xhtml":
		return "application/octet-stream"
	}
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}

func contentTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".txt", ".md":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".zip":
		return "application/zip"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

// forceAttachment returns true when the response must download, not render inline.
func forceAttachment(contentType, filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".html", ".htm", ".svg", ".js", ".mjs", ".css", ".xml", ".xhtml":
		return true
	}
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return dangerousContentTypes[ct]
}
