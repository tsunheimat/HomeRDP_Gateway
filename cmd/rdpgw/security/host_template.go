package security

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func ResolvePreferredUsernameHost(templateHost, username string) (string, error) {
	return ValidateRenderedHost(strings.Replace(templateHost, "{{ preferred_username }}", username, 1))
}

func ValidateRenderedHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("target host is required")
	}
	if strings.ContainsAny(host, "\r\n\t") {
		return "", fmt.Errorf("target host contains invalid whitespace")
	}

	if strings.Contains(host, ":") {
		renderedHost, port, err := net.SplitHostPort(host)
		if err != nil {
			return "", fmt.Errorf("invalid target host %q", host)
		}
		if strings.TrimSpace(renderedHost) == "" {
			return "", fmt.Errorf("invalid target host %q", host)
		}
		portNum, err := strconv.Atoi(port)
		if err != nil || portNum < 1 || portNum > 65535 {
			return "", fmt.Errorf("invalid target host %q", host)
		}
	}

	return host, nil
}
