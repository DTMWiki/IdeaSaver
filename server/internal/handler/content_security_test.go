package handler

import "testing"

func TestSafeServeContentTypeBlocksExecutable(t *testing.T) {
	cases := []struct {
		ct, name, want string
	}{
		{"text/html", "x.html", "application/octet-stream"},
		{"image/svg+xml", "x.svg", "application/octet-stream"},
		{"text/javascript", "x.js", "application/octet-stream"},
		{"image/png", "x.png", "image/png"},
		{"", "note.md", "text/plain"},
	}
	for _, tc := range cases {
		got := safeServeContentType(tc.ct, tc.name)
		if got != tc.want {
			t.Fatalf("%s/%s: got %q want %q", tc.ct, tc.name, got, tc.want)
		}
	}
}

func TestForceAttachment(t *testing.T) {
	if !forceAttachment("text/html", "a.html") {
		t.Fatal("html should force attachment")
	}
	if forceAttachment("image/png", "a.png") {
		t.Fatal("png should allow inline")
	}
}
