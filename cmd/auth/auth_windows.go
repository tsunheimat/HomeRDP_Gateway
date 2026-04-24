//go:build windows

package main

import (
	"fmt"
	"net"
)

func listenWithUmask(_ int, listen func() (net.Listener, error)) (net.Listener, error) {
	return listen()
}

func applyUnixSocketOwnership(_ string, owner, group string) error {
	if owner != "" || group != "" {
		return fmt.Errorf("auth socket owner/group options are not supported on windows")
	}
	return nil
}

func ensureUnixSocketPath(_, _ string) error {
	return nil
}
