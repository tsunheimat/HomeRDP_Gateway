package web

import (
	"context"
	"fmt"
	"net"
	"strings"
)

func lookupIPWithDNSServer(ctx context.Context, dnsServer string, host string) ([]net.IP, error) {
	dnsServer = normalizeDNSServerAddress(dnsServer)
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := &net.Dialer{}
			return dialer.DialContext(ctx, network, dnsServer)
		},
	}
	return resolver.LookupIP(ctx, "ip", host)
}

func normalizeDNSServerAddress(dnsServer string) string {
	dnsServer = strings.TrimSpace(dnsServer)
	if dnsServer == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(dnsServer); err == nil {
		return dnsServer
	}
	if ip := net.ParseIP(dnsServer); ip != nil {
		return net.JoinHostPort(dnsServer, "53")
	}
	if strings.Contains(dnsServer, ":") && !strings.HasPrefix(dnsServer, "[") {
		return net.JoinHostPort(dnsServer, "53")
	}
	return net.JoinHostPort(dnsServer, "53")
}

func (h *Handler) resolveRDPAddress(ctx context.Context, address string, targetIPOverride string, forceTargetIPOverride bool) (string, error) {
	host, port, hasPort, err := splitRDPAddress(address)
	if err != nil {
		return "", err
	}
	if net.ParseIP(host) != nil {
		return address, nil
	}

	targetIPOverride = strings.TrimSpace(targetIPOverride)
	if targetIPOverride != "" && net.ParseIP(targetIPOverride) == nil {
		return "", fmt.Errorf("target IP override must be an IP literal")
	}

	if forceTargetIPOverride && targetIPOverride != "" {
		return joinRDPAddress(targetIPOverride, port, hasPort), nil
	}

	dnsServer := normalizeDNSServerAddress(h.internalDNSServer)
	if !matchesInternalDomain(host, h.internalDomains) || dnsServer == "" {
		return address, nil
	}

	ips, err := h.internalDNSLookupIP(ctx, dnsServer, host)
	if err == nil && len(ips) > 0 {
		return joinRDPAddress(ips[0].String(), port, hasPort), nil
	}
	if targetIPOverride != "" {
		return joinRDPAddress(targetIPOverride, port, hasPort), nil
	}
	if err != nil {
		return "", fmt.Errorf("internal DNS lookup failed for %q: %w", host, err)
	}
	return "", fmt.Errorf("internal DNS lookup returned no addresses for %q", host)
}

func splitRDPAddress(address string) (host string, port string, hasPort bool, err error) {
	address = strings.TrimSpace(address)
	if strings.Contains(address, ":") {
		host, port, err = net.SplitHostPort(address)
		if err != nil {
			return "", "", false, err
		}
		return strings.Trim(host, "[]"), port, true, nil
	}
	return address, "", false, nil
}

func joinRDPAddress(host string, port string, hasPort bool) string {
	if !hasPort {
		return host
	}
	return net.JoinHostPort(host, port)
}

func matchesInternalDomain(host string, domains []string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return false
	}
	for _, domain := range domains {
		domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
		domain = strings.TrimPrefix(domain, ".")
		if domain == "" {
			continue
		}
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}
