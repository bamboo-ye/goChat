package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExtractToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	request := httptest.NewRequest("GET", "/chat?token=query-token", nil)
	request.Header.Set("Authorization", "Bearer header-token")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	if token := ExtractToken(context); token != "header-token" {
		t.Fatalf("expected header token, got %q", token)
	}
}
