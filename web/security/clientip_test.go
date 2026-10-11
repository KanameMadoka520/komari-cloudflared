package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRealClientIPUsesCloudflareHeaderFromLoopback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("CF-Connecting-IP", "203.0.113.10")
	req.Header.Set("X-Forwarded-For", "198.51.100.8")
	if got := realClientIP(req); got != "203.0.113.10" {
		t.Fatalf("got %q", got)
	}
}

func TestRealClientIPIgnoresSpoofedHeaderFromPublicPeer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("CF-Connecting-IP", "198.51.100.8")
	if got := realClientIP(req); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestRealClientIPUsesProxyObservedAddress(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.23.0.3:1234"
	req.Header.Set("X-Real-IP", "198.51.100.20")
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 198.51.100.20")
	if got := realClientIP(req); got != "198.51.100.20" {
		t.Fatalf("got %q", got)
	}
}

func TestGinClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	UseRealClientIP(r)
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:4000"
	req.Header.Set("CF-Connecting-IP", "203.0.113.10")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Body.String() != "203.0.113.10" {
		t.Fatalf("body %q", rec.Body.String())
	}
}
