package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/security"
)

func TestTokenInfoReturnsGenericInvalidTokenError(t *testing.T) {
	oldEncryptionKey := security.UserEncryptionKey
	oldSigningKey := security.UserSigningKey
	security.UserEncryptionKey = []byte("12345678901234567890123456789012")
	security.UserSigningKey = nil
	t.Cleanup(func() {
		security.UserEncryptionKey = oldEncryptionKey
		security.UserSigningKey = oldSigningKey
	})

	logs := captureTestLogs(t)
	req := httptest.NewRequest(http.MethodGet, "/tokeninfo?access_token=not-a-jwe", nil)
	rec := httptest.NewRecorder()

	TokenInfo(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if rec.Body.String() != "invalid token\n" {
		t.Fatalf("expected generic invalid token response, got %q", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "cannot get token") || strings.Contains(rec.Body.String(), "token validation failed") {
		t.Fatalf("response body leaked token validation error: %q", rec.Body.String())
	}
	if !strings.Contains(logs.String(), "Cannot get token") {
		t.Fatalf("expected detailed token error in server logs, got %q", logs.String())
	}
}
