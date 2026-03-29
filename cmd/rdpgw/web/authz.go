package web

import (
	"net/http"
	"strings"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

func IsAdmin(id identity.Identity, adminGroups []string) bool {
	if id == nil {
		return false
	}

	for _, group := range adminGroups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if id.InGroup(group) {
			return true
		}
	}

	return false
}

func (h *Handler) AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := identity.FromRequestCtx(r)
		if id == nil || !id.Authenticated() {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if !IsAdmin(id, h.adminGroups) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
