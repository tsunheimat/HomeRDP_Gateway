//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupUIDRejectsNegativeNumericID(t *testing.T) {
	if _, err := lookupUID("-1"); err == nil {
		t.Fatalf("expected lookupUID to reject negative numeric UID")
	}
}

func TestLookupGIDRejectsNegativeNumericID(t *testing.T) {
	if _, err := lookupGID("-1"); err == nil {
		t.Fatalf("expected lookupGID to reject negative numeric GID")
	}
}

func TestAuthSocketIsOwnerOnlyByDefault(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "auth")
	socket := filepath.Join(dir, "rdpgw-auth.sock")

	listener, err := listenUnixSocket(socket)
	if err != nil {
		t.Fatalf("listenUnixSocket returned error: %v", err)
	}
	defer listener.Close()

	info, err := os.Stat(socket)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected auth socket mode 0600, got %04o", got)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("expected auth socket not to be group/world accessible, got %04o", info.Mode().Perm())
	}
}

func TestAuthSocketAllowsConfiguredGroupMode(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "auth")
	socket := filepath.Join(dir, "rdpgw-auth.sock")

	listener, err := listenUnixSocketWithOptions(socket, unixSocketOptions{
		Mode:    0660,
		DirMode: 0750,
	})
	if err != nil {
		t.Fatalf("listenUnixSocketWithOptions returned error: %v", err)
	}
	defer listener.Close()

	info, err := os.Stat(socket)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if got := info.Mode().Perm(); got != 0660 {
		t.Fatalf("expected auth socket mode 0660, got %04o", got)
	}
	if info.Mode().Perm()&0007 != 0 {
		t.Fatalf("expected auth socket not to be world accessible, got %04o", info.Mode().Perm())
	}
}

func TestAuthSocketParentDirCreatedOwnerOnlyByDefault(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "auth")
	socket := filepath.Join(dir, "rdpgw-auth.sock")

	listener, err := listenUnixSocket(socket)
	if err != nil {
		t.Fatalf("listenUnixSocket returned error: %v", err)
	}
	defer listener.Close()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat socket parent dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", dir)
	}
	if got := info.Mode().Perm(); got != 0700 {
		t.Fatalf("expected socket parent dir mode 0700, got %04o", got)
	}
}

func TestAuthSocketParentDirAllowsConfiguredGroupMode(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "auth")
	socket := filepath.Join(dir, "rdpgw-auth.sock")

	listener, err := listenUnixSocketWithOptions(socket, unixSocketOptions{
		Mode:    0660,
		DirMode: 0750,
	})
	if err != nil {
		t.Fatalf("listenUnixSocketWithOptions returned error: %v", err)
	}
	defer listener.Close()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat socket parent dir: %v", err)
	}
	if got := info.Mode().Perm(); got != 0750 {
		t.Fatalf("expected socket parent dir mode 0750, got %04o", got)
	}
}

func TestAuthSocketRestoresUmask(t *testing.T) {
	oldUmask := setProcessUmask(0022)
	defer setProcessUmask(oldUmask)

	socket := filepath.Join(t.TempDir(), "rdpgw-auth.sock")
	listener, err := listenUnixSocket(socket)
	if err != nil {
		t.Fatalf("listenUnixSocket returned error: %v", err)
	}
	defer listener.Close()

	if currentUmask := getProcessUmask(); currentUmask != 0022 {
		t.Fatalf("expected umask to be restored to 0022, got %04o", currentUmask)
	}
}

func TestAuthSocketRefusesToRemoveNonSocketPath(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "rdpgw-auth.sock")
	if err := os.WriteFile(socket, []byte("keep"), 0600); err != nil {
		t.Fatalf("create non-socket file: %v", err)
	}

	listener, err := listenUnixSocket(socket)
	if err == nil {
		listener.Close()
		t.Fatalf("expected listenUnixSocket to reject non-socket path")
	}

	contents, err := os.ReadFile(socket)
	if err != nil {
		t.Fatalf("read non-socket file after listenUnixSocket: %v", err)
	}
	if string(contents) != "keep" {
		t.Fatalf("expected non-socket file contents to remain unchanged, got %q", string(contents))
	}
}

func TestAuthSocketRefusesSymlinkPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	socket := filepath.Join(dir, "rdpgw-auth.sock")
	if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
		t.Fatalf("create symlink target: %v", err)
	}
	if err := os.Symlink(target, socket); err != nil {
		t.Fatalf("create auth socket symlink: %v", err)
	}

	listener, err := listenUnixSocket(socket)
	if err == nil {
		listener.Close()
		t.Fatalf("expected listenUnixSocket to reject symlink path")
	}

	contents, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read symlink target after listenUnixSocket: %v", err)
	}
	if string(contents) != "keep" {
		t.Fatalf("expected symlink target contents to remain unchanged, got %q", string(contents))
	}
}

func TestAuthSocketOwnershipRefusesSymlinkPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	socket := filepath.Join(dir, "rdpgw-auth.sock")
	if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
		t.Fatalf("create symlink target: %v", err)
	}
	if err := os.Symlink(target, socket); err != nil {
		t.Fatalf("create auth socket symlink: %v", err)
	}

	if err := applyUnixSocketOwnership(socket, "0", ""); err == nil {
		t.Fatalf("expected applyUnixSocketOwnership to reject symlink path")
	}
}
