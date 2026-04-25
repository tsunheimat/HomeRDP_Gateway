package web

import (
	"log"
	"net/http"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

type Header struct {
	userHeader        string
	userIdHeader      string
	emailHeader       string
	displayNameHeader string
	trustedProxies    *TrustedProxyChecker
}

type HeaderConfig struct {
	UserHeader        string
	UserIdHeader      string
	EmailHeader       string
	DisplayNameHeader string
	TrustedProxyCIDRs []string
}

func (c *HeaderConfig) New() *Header {
	trustedProxies, err := NewTrustedProxyChecker(c.TrustedProxyCIDRs)
	if err != nil {
		log.Printf("invalid header auth trusted proxy config: %s", err)
		trustedProxies, _ = NewTrustedProxyChecker(nil)
	}
	return &Header{
		userHeader:        c.UserHeader,
		userIdHeader:      c.UserIdHeader,
		emailHeader:       c.EmailHeader,
		displayNameHeader: c.DisplayNameHeader,
		trustedProxies:    trustedProxies,
	}
}

// Authenticated middleware that extracts user identity from configurable proxy headers
func (h *Header) Authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := identity.FromRequestCtx(r)

		// Check if user is already authenticated
		if id.Authenticated() {
			next.ServeHTTP(w, r)
			return
		}

		if !h.trustedProxies.IsTrustedRemoteAddr(r.RemoteAddr) {
			http.Error(w, "Header authentication requires a trusted proxy", http.StatusUnauthorized)
			return
		}

		// Extract username from configured user header
		userName := r.Header.Get(h.userHeader)
		if userName == "" {
			http.Error(w, "No authenticated user from proxy", http.StatusUnauthorized)
			return
		}

		// Set identity for downstream processing
		id.SetUserName(userName)
		id.SetAuthenticated(true)
		id.SetAuthTime(time.Now())

		// Set optional user attributes from headers
		if h.userIdHeader != "" {
			if userId := r.Header.Get(h.userIdHeader); userId != "" {
				id.SetAttribute("user_id", userId)
			}
		}

		if h.emailHeader != "" {
			if email := r.Header.Get(h.emailHeader); email != "" {
				id.SetEmail(email)
			}
		}

		if h.displayNameHeader != "" {
			if displayName := r.Header.Get(h.displayNameHeader); displayName != "" {
				id.SetDisplayName(displayName)
			}
		}

		// Save the session identity
		if err := SaveSessionIdentity(r, w, id); err != nil {
			log.Printf("Header authentication: failed to save session: %v", err)
			http.Error(w, "failed to save session", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
