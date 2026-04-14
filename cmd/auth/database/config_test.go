package database

import (
	"github.com/bolkedebruin/rdpgw/cmd/auth/config"
	"os"
	"path/filepath"
	"testing"
)

func createTestDatabase() Database {
	var users = []config.UserConfig{}

	user1 := config.UserConfig{}
	user1.Username = "my_username"
	user1.Password = "my_password"
	users = append(users, user1)

	user2 := config.UserConfig{}
	user2.Username = "my_username2"
	user2.Password = "my_password2"
	users = append(users, user2)

	config := NewConfig(users)

	return config
}

func TestDatabaseConfigValidUsername(t *testing.T) {
	database := createTestDatabase()

	if database.GetPassword("my_username") != "my_password" {
		t.Fatalf("Wrong password returned")
	}
	if database.GetPassword("my_username2") != "my_password2" {
		t.Fatalf("Wrong password returned")
	}
}

func TestDatabaseInvalidUsername(t *testing.T) {
	database := createTestDatabase()

	if database.GetPassword("my_invalid_username") != "" {
		t.Fatalf("Non empty password returned for invalid username")
	}
}

func TestDatabaseReloadsUsersFromConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rdpgw-auth.yaml")
	if err := os.WriteFile(path, []byte("Users:\n  - Username: alice\n    Password: first\n"), 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	database, err := NewConfigFile(path)
	if err != nil {
		t.Fatalf("new config file database: %v", err)
	}

	if got := database.GetPassword("alice"); got != "first" {
		t.Fatalf("password = %q, want %q", got, "first")
	}

	if err := os.WriteFile(path, []byte("Users:\n  - Username: alice\n    Password: second\n"), 0o600); err != nil {
		t.Fatalf("write updated config: %v", err)
	}

	if got := database.GetPassword("alice"); got != "second" {
		t.Fatalf("password after reload = %q, want %q", got, "second")
	}
}

func TestDatabaseKeepsLastGoodConfigOnMalformedReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rdpgw-auth.yaml")
	if err := os.WriteFile(path, []byte("Users:\n  - Username: alice\n    Password: first\n"), 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	database, err := NewConfigFile(path)
	if err != nil {
		t.Fatalf("new config file database: %v", err)
	}

	if got := database.GetPassword("alice"); got != "first" {
		t.Fatalf("password = %q, want %q", got, "first")
	}

	if err := os.WriteFile(path, []byte("Users:\n  - Username: alice\n    Password: [\n"), 0o600); err != nil {
		t.Fatalf("write malformed config: %v", err)
	}

	if got := database.GetPassword("alice"); got != "first" {
		t.Fatalf("password after malformed reload = %q, want %q", got, "first")
	}
}
