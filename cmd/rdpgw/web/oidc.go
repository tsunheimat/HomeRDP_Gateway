package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
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
}

type OIDCConfig struct {
	OAuth2Config      *oauth2.Config
	OIDCTokenVerifier *oidc.IDTokenVerifier
	GroupsClaim       string
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
	}
}

// storeOIDCState stores the OIDC state and redirect URL in the session
func storeOIDCState(w http.ResponseWriter, r *http.Request, state string, redirectURL string) error {
	session, err := GetSession(r)
	if err != nil {
		return err
	}

	// Store state data directly as a concatenated string: state + "|" + redirectURL
	stateValue := state + "|" + redirectURL
	session.Values[oidcStateKey] = stateValue
	session.Options.MaxAge = int(CacheExpiration.Seconds())

	return sessionStore.Save(r, w, session)
}

// getOIDCState retrieves the redirect URL for the given state from the session
func getOIDCState(r *http.Request, state string) (string, bool) {
	session, err := GetSession(r)
	if err != nil {
		log.Printf("Error getting session for OIDC state: %v", err)
		return "", false
	}

	stateData, exists := session.Values[oidcStateKey]
	if !exists {
		log.Printf("No OIDC state data found in session")
		return "", false
	}

	stateValue, ok := stateData.(string)
	if !ok {
		log.Printf("Invalid OIDC state data format in session")
		return "", false
	}

	// Parse state data: state + "|" + redirectURL
	expectedPrefix := state + "|"
	if !strings.HasPrefix(stateValue, expectedPrefix) {
		log.Printf("OIDC state '%s' not found in session", state)
		return "", false
	}

	redirectURL := stateValue[len(expectedPrefix):]
	return redirectURL, true
}

func (h *OIDC) HandleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	url, found := getOIDCState(r, state)
	if !found {
		log.Printf("OIDC HandleCallback: unknown state '%s'", state)
		http.Error(w, "unknown state", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	oauth2Token, err := h.oAuth2Config.Exchange(ctx, r.URL.Query().Get("code"))
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

	id := identity.FromRequestCtx(r)

	if err := populateIdentityFromClaims(id, data, h.groupsClaim); err != nil {
		log.Printf("OIDC HandleCallback: failed to populate identity from claims: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	id.SetAuthenticated(true)
	id.SetAuthTime(time.Now())
	id.SetAttribute(identity.AttrAccessToken, oauth2Token.AccessToken)

	if err := SaveSessionIdentity(r, w, id); err != nil {
		log.Printf("OIDC HandleCallback: failed to save session identity: %v", err)
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
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
			seed := make([]byte, 16)
			_, err := rand.Read(seed)
			if err != nil {
				log.Printf("OIDC Authenticated: failed to generate state: %v", err)
				http.Error(w, "authentication failed", http.StatusInternalServerError)
				return
			}
			state := hex.EncodeToString(seed)

			log.Printf("OIDC Authenticated: storing state '%s' for redirect to '%s'", state, r.RequestURI)
			err = storeOIDCState(w, r, state, r.RequestURI)
			if err != nil {
				log.Printf("OIDC Authenticated: failed to store state: %v", err)
				http.Error(w, "failed to save session", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, h.oAuth2Config.AuthCodeURL(state), http.StatusFound)
			return
		}

		// replace the identity with the one from the sessions
		next.ServeHTTP(w, r)
	})
}
