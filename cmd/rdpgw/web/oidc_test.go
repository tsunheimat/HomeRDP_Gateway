package web

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"
)

type oidcTransactionSnapshot struct {
	State        string    `json:"state"`
	RedirectURL  string    `json:"redirect_url"`
	Nonce        string    `json:"nonce"`
	CodeVerifier string    `json:"code_verifier"`
	ExpiresAt    time.Time `json:"expires_at"`
}

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

func TestOIDCAuthenticatedRedirectIncludesNonceAndPKCETransaction(t *testing.T) {
	initOIDCTestStore(t)

	oidcHandler := (&OIDCConfig{
		OAuth2Config: &oauth2.Config{
			ClientID: "rdpgw",
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://issuer.example/authorize",
			},
		},
	}).New()

	res, sessionCookie := startOIDCLogin(t, oidcHandler, "/connect/entries/id.rdp?download=1")

	location := res.Header.Get("Location")
	if location == "" {
		t.Fatal("expected redirect location to be set")
	}
	authURL, err := http.NewRequest(http.MethodGet, location, nil)
	if err != nil {
		t.Fatalf("failed to parse auth redirect location %q: %v", location, err)
	}

	query := authURL.URL.Query()
	state := query.Get("state")
	if state == "" {
		t.Fatalf("expected redirect location %q to include state", location)
	}
	if query.Get("nonce") == "" {
		t.Fatalf("expected redirect location %q to include nonce", location)
	}
	if query.Get("code_challenge") == "" {
		t.Fatalf("expected redirect location %q to include code_challenge", location)
	}
	if got := query.Get("code_challenge_method"); got != "S256" {
		t.Fatalf("code_challenge_method = %q, want S256", got)
	}

	transaction := requireOIDCTransaction(t, sessionCookie, state)
	if transaction.State != state {
		t.Fatalf("stored state = %q, want %q", transaction.State, state)
	}
	if transaction.Nonce != query.Get("nonce") {
		t.Fatalf("stored nonce = %q, want redirect nonce %q", transaction.Nonce, query.Get("nonce"))
	}
	if transaction.CodeVerifier == "" {
		t.Fatal("expected stored code verifier to be set")
	}
	if transaction.RedirectURL != "/connect/entries/id.rdp?download=1" {
		t.Fatalf("stored redirect URL = %q, want original safe URL", transaction.RedirectURL)
	}
}

func TestOIDCPKCEChallengeMatchesStoredVerifier(t *testing.T) {
	initOIDCTestStore(t)

	oidcHandler := (&OIDCConfig{
		OAuth2Config: &oauth2.Config{
			ClientID: "rdpgw",
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://issuer.example/authorize",
			},
		},
	}).New()

	res, sessionCookie := startOIDCLogin(t, oidcHandler, "/admin")
	authURL, err := http.NewRequest(http.MethodGet, res.Header.Get("Location"), nil)
	if err != nil {
		t.Fatalf("failed to parse auth redirect: %v", err)
	}

	state := authURL.URL.Query().Get("state")
	transaction := requireOIDCTransaction(t, sessionCookie, state)

	sum := sha256.Sum256([]byte(transaction.CodeVerifier))
	wantChallenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if got := authURL.URL.Query().Get("code_challenge"); got != wantChallenge {
		t.Fatalf("code_challenge = %q, want S256 challenge %q", got, wantChallenge)
	}
}

func TestOIDCHandleCallbackSendsPKCEVerifier(t *testing.T) {
	initOIDCTestStore(t)

	const issuer = "https://issuer.example"
	signingKey := newOIDCTestSigningKey(t)
	var gotCodeVerifier string
	expectedNonce := ""

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse token request form: %v", err)
		}
		gotCodeVerifier = r.Form.Get("code_verifier")
		rawIDToken := signOIDCTestToken(t, signingKey, issuer, "rdpgw", expectedNonce, true)
		writeOIDCTestTokenResponse(t, w, rawIDToken)
	}))
	t.Cleanup(tokenServer.Close)

	oidcHandler := newOIDCTestHandler(tokenServer.URL, issuer, signingKey)
	_, sessionCookie := startOIDCLogin(t, oidcHandler, "/admin")
	transaction := requireOIDCTransactionFromCookie(t, sessionCookie)
	expectedNonce = transaction.Nonce

	callbackReq := newOIDCCallbackRequest(t, sessionCookie, transaction.State, "auth-code")
	rec := httptest.NewRecorder()
	oidcHandler.HandleCallback(rec, callbackReq)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected callback status %d, got %d with body %q", http.StatusFound, rec.Code, rec.Body.String())
	}
	if gotCodeVerifier != transaction.CodeVerifier {
		t.Fatalf("token request code_verifier = %q, want stored verifier %q", gotCodeVerifier, transaction.CodeVerifier)
	}
}

func TestOIDCHandleCallbackRejectsMissingOrMismatchedNonce(t *testing.T) {
	cases := []struct {
		name         string
		nonce        string
		includeNonce bool
	}{
		{name: "missing nonce", includeNonce: false},
		{name: "mismatched nonce", nonce: "wrong-nonce", includeNonce: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			initOIDCTestStore(t)

			const issuer = "https://issuer.example"
			signingKey := newOIDCTestSigningKey(t)
			tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rawIDToken := signOIDCTestToken(t, signingKey, issuer, "rdpgw", tc.nonce, tc.includeNonce)
				writeOIDCTestTokenResponse(t, w, rawIDToken)
			}))
			t.Cleanup(tokenServer.Close)

			oidcHandler := newOIDCTestHandler(tokenServer.URL, issuer, signingKey)
			_, sessionCookie := startOIDCLogin(t, oidcHandler, "/admin")
			transaction := requireOIDCTransactionFromCookie(t, sessionCookie)

			callbackReq := newOIDCCallbackRequest(t, sessionCookie, transaction.State, "auth-code")
			rec := httptest.NewRecorder()
			oidcHandler.HandleCallback(rec, callbackReq)

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("expected status %d, got %d with body %q", http.StatusInternalServerError, rec.Code, rec.Body.String())
			}
			if rec.Body.String() != "authentication failed\n" {
				t.Fatalf("expected generic authentication failure, got %q", rec.Body.String())
			}
		})
	}
}

func TestOIDCHandleCallbackAcceptsMatchingNonceAndSavesIdentity(t *testing.T) {
	initOIDCTestStore(t)

	const issuer = "https://issuer.example"
	signingKey := newOIDCTestSigningKey(t)
	expectedNonce := ""
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawIDToken := signOIDCTestToken(t, signingKey, issuer, "rdpgw", expectedNonce, true)
		writeOIDCTestTokenResponse(t, w, rawIDToken)
	}))
	t.Cleanup(tokenServer.Close)

	oidcHandler := newOIDCTestHandler(tokenServer.URL, issuer, signingKey)
	_, sessionCookie := startOIDCLogin(t, oidcHandler, "/original")
	transaction := requireOIDCTransactionFromCookie(t, sessionCookie)
	expectedNonce = transaction.Nonce

	callbackReq := newOIDCCallbackRequest(t, sessionCookie, transaction.State, "auth-code")
	rec := httptest.NewRecorder()
	oidcHandler.HandleCallback(rec, callbackReq)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected callback status %d, got %d with body %q", http.StatusFound, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/original" {
		t.Fatalf("callback redirect location = %q, want /original", got)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/", nil)
	addSessionCookieFromResponse(t, sessionReq, rec.Result())
	id, err := GetSessionIdentity(sessionReq)
	if err != nil {
		t.Fatalf("GetSessionIdentity returned error: %v", err)
	}
	if id == nil || !id.Authenticated() {
		t.Fatal("expected authenticated identity to be saved in session")
	}
	if id.UserName() != "matthew" {
		t.Fatalf("saved username = %q, want matthew", id.UserName())
	}
}

func TestOIDCHandleCallbackConsumesStateAfterUse(t *testing.T) {
	initOIDCTestStore(t)

	const issuer = "https://issuer.example"
	signingKey := newOIDCTestSigningKey(t)
	tokenRequests := 0
	expectedNonce := ""
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests++
		rawIDToken := signOIDCTestToken(t, signingKey, issuer, "rdpgw", expectedNonce, true)
		writeOIDCTestTokenResponse(t, w, rawIDToken)
	}))
	t.Cleanup(tokenServer.Close)

	oidcHandler := newOIDCTestHandler(tokenServer.URL, issuer, signingKey)
	_, sessionCookie := startOIDCLogin(t, oidcHandler, "/original")
	transaction := requireOIDCTransactionFromCookie(t, sessionCookie)
	expectedNonce = transaction.Nonce

	firstReq := newOIDCCallbackRequest(t, sessionCookie, transaction.State, "auth-code")
	firstRec := httptest.NewRecorder()
	oidcHandler.HandleCallback(firstRec, firstReq)
	if firstRec.Code != http.StatusFound {
		t.Fatalf("expected first callback status %d, got %d with body %q", http.StatusFound, firstRec.Code, firstRec.Body.String())
	}

	replayReq := httptest.NewRequest(http.MethodGet, "/callback?state="+transaction.State+"&code=auth-code", nil)
	addSessionCookieFromResponse(t, replayReq, firstRec.Result())
	replayReq = identity.AddToRequestCtx(identity.NewUser(), replayReq)
	replayRec := httptest.NewRecorder()
	oidcHandler.HandleCallback(replayRec, replayReq)

	if replayRec.Code != http.StatusBadRequest {
		t.Fatalf("expected replay status %d, got %d with body %q", http.StatusBadRequest, replayRec.Code, replayRec.Body.String())
	}
	if tokenRequests != 1 {
		t.Fatalf("expected replay to fail before token exchange; token requests = %d", tokenRequests)
	}
}

func TestOIDCHandleCallbackRejectsReplayedOriginalLoginCookieBeforeTokenExchange(t *testing.T) {
	initOIDCTestStore(t)

	const issuer = "https://issuer.example"
	signingKey := newOIDCTestSigningKey(t)
	tokenRequests := 0
	expectedNonce := ""
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests++
		rawIDToken := signOIDCTestToken(t, signingKey, issuer, "rdpgw", expectedNonce, true)
		writeOIDCTestTokenResponse(t, w, rawIDToken)
	}))
	t.Cleanup(tokenServer.Close)

	oidcHandler := newOIDCTestHandler(tokenServer.URL, issuer, signingKey)
	_, originalLoginCookie := startOIDCLogin(t, oidcHandler, "/original")
	transaction := requireOIDCTransactionFromCookie(t, originalLoginCookie)
	expectedNonce = transaction.Nonce

	firstReq := newOIDCCallbackRequest(t, originalLoginCookie, transaction.State, "auth-code")
	firstRec := httptest.NewRecorder()
	oidcHandler.HandleCallback(firstRec, firstReq)
	if firstRec.Code != http.StatusFound {
		t.Fatalf("expected first callback status %d, got %d with body %q", http.StatusFound, firstRec.Code, firstRec.Body.String())
	}

	replayReq := newOIDCCallbackRequest(t, originalLoginCookie, transaction.State, "auth-code")
	replayRec := httptest.NewRecorder()
	oidcHandler.HandleCallback(replayRec, replayReq)

	if replayRec.Code != http.StatusBadRequest {
		t.Fatalf("expected stale-cookie replay status %d, got %d with body %q", http.StatusBadRequest, replayRec.Code, replayRec.Body.String())
	}
	if tokenRequests != 1 {
		t.Fatalf("expected stale-cookie replay to fail before token exchange; token requests = %d", tokenRequests)
	}
}

func TestOIDCHandleCallbackRejectsExpiredTransactionBeforeTokenExchange(t *testing.T) {
	initOIDCTestStore(t)

	const issuer = "https://issuer.example"
	signingKey := newOIDCTestSigningKey(t)
	tokenRequests := 0
	expectedNonce := ""
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests++
		rawIDToken := signOIDCTestToken(t, signingKey, issuer, "rdpgw", expectedNonce, true)
		writeOIDCTestTokenResponse(t, w, rawIDToken)
	}))
	t.Cleanup(tokenServer.Close)

	oidcHandler := newOIDCTestHandler(tokenServer.URL, issuer, signingKey)
	_, sessionCookie := startOIDCLogin(t, oidcHandler, "/original")
	transaction := requireOIDCTransactionFromCookie(t, sessionCookie)
	expectedNonce = transaction.Nonce
	expiredCookie := oidcTransactionCookieWithExpiry(t, sessionCookie, time.Now().Add(-time.Minute))

	callbackReq := newOIDCCallbackRequest(t, expiredCookie, transaction.State, "auth-code")
	rec := httptest.NewRecorder()
	oidcHandler.HandleCallback(rec, callbackReq)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected expired transaction status %d, got %d with body %q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if tokenRequests != 0 {
		t.Fatalf("expected expired transaction to fail before token exchange; token requests = %d", tokenRequests)
	}
}

func initOIDCTestStore(t *testing.T) {
	t.Helper()

	sessionKey := []byte("testsessionkeytestsessionkey1234")
	encryptionKey := []byte("testencryptionkeytestencrypt1234")
	InitStore(sessionKey, encryptionKey, "cookie", 8192)
}

func startOIDCLogin(t *testing.T, oidcHandler *OIDC, requestURI string) (*http.Response, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RequestURI = requestURI
	w := httptest.NewRecorder()

	handler := EnrichContext(oidcHandler.Authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected downstream handler call")
	})))
	handler.ServeHTTP(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("expected OIDC login redirect status %d, got %d", http.StatusFound, res.StatusCode)
	}
	return res, requireSessionCookie(t, res)
}

func requireSessionCookie(t *testing.T, res *http.Response) *http.Cookie {
	t.Helper()

	var sessionCookie *http.Cookie
	for _, cookie := range res.Cookies() {
		if cookie.Name == rdpGwSession {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected %s cookie in response", rdpGwSession)
	}
	return sessionCookie
}

func addSessionCookieFromResponse(t *testing.T, req *http.Request, res *http.Response) {
	t.Helper()

	req.AddCookie(requireSessionCookie(t, res))
}

func oidcTransactionCookieWithExpiry(t *testing.T, sessionCookie *http.Cookie, expiresAt time.Time) *http.Cookie {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/callback", nil)
	req.AddCookie(sessionCookie)
	session, err := GetSession(req)
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}

	raw, exists := session.Values[oidcStateKey]
	if !exists {
		t.Fatalf("expected %s in session", oidcStateKey)
	}

	encoded, ok := raw.(string)
	if !ok {
		t.Fatalf("expected OIDC transaction JSON string in session, got %T", raw)
	}

	var transaction oidcTransactionSnapshot
	if err := json.Unmarshal([]byte(encoded), &transaction); err != nil {
		t.Fatalf("expected OIDC transaction to be JSON, got %q: %v", encoded, err)
	}
	transaction.ExpiresAt = expiresAt
	stateValue, err := json.Marshal(transaction)
	if err != nil {
		t.Fatalf("failed to marshal OIDC transaction: %v", err)
	}
	session.Values[oidcStateKey] = string(stateValue)
	session.Options.MaxAge = int(CacheExpiration.Seconds())

	w := httptest.NewRecorder()
	if err := sessionStore.Save(req, w, session); err != nil {
		t.Fatalf("failed to save mutated OIDC transaction: %v", err)
	}
	return requireSessionCookie(t, w.Result())
}

func requireOIDCTransactionFromCookie(t *testing.T, sessionCookie *http.Cookie) oidcTransactionSnapshot {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/callback", nil)
	req.AddCookie(sessionCookie)
	session, err := GetSession(req)
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}

	raw, exists := session.Values[oidcStateKey]
	if !exists {
		t.Fatalf("expected %s in session", oidcStateKey)
	}

	var encoded string
	switch v := raw.(type) {
	case string:
		encoded = v
	case []byte:
		encoded = string(v)
	default:
		t.Fatalf("expected OIDC transaction JSON string in session, got %T", raw)
	}

	var transaction oidcTransactionSnapshot
	if err := json.Unmarshal([]byte(encoded), &transaction); err != nil {
		t.Fatalf("expected OIDC transaction to be JSON, got %q: %v", encoded, err)
	}
	return transaction
}

func requireOIDCTransaction(t *testing.T, sessionCookie *http.Cookie, state string) oidcTransactionSnapshot {
	t.Helper()

	transaction := requireOIDCTransactionFromCookie(t, sessionCookie)
	if transaction.State != state {
		t.Fatalf("stored state = %q, want %q", transaction.State, state)
	}
	return transaction
}

func newOIDCTestSigningKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return key
}

func newOIDCTestHandler(tokenURL string, issuer string, signingKey *rsa.PrivateKey) *OIDC {
	return (&OIDCConfig{
		OAuth2Config: &oauth2.Config{
			ClientID:     "rdpgw",
			ClientSecret: "client-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  issuer + "/authorize",
				TokenURL: tokenURL,
			},
		},
		OIDCTokenVerifier: oidc.NewVerifier(
			issuer,
			&oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&signingKey.PublicKey}},
			&oidc.Config{ClientID: "rdpgw"},
		),
	}).New()
}

func signOIDCTestToken(t *testing.T, signingKey *rsa.PrivateKey, issuer string, audience string, nonce string, includeNonce bool) string {
	t.Helper()

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: signingKey},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer:   issuer,
		Subject:  "user-123",
		Audience: jwt.Audience{audience},
		Expiry:   jwt.NewNumericDate(now.Add(time.Hour)),
		IssuedAt: jwt.NewNumericDate(now),
	}
	privateClaims := map[string]interface{}{
		"preferred_username": "matthew",
		"email":              "matthew@example.com",
		"name":               "Matthew Chiu",
		"groups":             []string{"rdpgw-admins"},
	}
	if includeNonce {
		privateClaims["nonce"] = nonce
	}

	rawIDToken, err := jwt.Signed(signer).Claims(claims).Claims(privateClaims).Serialize()
	if err != nil {
		t.Fatalf("failed to sign ID token: %v", err)
	}
	return rawIDToken
}

func writeOIDCTestTokenResponse(t *testing.T, w http.ResponseWriter, rawIDToken string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": "access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     rawIDToken,
	}); err != nil {
		t.Fatalf("failed to write token response: %v", err)
	}
}

func newOIDCCallbackRequest(t *testing.T, sessionCookie *http.Cookie, state string, code string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/callback?state="+state+"&code="+code, nil)
	req.AddCookie(sessionCookie)
	return identity.AddToRequestCtx(identity.NewUser(), req)
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
