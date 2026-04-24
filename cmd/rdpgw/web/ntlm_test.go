package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
		{name: "wrong ntlm delimiter", header: "NTLMx"},
		{name: "short negotiate scheme", header: "Negotiate"},
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
