package service

import "testing"

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
		{
			name: "missing param",
			raw:  "https://example.com/hls.m3u8?token=abc",
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
