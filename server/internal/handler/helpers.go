package handler

import (
	"fmt"
	"strings"
	"time"
)

func isImageMime(mime string) bool {
	return len(mime) > 6 && mime[:6] == "image/"
}

func parseAuditLogTime(raw string, inclusiveEnd bool) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err != nil {
			continue
		}
		if inclusiveEnd && layout == "2006-01-02" {
			parsed = parsed.Add(24*time.Hour - time.Nanosecond)
		}
		return &parsed, nil
	}

	return nil, fmt.Errorf("invalid time format")
}
