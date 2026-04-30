package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

func TestEnrichContextMalformedSessionCookieReturnsGenericError(t *testing.T) {
	InitStore([]byte("thisisasessionkeyreplacethisjetzt"), []byte("thisisasessionencryptionkey12345"), "cookie", 8192)

	logs := captureTestLogs(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: rdpGwSession, Value: "not-a-valid-signed-session"})
	rec := httptest.NewRecorder()

	EnrichContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run when session decoding fails")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if rec.Body.String() != "authentication failed\n" {
		t.Fatalf("expected generic authentication failure, got %q", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "securecookie") || strings.Contains(rec.Body.String(), "not valid") {
		t.Fatalf("response body leaked session error: %q", rec.Body.String())
	}
	if !strings.Contains(logs.String(), "securecookie") {
		t.Fatalf("expected detailed session error in server logs, got %q", logs.String())
	}
}

func TestEnrichContextRejectsPersistedHeaderSessionFromUntrustedPeer(t *testing.T) {
	InitStore([]byte("thisisasessionkeyreplacethisjetzt"), []byte("thisisasessionencryptionkey12345"), "cookie", 8192)

	saveReq := httptest.NewRequest(http.MethodGet, "/", nil)
	saveRec := httptest.NewRecorder()
	id := identity.NewUser()
	id.SetUserName("header-user")
	id.SetAuthenticated(true)
	id.SetAttribute(identity.AttrAuthSource, identity.AuthSourceHeader)
	if err := SaveSessionIdentity(saveReq, saveRec, id); err != nil {
		t.Fatalf("save header session: %v", err)
	}

	handler, err := EnrichContextWithTrustedProxyCIDRs([]string{"198.51.100.0/24"})
	if err != nil {
		t.Fatalf("trusted proxy middleware: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/tokeninfo", nil)
	req.RemoteAddr = "203.0.113.10:54321"
	for _, cookie := range saveRec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run for persisted header session from untrusted peer")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestEnrichContextIgnoresSpoofedXForwardedForFromDirectClient(t *testing.T) {
	handler, err := EnrichContextWithTrustedProxyCIDRs([]string{"10.0.0.0/24"})
	if err != nil {
		t.Fatalf("trusted proxy middleware: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:54321"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")

	var got interface{}
	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = identity.FromRequestCtx(r).GetAttribute(identity.AttrClientIp)
	})).ServeHTTP(httptest.NewRecorder(), req)

	if got != "203.0.113.10" {
		t.Fatalf("expected direct remote IP, got %v", got)
	}
}

func TestEnrichContextAcceptsXForwardedForFromTrustedProxy(t *testing.T) {
	handler, err := EnrichContextWithTrustedProxyCIDRs([]string{"10.0.0.0/24"})
	if err != nil {
		t.Fatalf("trusted proxy middleware: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.12:54321"
	req.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.12")

	var got interface{}
	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = identity.FromRequestCtx(r).GetAttribute(identity.AttrClientIp)
	})).ServeHTTP(httptest.NewRecorder(), req)

	if got != "198.51.100.9" {
		t.Fatalf("expected forwarded client IP, got %v", got)
	}
}

func TestEnrichContextFallsBackForMalformedXForwardedFor(t *testing.T) {
	handler, err := EnrichContextWithTrustedProxyCIDRs([]string{"10.0.0.0/24"})
	if err != nil {
		t.Fatalf("trusted proxy middleware: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.12:54321"
	req.Header.Set("X-Forwarded-For", "not-an-ip")

	var got interface{}
	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = identity.FromRequestCtx(r).GetAttribute(identity.AttrClientIp)
	})).ServeHTTP(httptest.NewRecorder(), req)

	if got != "10.0.0.12" {
		t.Fatalf("expected trusted proxy remote IP fallback, got %v", got)
	}
}

func TestEnrichContextFallsBackForMixedMalformedXForwardedFor(t *testing.T) {
	handler, err := EnrichContextWithTrustedProxyCIDRs([]string{"10.0.0.0/24"})
	if err != nil {
		t.Fatalf("trusted proxy middleware: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.12:54321"
	req.Header.Set("X-Forwarded-For", "not-an-ip, 198.51.100.9, 198.51.100.10")

	var got interface{}
	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = identity.FromRequestCtx(r).GetAttribute(identity.AttrClientIp)
	})).ServeHTTP(httptest.NewRecorder(), req)

	if got != "10.0.0.12" {
		t.Fatalf("expected trusted proxy remote IP fallback, got %v", got)
	}
}

func TestEnrichContextFallsBackForEmptyXForwardedForEntry(t *testing.T) {
	handler, err := EnrichContextWithTrustedProxyCIDRs([]string{"10.0.0.0/24"})
	if err != nil {
		t.Fatalf("trusted proxy middleware: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.12:54321"
	req.Header.Set("X-Forwarded-For", "198.51.100.9, , 198.51.100.10")

	var got interface{}
	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = identity.FromRequestCtx(r).GetAttribute(identity.AttrClientIp)
	})).ServeHTTP(httptest.NewRecorder(), req)

	if got != "10.0.0.12" {
		t.Fatalf("expected trusted proxy remote IP fallback, got %v", got)
	}
}
