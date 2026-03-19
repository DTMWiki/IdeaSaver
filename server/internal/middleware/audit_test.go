package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInferAction(t *testing.T) {
	t.Setenv("GIN_MODE", "test")
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name        string
		method      string
		route       string
		requestPath string
		want        string
	}{
		{
			name:        "create directory",
			method:      "POST",
			route:       "/api/files/mkdir",
			requestPath: "/api/files/mkdir",
			want:        "create_directory",
		},
		{
			name:        "share delete",
			method:      "DELETE",
			route:       "/api/shares/:id",
			requestPath: "/api/shares/abc",
			want:        "share_delete",
		},
		{
			name:        "video delete handled in service",
			method:      "DELETE",
			route:       "/api/videos/:id",
			requestPath: "/api/videos/abc",
			want:        "",
		},
		{
			name:        "upload chunk ignored",
			method:      "POST",
			route:       "/api/upload/:task_id/chunk",
			requestPath: "/api/upload/abc/chunk",
			want:        "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var got string
			r := gin.New()
			r.Handle(tc.method, tc.route, func(c *gin.Context) {
				got = inferAction(c)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.requestPath, nil)
			r.ServeHTTP(w, req)

			if got != tc.want {
				t.Fatalf("inferAction() = %q, want %q", got, tc.want)
			}
		})
	}
}
