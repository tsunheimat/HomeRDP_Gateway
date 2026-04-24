package web

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

type TrustedProxyChecker struct {
	networks []*net.IPNet
}

func NewTrustedProxyChecker(cidrs []string) (*TrustedProxyChecker, error) {
	checker := &TrustedProxyChecker{}
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", cidr, err)
		}
		checker.networks = append(checker.networks, network)
	}
	return checker, nil
}

func (c *TrustedProxyChecker) HasTrustedProxies() bool {
	return c != nil && len(c.networks) > 0
}

func (c *TrustedProxyChecker) IsTrustedRemoteAddr(remoteAddr string) bool {
	if !c.HasTrustedProxies() {
		return false
	}

	remoteIP := parseRemoteIP(remoteAddr)
	if remoteIP == nil {
		return false
	}

	for _, network := range c.networks {
		if network.Contains(remoteIP) {
			return true
		}
	}
	return false
}

func (c *TrustedProxyChecker) ClientIPAndProxies(r *http.Request) (string, []string) {
	remoteIP := remoteIPString(r.RemoteAddr)
	if remoteIP == "" {
		remoteIP = r.RemoteAddr
	}

	if !c.IsTrustedRemoteAddr(r.RemoteAddr) {
		return remoteIP, nil
	}

	clientIP, proxies, ok := firstValidForwardedFor(r.Header.Get("X-Forwarded-For"))
	if !ok {
		return remoteIP, nil
	}
	return clientIP, proxies
}

func firstValidForwardedFor(header string) (string, []string, bool) {
	if header == "" {
		return "", nil, false
	}

	parts := strings.Split(header, ",")
	validIPs := make([]string, 0, len(parts))
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			return "", nil, false
		}
		ip := net.ParseIP(candidate)
		if ip == nil {
			return "", nil, false
		}
		validIPs = append(validIPs, ip.String())
	}
	if len(validIPs) == 0 {
		return "", nil, false
	}
	return validIPs[0], validIPs[1:], true
}

func parseRemoteIP(remoteAddr string) net.IP {
	return net.ParseIP(remoteIPString(remoteAddr))
}

func remoteIPString(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	if ip := net.ParseIP(remoteAddr); ip != nil {
		return ip.String()
	}
	return ""
}
