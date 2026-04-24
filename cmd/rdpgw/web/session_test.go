package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

func TestSessionCookieSecureFlagDefaultsFalse(t *testing.T) {
	sessionKey := []byte("testsessionkeytestsessionkey1234")
	encryptionKey := []byte("testencryptionkeytestencrypt1234")
	InitStore(sessionKey, encryptionKey, "cookie", 8192, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	id := identity.NewUser()
	id.SetUserName("user@example.com")
	id.SetAuthenticated(true)

	if err := SaveSessionIdentity(req, rec, id); err != nil {
		t.Fatalf("save session identity: %v", err)
	}

	cookie := findCookie(t, rec.Result().Cookies(), rdpGwSession)
	if cookie.Secure {
		t.Fatal("expected default session cookie to omit Secure")
	}
}

func TestSessionCookieSecureFlagCanBeForcedForTLSTermination(t *testing.T) {
	sessionKey := []byte("testsessionkeytestsessionkey1234")
	encryptionKey := []byte("testencryptionkeytestencrypt1234")
	InitStore(sessionKey, encryptionKey, "cookie", 8192, true)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	id := identity.NewUser()
	id.SetUserName("user@example.com")
	id.SetAuthenticated(true)

	if err := SaveSessionIdentity(req, rec, id); err != nil {
		t.Fatalf("save session identity: %v", err)
	}

	cookie := findCookie(t, rec.Result().Cookies(), rdpGwSession)
	if !cookie.Secure {
		t.Fatal("expected SecureCookies session cookie to set Secure")
	}
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s not found in %v", name, cookies)
	return nil
}
