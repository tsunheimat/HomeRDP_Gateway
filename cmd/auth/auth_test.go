package main

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/auth/config"
	"github.com/bolkedebruin/rdpgw/cmd/auth/database"
	"github.com/bolkedebruin/rdpgw/shared/auth"
)

func TestAuthenticateUsesConfiguredUsers(t *testing.T) {
	service := NewAuthService(database.NewConfig([]config.UserConfig{
		{Username: "alice", Password: "secret"},
	}))

	res, err := service.Authenticate(context.Background(), &auth.UserPass{
		Username: "alice",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if !res.Authenticated {
		t.Fatalf("expected configured user to authenticate, got %+v", res)
	}
}

func TestAuthenticateRejectsWrongPassword(t *testing.T) {
	service := NewAuthService(database.NewConfig([]config.UserConfig{
		{Username: "alice", Password: "secret"},
	}))

	res, err := service.Authenticate(context.Background(), &auth.UserPass{
		Username: "alice",
		Password: "wrong",
	})
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if res.Authenticated {
		t.Fatalf("expected wrong password to be rejected, got %+v", res)
	}
}

func TestAuthSocketIsOwnerOnly(t *testing.T) {
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

func TestAuthSocketParentDirCreatedOwnerOnly(t *testing.T) {
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

func TestAuthSocketRestoresUmask(t *testing.T) {
	oldUmask := syscall.Umask(0022)
	defer syscall.Umask(oldUmask)

	socket := filepath.Join(t.TempDir(), "rdpgw-auth.sock")
	listener, err := listenUnixSocket(socket)
	if err != nil {
		t.Fatalf("listenUnixSocket returned error: %v", err)
	}
	defer listener.Close()

	currentUmask := syscall.Umask(oldUmask)
	if currentUmask != 0022 {
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
