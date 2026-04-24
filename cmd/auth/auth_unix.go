//go:build !windows

package main

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
	"sync"
	"syscall"
)

var umaskMu sync.Mutex

func listenWithUmask(mask int, listen func() (net.Listener, error)) (net.Listener, error) {
	umaskMu.Lock()
	defer umaskMu.Unlock()

	oldUmask := syscall.Umask(mask)
	defer syscall.Umask(oldUmask)

	return listen()
}

func applyUnixSocketOwnership(path, owner, group string) error {
	uid := -1
	gid := -1

	if err := ensureUnixSocketPath(path, "set auth socket ownership"); err != nil {
		return err
	}

	if owner != "" {
		parsedUID, err := lookupUID(owner)
		if err != nil {
			return err
		}
		uid = parsedUID
	}
	if group != "" {
		parsedGID, err := lookupGID(group)
		if err != nil {
			return err
		}
		gid = parsedGID
	}
	if uid == -1 && gid == -1 {
		return nil
	}
	if err := os.Lchown(path, uid, gid); err != nil {
		return fmt.Errorf("set auth socket ownership: %w", err)
	}
	if err := ensureUnixSocketPath(path, "verify auth socket ownership target"); err != nil {
		return err
	}
	return nil
}

func lookupUID(owner string) (int, error) {
	if uid, err := strconv.Atoi(owner); err == nil {
		if uid < 0 {
			return 0, fmt.Errorf("auth socket owner UID %q must not be negative", owner)
		}
		return uid, nil
	}
	u, err := user.Lookup(owner)
	if err != nil {
		return 0, fmt.Errorf("lookup auth socket owner %q: %w", owner, err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, fmt.Errorf("parse UID %q for auth socket owner %q: %w", u.Uid, owner, err)
	}
	if uid < 0 {
		return 0, fmt.Errorf("UID %q for auth socket owner %q must not be negative", u.Uid, owner)
	}
	return uid, nil
}

func lookupGID(group string) (int, error) {
	if gid, err := strconv.Atoi(group); err == nil {
		if gid < 0 {
			return 0, fmt.Errorf("auth socket group GID %q must not be negative", group)
		}
		return gid, nil
	}
	g, err := user.LookupGroup(group)
	if err != nil {
		return 0, fmt.Errorf("lookup auth socket group %q: %w", group, err)
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return 0, fmt.Errorf("parse GID %q for auth socket group %q: %w", g.Gid, group, err)
	}
	if gid < 0 {
		return 0, fmt.Errorf("GID %q for auth socket group %q must not be negative", g.Gid, group)
	}
	return gid, nil
}

func ensureUnixSocketPath(path, action string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s: stat auth socket: %w", action, err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s: refusing to follow non-socket auth socket path %q", action, path)
	}
	return nil
}

func setProcessUmask(mask int) int {
	umaskMu.Lock()
	defer umaskMu.Unlock()
	return syscall.Umask(mask)
}

func getProcessUmask() int {
	umaskMu.Lock()
	defer umaskMu.Unlock()
	oldUmask := syscall.Umask(0)
	syscall.Umask(oldUmask)
	return oldUmask
}
