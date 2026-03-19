package storage

import "testing"

func TestExtractPlayURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data map[string]any
		want string
	}{
		{
			name: "top-level play_url",
			data: map[string]any{
				"play_url": "https://example.com/a.m3u8",
			},
			want: "https://example.com/a.m3u8",
		},
		{
			name: "streams fallback",
			data: map[string]any{
				"streams": []any{
					map[string]any{"url": "https://example.com/b.m3u8"},
				},
			},
			want: "https://example.com/b.m3u8",
		},
		{
			name: "stream fallback",
			data: map[string]any{
				"stream": []any{
					map[string]any{"url": "//example.com/c.m3u8"},
				},
			},
			want: "https://example.com/c.m3u8",
		},
		{
			name: "backup url fallback",
			data: map[string]any{
				"stream": []any{
					map[string]any{"backup_urls": []any{"https://example.com/d.m3u8"}},
				},
			},
			want: "https://example.com/d.m3u8",
		},
		{
			name: "missing",
			data: map[string]any{
				"streams": []any{},
			},
			want: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := extractPlayURL(tc.data)
			if got != tc.want {
				t.Fatalf("extractPlayURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeRemoteURL(t *testing.T) {
	t.Parallel()

	if got := normalizeRemoteURL("//cdn.example.com/thumb.jpg"); got != "https://cdn.example.com/thumb.jpg" {
		t.Fatalf("normalizeRemoteURL() = %q", got)
	}
	if got := normalizeRemoteURL("https://cdn.example.com/thumb.jpg"); got != "https://cdn.example.com/thumb.jpg" {
		t.Fatalf("normalizeRemoteURL() preserved = %q", got)
	}
}

func TestPlayerUserIDFromURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "userId param",
			raw:  "https://example.com/player/get.mp4?vcode=abc&userId=12345",
			want: "12345",
		},
		{
			name: "uid param",
			raw:  "https://example.com/hls.m3u8?uid=987&token=abc",
			want: "987",
		},
		{
			name: "invalid url",
			raw:  ":::",
			want: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := playerUserIDFromURL(tc.raw)
			if got != tc.want {
				t.Fatalf("playerUserIDFromURL(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
