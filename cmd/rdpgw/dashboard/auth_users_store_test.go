package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileAuthUserStoreRoundTrip(t *testing.T) {
	t.Parallel()

	store, err := NewFileAuthUserStore(filepath.Join(t.TempDir(), "auth-users.json"))
	if err != nil {
		t.Fatalf("new auth user store: %v", err)
	}

	user := AuthUser{
		Username: "alice",
		Password: "secret",
		Enabled:  true,
	}
	if err := store.Put(user); err != nil {
		t.Fatalf("put auth user: %v", err)
	}

	loaded, err := store.Get("alice")
	if err != nil {
		t.Fatalf("get auth user: %v", err)
	}
	if loaded.Username != "alice" || loaded.Password != "secret" || !loaded.Enabled {
		t.Fatalf("unexpected loaded user: %+v", loaded)
	}

	listed, err := store.List()
	if err != nil {
		t.Fatalf("list auth users: %v", err)
	}
	if len(listed) != 1 || listed[0].Username != "alice" {
		t.Fatalf("unexpected listed users: %+v", listed)
	}

	if err := store.Delete("alice"); err != nil {
		t.Fatalf("delete auth user: %v", err)
	}
	if _, err := store.Get("alice"); err != ErrAuthUserNotFound {
		t.Fatalf("expected ErrAuthUserNotFound, got %v", err)
	}
}

func TestWriteAuthHelperConfigOmitsDisabledUsers(t *testing.T) {
	t.Parallel()

	outputPath := filepath.Join(t.TempDir(), "rdpgw-auth.yaml")
	err := WriteAuthHelperConfig(outputPath, []AuthUser{
		{Username: "alice", Password: "secret", Enabled: true},
		{Username: "bob", Password: "disabled", Enabled: false},
	})
	if err != nil {
		t.Fatalf("write auth helper config: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read helper config: %v", err)
	}

	text := string(data)
	if !strings.Contains(text, "Users:") {
		t.Fatalf("expected Users section, got %q", text)
	}
	if !strings.Contains(text, "Username: alice") || !strings.Contains(text, "Password: secret") {
		t.Fatalf("expected enabled user in helper config, got %q", text)
	}
	if strings.Contains(text, "bob") || strings.Contains(text, "disabled") {
		t.Fatalf("expected disabled user to be omitted, got %q", text)
	}
}
