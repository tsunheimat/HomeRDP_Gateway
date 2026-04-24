package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

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
