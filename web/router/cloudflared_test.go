package router

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestCloudflaredRoutesRequireAdministrator(t *testing.T) {
	r := gin.New()
	registerAdminRoutes(r)
	for _, route := range []struct{ method, path string }{
		{"GET", "/api/admin/settings/cloudflared"},
		{"GET", "/api/admin/client/node/traffic"},
		{"POST", "/api/admin/client/node/traffic/reset"},
		{"POST", "/api/admin/client/node/traffic/set"},
		{"POST", "/api/admin/settings/cloudflared/token"},
		{"POST", "/api/admin/settings/cloudflared/start"},
		{"POST", "/api/admin/settings/cloudflared/stop"},
		{"POST", "/api/admin/settings/cloudflared/remove-token"},
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(route.method, route.path, nil))
		if rec.Code != 401 {
			t.Errorf("%s %s: got %d, want 401", route.method, route.path, rec.Code)
		}
	}
}
