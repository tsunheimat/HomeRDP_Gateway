package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

func TestIsAdmin(t *testing.T) {
	id := identity.NewUser()
	id.SetGroups([]string{"homelab-users", "rdpgw-admins"})

	if !IsAdmin(id, []string{"rdpgw-admins"}) {
		t.Fatalf("expected IsAdmin to return true")
	}

	if IsAdmin(id, []string{"other-group"}) {
		t.Fatalf("expected IsAdmin to return false")
	}
}

func TestAdminOnly(t *testing.T) {
	handler := &Handler{adminGroups: []string{"rdpgw-admins"}}
	next := handler.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = identity.AddToRequestCtx(identity.NewUser(), req)
		rr := httptest.NewRecorder()

		next.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		id := identity.NewUser()
		id.SetAuthenticated(true)
		id.SetGroups([]string{"homelab-users"})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = identity.AddToRequestCtx(id, req)
		rr := httptest.NewRecorder()

		next.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rr.Code)
		}
	})

	t.Run("allowed", func(t *testing.T) {
		id := identity.NewUser()
		id.SetAuthenticated(true)
		id.SetGroups([]string{"rdpgw-admins"})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = identity.AddToRequestCtx(id, req)
		rr := httptest.NewRecorder()

		next.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rr.Code)
		}
	})
}
