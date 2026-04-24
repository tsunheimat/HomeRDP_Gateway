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
	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("set auth socket ownership: %w", err)
	}
	return nil
}

func lookupUID(owner string) (int, error) {
	if uid, err := strconv.Atoi(owner); err == nil {
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
	return uid, nil
}

func lookupGID(group string) (int, error) {
	if gid, err := strconv.Atoi(group); err == nil {
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
	return gid, nil
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
