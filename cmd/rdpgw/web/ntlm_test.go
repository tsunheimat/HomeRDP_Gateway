package web

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestNTLMGetAuthPayloadRejectsShortHeadersWithoutPanic(t *testing.T) {
	handler := &NTLMAuthHandler{}
	cases := []struct {
		name   string
		header string
	}{
		{name: "missing", header: ""},
		{name: "single character", header: "N"},
		{name: "short ntlm scheme", header: "NTLM"},
		{name: "empty ntlm payload", header: "NTLM "},
		{name: "whitespace-only ntlm payload", header: "NTLM  "},
		{name: "malformed ntlm payload", header: "NTLM X"},
		{name: "wrong ntlm delimiter", header: "NTLMx"},
		{name: "short negotiate scheme", header: "Negotiate"},
		{name: "empty negotiate payload", header: "Negotiate "},
		{name: "whitespace-only negotiate payload", header: "Negotiate  "},
		{name: "malformed negotiate payload", header: "Negotiate X"},
		{name: "partial negotiate scheme", header: "Neg"},
		{name: "unsupported scheme", header: "Basic abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			payload, mode, err := handler.getAuthPayload(req)
			if err == nil {
				t.Fatalf("expected error for malformed header %q", tc.header)
			}
			if mode != authNone {
				t.Fatalf("expected authNone, got %v", mode)
			}
			if payload != "" {
				t.Fatalf("expected empty payload, got %q", payload)
			}
		})
	}
}

func TestNTLMSessionCookieSecureFlag(t *testing.T) {
	cases := []struct {
		name           string
		secureConfig   bool
		tlsRequest     bool
		expectedSecure bool
	}{
		{
			name:           "default_http",
			expectedSecure: false,
		},
		{
			name:           "tls_request",
			tlsRequest:     true,
			expectedSecure: true,
		},
		{
			name:           "secure_cookies_for_tls_termination",
			secureConfig:   true,
			expectedSecure: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &NTLMAuthHandler{SecureCookies: tc.secureConfig}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.tlsRequest {
				req.TLS = &tls.ConnectionState{}
			}
			rec := httptest.NewRecorder()

			handler.setSessionCookie(rec, req, "session-id")
			cookie := findCookie(t, rec.Result().Cookies(), ntlmSessionCookieName)

			if cookie.Secure != tc.expectedSecure {
				t.Fatalf("expected Secure=%t, got %t", tc.expectedSecure, cookie.Secure)
			}
		})
	}
}

func TestNTLMAuthRejectsMalformedPayloadsBeforeBackend(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{name: "ntlm malformed base64", header: "NTLM X"},
		{name: "negotiate malformed base64", header: "Negotiate X"},
		{name: "ntlm whitespace only", header: "NTLM  "},
		{name: "negotiate whitespace only", header: "Negotiate  "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			socketPath := filepath.Join(t.TempDir(), "auth.sock")
			listener, err := net.Listen("unix", socketPath)
			if err != nil {
				t.Fatalf("listen on unix socket: %v", err)
			}
			defer listener.Close()

			var backendCalls atomic.Int32
			acceptDone := make(chan struct{})
			go func() {
				defer close(acceptDone)
				if unixListener, ok := listener.(*net.UnixListener); ok {
					_ = unixListener.SetDeadline(time.Now().Add(250 * time.Millisecond))
				}
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				backendCalls.Add(1)
				_ = conn.Close()
			}()

			handler := &NTLMAuthHandler{SocketAddress: socketPath, Timeout: 1}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.header)
			rec := httptest.NewRecorder()

			nextCalled := false
			handler.NTLMAuth(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
			}).ServeHTTP(rec, req)

			<-acceptDone

			if nextCalled {
				t.Fatal("next handler was called for malformed Authorization header")
			}
			if calls := backendCalls.Load(); calls != 0 {
				t.Fatalf("backend was called %d time(s) for malformed Authorization header", calls)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
			}
			if got := rec.Header().Values("WWW-Authenticate"); len(got) != 2 {
				t.Fatalf("expected NTLM and Negotiate challenges, got %v", got)
			}
		})
	}
}

func TestNTLMGetAuthPayloadParsesSupportedSchemes(t *testing.T) {
	handler := &NTLMAuthHandler{}
	cases := []struct {
		name        string
		header      string
		wantPayload string
		wantMode    ntlmAuthMode
	}{
		{
			name:        "ntlm",
			header:      "NTLM TlRMTVNTUAABAAA=",
			wantPayload: "TlRMTVNTUAABAAA=",
			wantMode:    authNTLM,
		},
		{
			name:        "negotiate",
			header:      "Negotiate YIIBgQYGKwYBBQU=",
			wantPayload: "YIIBgQYGKwYBBQU=",
			wantMode:    authNegotiate,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.header)

			payload, mode, err := handler.getAuthPayload(req)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if mode != tc.wantMode {
				t.Fatalf("expected mode %v, got %v", tc.wantMode, mode)
			}
			if payload != tc.wantPayload {
				t.Fatalf("expected payload %q, got %q", tc.wantPayload, payload)
			}
		})
	}
}
