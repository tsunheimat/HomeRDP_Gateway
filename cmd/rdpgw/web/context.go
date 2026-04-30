package web

import (
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/jcmturner/goidentity/v6"
	"log"
	"net/http"
)

func EnrichContext(next http.Handler) http.Handler {
	checker, _ := NewTrustedProxyChecker(nil)
	return enrichContextWithTrustedProxies(checker)(next)
}

func EnrichContextWithTrustedProxyCIDRs(cidrs []string) (func(http.Handler) http.Handler, error) {
	checker, err := NewTrustedProxyChecker(cidrs)
	if err != nil {
		return nil, err
	}
	return enrichContextWithTrustedProxies(checker), nil
}

func enrichContextWithTrustedProxies(trustedProxies *TrustedProxyChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := GetSessionIdentity(r)
			if err != nil {
				log.Printf("failed to get session identity: %v", err)
				http.Error(w, "authentication failed", http.StatusInternalServerError)
				return
			}

			if id == nil {
				id = identity.NewUser()
			}
			if id.Authenticated() && id.GetAttribute(identity.AttrAuthSource) == identity.AuthSourceHeader && !trustedProxies.IsTrustedRemoteAddr(r.RemoteAddr) {
				http.Error(w, "Header authentication requires a trusted proxy", http.StatusUnauthorized)
				return
			}

			log.Printf("Identity SessionId: %s, UserName: %s: Authenticated: %t",
				id.SessionId(), id.UserName(), id.Authenticated())

			clientIP, proxies := trustedProxies.ClientIPAndProxies(r)
			id.SetAttribute(identity.AttrClientIp, clientIP)
			id.SetAttribute(identity.AttrProxies, proxies)
			id.SetAttribute(identity.AttrRemoteAddr, r.RemoteAddr)

			next.ServeHTTP(w, identity.AddToRequestCtx(id, r))
		})
	}
}

func TransposeSPNEGOContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gid := goidentity.FromHTTPRequestContext(r)
		if gid != nil {
			id := identity.FromRequestCtx(r)
			id.SetUserName(gid.UserName())
			id.SetAuthenticated(gid.Authenticated())
			id.SetDomain(gid.Domain())
			id.SetAuthTime(gid.AuthTime())
			r = identity.AddToRequestCtx(id, r)
		}
		next.ServeHTTP(w, r)
	})
}
