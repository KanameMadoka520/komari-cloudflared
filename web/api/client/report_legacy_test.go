package client

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegacyIngestionRejectsBodyOnlyIdentity(t *testing.T) {
	for _, handler := range []gin.HandlerFunc{UploadReport, UploadBasicInfo} {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"uuid":"victim","cpu":{"usage":1}}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		handler(ctx)
		if rec.Code != 401 {
			t.Fatalf("body UUID granted client identity: status=%d", rec.Code)
		}
	}
}
