package service

import (
	"testing"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
)

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

func TestMergeVideoStatusDoesNotAutoReadyWithoutCallback(t *testing.T) {
	t.Parallel()

	got := mergeVideoStatus(videoTranscodeProcessing, 10)
	if got != videoTranscodeProcessing {
		t.Fatalf("mergeVideoStatus() = %q, want %q", got, videoTranscodeProcessing)
	}
}

func TestPlaybackMessageForReadyWithoutPlayableSource(t *testing.T) {
	t.Parallel()

	message := playbackMessage(&model.Video{
		TranscodeStatus:  videoTranscodeReady,
		TranscodeMessage: "",
	})
	if message != "已收到转码成功回调，正在同步播放信息" {
		t.Fatalf("playbackMessage() = %q", message)
	}
}

func TestCanPlayVideoRequiresSDKMetadata(t *testing.T) {
	t.Parallel()

	if canPlayVideo(&model.Video{PlayURL: "https://example.com/raw.mp4"}) {
		t.Fatalf("canPlayVideo() should reject direct-url-only playback")
	}

	if !canPlayVideo(&model.Video{VCode: "abc", PlayerUserID: "123"}) {
		t.Fatalf("canPlayVideo() should accept vcode + sdk user id")
	}
}
