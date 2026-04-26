package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"golang.org/x/oauth2"
)

func TestFindUserNameInClaims(t *testing.T) {
	cases := []struct {
		data map[string]interface{}
		ret  string
		name string
	}{
		{
			data: map[string]interface{}{
				"preferred_username": "exists",
			},
			ret:  "exists",
			name: "preferred_username",
		},
		{
			data: map[string]interface{}{
				"upn": "exists",
			},
			ret:  "exists",
			name: "upn",
		},
		{
			data: map[string]interface{}{
				"unique_name": "exists",
			},
			ret:  "exists",
			name: "unique_name",
		},
		{
			data: map[string]interface{}{
				"fail": "exists",
			},
			ret:  "",
			name: "fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := findUsernameInClaims(tc.data)
			if s != tc.ret {
				t.Fatalf("expected return: %v, got: %v", tc.ret, s)
			}
		})
	}
}

func TestSafeOIDCRedirectURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "root", in: "/", want: "/"},
		{name: "normal path", in: "/admin", want: "/admin"},
		{name: "path with query", in: "/connect/entries/id.rdp?download=1", want: "/connect/entries/id.rdp?download=1"},
		{name: "protocol relative external", in: "//evil.example/path", want: "/"},
		{name: "absolute external", in: "https://evil.example/path", want: "/"},
		{name: "empty", in: "", want: "/"},
		{name: "relative path", in: "admin", want: "/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safeOIDCRedirectURL(tc.in); got != tc.want {
				t.Fatalf("safeOIDCRedirectURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestOIDCStateManagement(t *testing.T) {
	// Initialize session store for testing
	sessionKey := []byte("testsessionkeytestsessionkey1234")    // 32 bytes
	encryptionKey := []byte("testencryptionkeytestencrypt1234") // 32 bytes
	InitStore(sessionKey, encryptionKey, "cookie", 8192)

	// Test storing and retrieving OIDC state
	t.Run("StoreAndRetrieveState", func(t *testing.T) {
		// Create test request and response
		req := httptest.NewRequest("GET", "/connect", nil)
		w := httptest.NewRecorder()

		state := "test-state-12345"
		redirectURL := "/original-request"

		// Store state
		err := storeOIDCState(w, req, state, redirectURL)
		if err != nil {
			t.Fatalf("Failed to store OIDC state: %v", err)
		}

		// Create a new request with the same session cookie
		callbackReq := httptest.NewRequest("GET", "/callback", nil)

		// Copy session cookie from response to new request
		for _, cookie := range w.Result().Cookies() {
			callbackReq.AddCookie(cookie)
		}

		// Retrieve state
		retrievedURL, found := getOIDCState(callbackReq, state)
		if !found {
			t.Fatal("Expected to find OIDC state, but it was not found")
		}

		if retrievedURL != redirectURL {
			t.Fatalf("Expected redirect URL '%s', got '%s'", redirectURL, retrievedURL)
		}
	})

	t.Run("StateNotFound", func(t *testing.T) {
		// Create test request
		req := httptest.NewRequest("GET", "/callback", nil)

		// Try to retrieve non-existent state
		_, found := getOIDCState(req, "non-existent-state")
		if found {
			t.Fatal("Expected state not to be found, but it was found")
		}
	})

	t.Run("EmptySession", func(t *testing.T) {
		// Create fresh request with no session
		req := httptest.NewRequest("GET", "/callback", nil)

		// Try to retrieve state from empty session
		_, found := getOIDCState(req, "any-state")
		if found {
			t.Fatal("Expected state not to be found in empty session, but it was found")
		}
	})
}

func TestOIDCAuthenticatedRedirectSetsSingleSessionCookie(t *testing.T) {
	sessionKey := []byte("testsessionkeytestsessionkey1234")
	encryptionKey := []byte("testencryptionkeytestencrypt1234")
	InitStore(sessionKey, encryptionKey, "cookie", 8192)

	oidc := (&OIDCConfig{
		OAuth2Config: &oauth2.Config{
			ClientID: "rdpgw",
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://issuer.example/authorize",
			},
		},
	}).New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler := EnrichContext(oidc.Authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected downstream handler call")
	})))
	handler.ServeHTTP(w, req)

	res := w.Result()
	var sessionCookies []*http.Cookie
	for _, cookie := range res.Cookies() {
		if cookie.Name == rdpGwSession {
			sessionCookies = append(sessionCookies, cookie)
		}
	}

	if len(sessionCookies) != 1 {
		t.Fatalf("expected exactly one %s cookie, got %d", rdpGwSession, len(sessionCookies))
	}

	location := res.Header.Get("Location")
	if location == "" {
		t.Fatal("expected redirect location to be set")
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "/callback", nil)
	callbackReq.AddCookie(sessionCookies[0])
	state := ""
	redirectReq := req
	if parsedURL, err := http.NewRequest(http.MethodGet, location, nil); err == nil {
		state = parsedURL.URL.Query().Get("state")
		redirectReq = parsedURL
	}
	if state == "" {
		t.Fatalf("expected redirect location %q to include a state parameter", location)
	}

	retrievedURL, found := getOIDCState(callbackReq, state)
	if !found {
		t.Fatal("expected to find OIDC state from the redirect session cookie")
	}
	if retrievedURL != redirectReq.RequestURI && retrievedURL != "/" {
		t.Fatalf("expected redirect URL to be preserved, got %q", retrievedURL)
	}
}

func TestOIDCAuthenticatedStoresOnlySafeRedirectURL(t *testing.T) {
	cases := []struct {
		name       string
		requestURI string
		want       string
	}{
		{name: "normal admin path", requestURI: "/admin", want: "/admin"},
		{name: "protocol relative external", requestURI: "//evil.example/path", want: "/"},
		{name: "absolute external", requestURI: "https://evil.example/path", want: "/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessionKey := []byte("testsessionkeytestsessionkey1234")
			encryptionKey := []byte("testencryptionkeytestencrypt1234")
			InitStore(sessionKey, encryptionKey, "cookie", 8192)

			oidc := (&OIDCConfig{
				OAuth2Config: &oauth2.Config{
					ClientID: "rdpgw",
					Endpoint: oauth2.Endpoint{
						AuthURL: "https://issuer.example/authorize",
					},
				},
			}).New()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RequestURI = tc.requestURI
			w := httptest.NewRecorder()

			handler := EnrichContext(oidc.Authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("unexpected downstream handler call")
			})))
			handler.ServeHTTP(w, req)

			location := w.Result().Header.Get("Location")
			if location == "" {
				t.Fatal("expected redirect location to be set")
			}
			parsedLocation, err := http.NewRequest(http.MethodGet, location, nil)
			if err != nil {
				t.Fatalf("failed to parse auth redirect location %q: %v", location, err)
			}
			state := parsedLocation.URL.Query().Get("state")
			if state == "" {
				t.Fatalf("expected redirect location %q to include a state parameter", location)
			}

			callbackReq := httptest.NewRequest(http.MethodGet, "/callback", nil)
			for _, cookie := range w.Result().Cookies() {
				callbackReq.AddCookie(cookie)
			}

			got, found := getOIDCState(callbackReq, state)
			if !found {
				t.Fatal("expected to find OIDC state from the redirect session cookie")
			}
			if got != tc.want {
				t.Fatalf("stored redirect URL = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOIDCHandleCallbackReturnsGenericTokenExchangeError(t *testing.T) {
	sessionKey := []byte("testsessionkeytestsessionkey1234")
	encryptionKey := []byte("testencryptionkeytestencrypt1234")
	InitStore(sessionKey, encryptionKey, "cookie", 8192)

	const rawProviderError = "provider internal stack trace: invalid_client_secret"
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, rawProviderError, http.StatusInternalServerError)
	}))
	t.Cleanup(tokenServer.Close)

	oidc := (&OIDCConfig{
		OAuth2Config: &oauth2.Config{
			ClientID:     "rdpgw",
			ClientSecret: "client-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		},
	}).New()

	stateReq := httptest.NewRequest(http.MethodGet, "/connect", nil)
	stateRec := httptest.NewRecorder()
	if err := storeOIDCState(stateRec, stateReq, "state-123", "/original"); err != nil {
		t.Fatalf("storeOIDCState returned error: %v", err)
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "/callback?state=state-123&code=bad-code", nil)
	for _, cookie := range stateRec.Result().Cookies() {
		callbackReq.AddCookie(cookie)
	}
	callbackReq = identity.AddToRequestCtx(identity.NewUser(), callbackReq)

	logs := captureTestLogs(t)
	rec := httptest.NewRecorder()
	oidc.HandleCallback(rec, callbackReq)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if rec.Body.String() != "authentication failed\n" {
		t.Fatalf("expected generic authentication failure, got %q", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), rawProviderError) || strings.Contains(rec.Body.String(), "Failed to exchange token") {
		t.Fatalf("response body leaked provider error: %q", rec.Body.String())
	}
	if !strings.Contains(logs.String(), rawProviderError) {
		t.Fatalf("expected detailed provider error in server logs, got %q", logs.String())
	}
}

func TestPopulateIdentityFromClaims(t *testing.T) {
	id := identity.NewUser()
	claims := map[string]interface{}{
		"preferred_username": "matthew",
		"email":              "matthew@example.com",
		"name":               "Matthew Chiu",
		"groups":             []interface{}{"rdpgw-admins", "homelab-users", "rdpgw-admins"},
	}

	if err := populateIdentityFromClaims(id, claims, "groups"); err != nil {
		t.Fatalf("populateIdentityFromClaims returned error: %v", err)
	}

	if id.UserName() != "matthew" {
		t.Fatalf("expected username matthew, got %q", id.UserName())
	}

	if id.Email() != "matthew@example.com" {
		t.Fatalf("expected email matthew@example.com, got %q", id.Email())
	}

	if id.DisplayName() != "Matthew Chiu" {
		t.Fatalf("expected display name Matthew Chiu, got %q", id.DisplayName())
	}

	expectedGroups := []string{"homelab-users", "rdpgw-admins"}
	if !reflect.DeepEqual(id.Groups(), expectedGroups) {
		t.Fatalf("expected groups %v, got %v", expectedGroups, id.Groups())
	}
}
