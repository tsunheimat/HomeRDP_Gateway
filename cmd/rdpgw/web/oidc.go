package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	CacheExpiration = time.Minute * 2
	CleanupInterval = time.Minute * 5
	oidcStateKey    = "OIDCSTATE"
)

type OIDC struct {
	oAuth2Config      *oauth2.Config
	oidcTokenVerifier *oidc.IDTokenVerifier
	groupsClaim       string
	consumedStates    *oidcConsumedStateCache
}

type OIDCConfig struct {
	OAuth2Config      *oauth2.Config
	OIDCTokenVerifier *oidc.IDTokenVerifier
	GroupsClaim       string
}

type oidcTransaction struct {
	State        string    `json:"state"`
	RedirectURL  string    `json:"redirect_url"`
	Nonce        string    `json:"nonce"`
	CodeVerifier string    `json:"code_verifier"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type oidcConsumedStateCache struct {
	mu     sync.Mutex
	ttl    time.Duration
	states map[string]time.Time
}

func (c *OIDCConfig) New() *OIDC {
	groupsClaim := c.GroupsClaim
	if groupsClaim == "" {
		groupsClaim = "groups"
	}

	return &OIDC{
		oAuth2Config:      c.OAuth2Config,
		oidcTokenVerifier: c.OIDCTokenVerifier,
		groupsClaim:       groupsClaim,
		consumedStates:    newOIDCConsumedStateCache(CacheExpiration),
	}
}

func newOIDCConsumedStateCache(ttl time.Duration) *oidcConsumedStateCache {
	if ttl <= 0 {
		ttl = CacheExpiration
	}
	return &oidcConsumedStateCache{
		ttl:    ttl,
		states: make(map[string]time.Time),
	}
}

func (c *oidcConsumedStateCache) markConsumed(state string, now time.Time) bool {
	if state == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for cachedState, expiresAt := range c.states {
		if !expiresAt.After(now) {
			delete(c.states, cachedState)
		}
	}
	if _, exists := c.states[state]; exists {
		return false
	}

	c.states[state] = now.Add(c.ttl)
	return true
}

func randomURLSafeString(byteLen int) (string, error) {
	seed := make([]byte, byteLen)
	if _, err := rand.Read(seed); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(seed), nil
}

func newOIDCTransaction(state string, redirectURL string) (*oidcTransaction, error) {
	nonce, err := randomURLSafeString(32)
	if err != nil {
		return nil, err
	}
	codeVerifier, err := randomURLSafeString(32)
	if err != nil {
		return nil, err
	}

	return &oidcTransaction{
		State:        state,
		RedirectURL:  redirectURL,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
		ExpiresAt:    time.Now().Add(CacheExpiration),
	}, nil
}

// safeOIDCRedirectURL returns a site-local absolute-path reference suitable for
// a post-login redirect. Unsafe values fall back to the application root so a
// crafted unauthenticated request cannot become a protocol-relative or absolute
// external redirect after OIDC login.
func safeOIDCRedirectURL(redirectURL string) string {
	if redirectURL == "" || !strings.HasPrefix(redirectURL, "/") || strings.HasPrefix(redirectURL, "//") {
		return "/"
	}

	parsed, err := url.ParseRequestURI(redirectURL)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return "/"
	}

	return parsed.RequestURI()
}

func storeOIDCTransaction(w http.ResponseWriter, r *http.Request, transaction *oidcTransaction) error {
	session, err := GetSession(r)
	if err != nil {
		return err
	}

	stateValue, err := json.Marshal(transaction)
	if err != nil {
		return err
	}
	session.Values[oidcStateKey] = string(stateValue)
	session.Options.MaxAge = int(CacheExpiration.Seconds())

	return sessionStore.Save(r, w, session)
}

// storeOIDCState stores a complete OIDC transaction in the session.
func storeOIDCState(w http.ResponseWriter, r *http.Request, state string, redirectURL string) error {
	transaction, err := newOIDCTransaction(state, redirectURL)
	if err != nil {
		return err
	}
	return storeOIDCTransaction(w, r, transaction)
}

func readOIDCTransaction(r *http.Request) (*oidcTransaction, bool) {
	session, err := GetSession(r)
	if err != nil {
		log.Printf("Error getting session for OIDC state: %v", err)
		return nil, false
	}

	stateData, exists := session.Values[oidcStateKey]
	if !exists {
		log.Printf("No OIDC state data found in session")
		return nil, false
	}

	stateValue, ok := stateData.(string)
	if !ok {
		log.Printf("Invalid OIDC state data format in session")
		return nil, false
	}

	var transaction oidcTransaction
	if err := json.Unmarshal([]byte(stateValue), &transaction); err == nil {
		return &transaction, true
	}

	parts := strings.SplitN(stateValue, "|", 2)
	if len(parts) != 2 {
		log.Printf("Invalid OIDC state data format in session")
		return nil, false
	}
	return &oidcTransaction{State: parts[0], RedirectURL: parts[1]}, true
}

func getOIDCTransaction(r *http.Request, state string) (*oidcTransaction, bool) {
	transaction, found := readOIDCTransaction(r)
	if !found {
		return nil, false
	}
	if transaction.State != state {
		log.Printf("OIDC state not found in session")
		return nil, false
	}
	if !transaction.ExpiresAt.After(time.Now()) {
		log.Printf("OIDC transaction expired")
		return nil, false
	}
	return transaction, true
}

// getOIDCState retrieves the redirect URL for the given state from the session
func getOIDCState(r *http.Request, state string) (string, bool) {
	transaction, found := getOIDCTransaction(r, state)
	if !found {
		return "", false
	}
	return transaction.RedirectURL, true
}

func (h *OIDC) consumeOIDCTransaction(w http.ResponseWriter, r *http.Request, state string) (*oidcTransaction, bool, error) {
	session, err := GetSession(r)
	if err != nil {
		return nil, false, err
	}

	transaction, found := getOIDCTransaction(r, state)
	if !found {
		return nil, false, nil
	}
	if !h.consumedStates.markConsumed(state, time.Now()) {
		return nil, false, nil
	}
	delete(session.Values, oidcStateKey)
	if err := sessionStore.Save(r, w, session); err != nil {
		return nil, false, err
	}
	return transaction, true, nil
}

func (h *OIDC) HandleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	transaction, found, err := h.consumeOIDCTransaction(w, r, state)
	if err != nil {
		log.Printf("OIDC HandleCallback: failed to consume OIDC state: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	if !found {
		log.Printf("OIDC HandleCallback: unknown state")
		http.Error(w, "unknown state", http.StatusBadRequest)
		return
	}
	if transaction.CodeVerifier == "" {
		log.Printf("OIDC HandleCallback: OIDC transaction missing PKCE verifier")
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	oauth2Token, err := h.oAuth2Config.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(transaction.CodeVerifier))
	if err != nil {
		log.Printf("OIDC HandleCallback: failed to exchange token: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Printf("OIDC HandleCallback: oauth2 token missing id_token")
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	idToken, err := h.oidcTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("OIDC HandleCallback: failed to verify ID token: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	resp := struct {
		OAuth2Token   *oauth2.Token
		IDTokenClaims *json.RawMessage // ID Token payload is just JSON.
	}{oauth2Token, new(json.RawMessage)}

	if err := idToken.Claims(&resp.IDTokenClaims); err != nil {
		log.Printf("OIDC HandleCallback: failed to read ID token claims: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(*resp.IDTokenClaims, &data); err != nil {
		log.Printf("OIDC HandleCallback: failed to parse ID token claims: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	nonce, ok := data["nonce"].(string)
	if !ok || subtle.ConstantTimeCompare([]byte(nonce), []byte(transaction.Nonce)) != 1 {
		log.Printf("OIDC HandleCallback: ID token nonce validation failed")
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	id := identity.FromRequestCtx(r)

	if err := populateIdentityFromClaims(id, data, h.groupsClaim); err != nil {
		log.Printf("OIDC HandleCallback: failed to populate identity from claims: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	id.SetAuthenticated(true)
	id.SetAuthTime(time.Now())
	id.SetAttribute(identity.AttrAccessToken, oauth2Token.AccessToken)
	id.SetAttribute(identity.AttrAuthSource, identity.AuthSourceOIDC)

	if err := SaveSessionIdentity(r, w, id); err != nil {
		log.Printf("OIDC HandleCallback: failed to save session identity: %v", err)
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, safeOIDCRedirectURL(transaction.RedirectURL), http.StatusFound)
}

func findUsernameInClaims(data map[string]interface{}) string {
	candidates := []string{"preferred_username", "unique_name", "upn", "username"}
	for _, claim := range candidates {
		userName, found := data[claim].(string)
		if found {
			return userName
		}
	}

	return ""
}

func populateIdentityFromClaims(id identity.Identity, data map[string]interface{}, groupsClaim string) error {
	userName := findUsernameInClaims(data)
	if userName == "" {
		return errors.New("no oidc claim for username found")
	}
	id.SetUserName(userName)

	if email, ok := data["email"].(string); ok {
		id.SetEmail(email)
	}

	if displayName, ok := data["name"].(string); ok {
		id.SetDisplayName(displayName)
	} else if displayName, ok := data["display_name"].(string); ok {
		id.SetDisplayName(displayName)
	}

	if groupsClaim == "" {
		groupsClaim = "groups"
	}
	id.SetGroups(extractGroups(data[groupsClaim]))

	return nil
}

func extractGroups(raw interface{}) []string {
	switch groups := raw.(type) {
	case []string:
		return groups
	case []interface{}:
		membership := make([]string, 0, len(groups))
		for _, group := range groups {
			if groupName, ok := group.(string); ok {
				membership = append(membership, groupName)
			}
		}
		return membership
	case string:
		return []string{groups}
	default:
		return nil
	}
}

func (h *OIDC) Authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := identity.FromRequestCtx(r)

		if !id.Authenticated() {
			state, err := randomURLSafeString(32)
			if err != nil {
				log.Printf("OIDC Authenticated: failed to generate state: %v", err)
				http.Error(w, "authentication failed", http.StatusInternalServerError)
				return
			}

			redirectURL := safeOIDCRedirectURL(r.RequestURI)
			transaction, err := newOIDCTransaction(state, redirectURL)
			if err != nil {
				log.Printf("OIDC Authenticated: failed to generate OIDC transaction: %v", err)
				http.Error(w, "authentication failed", http.StatusInternalServerError)
				return
			}
			log.Printf("OIDC Authenticated: storing OIDC transaction for redirect to '%s'", redirectURL)
			if err := storeOIDCTransaction(w, r, transaction); err != nil {
				log.Printf("OIDC Authenticated: failed to store state: %v", err)
				http.Error(w, "failed to save session", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, h.oAuth2Config.AuthCodeURL(
				state,
				oauth2.SetAuthURLParam("nonce", transaction.Nonce),
				oauth2.S256ChallengeOption(transaction.CodeVerifier),
			), http.StatusFound)
			return
		}

		// replace the identity with the one from the sessions
		next.ServeHTTP(w, r)
	})
}
