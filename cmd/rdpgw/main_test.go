package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/config"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/security"
)

func TestAuthHelperConfigPathUsesDashboardSettingByDefault(t *testing.T) {
	t.Setenv("RDPGW_AUTH_HELPER_CONFIG", "")

	cfg := config.Configuration{
		Dashboard: config.DashboardConfig{
			AuthHelperConfigPath: "/var/lib/rdpgw/dashboard/rdpgw-auth.yaml",
		},
	}

	if got := authHelperConfigPath(cfg); got != "/var/lib/rdpgw/dashboard/rdpgw-auth.yaml" {
		t.Fatalf("auth helper config path = %q", got)
	}
}

func TestAuthHelperConfigPathHonorsRuntimeOverride(t *testing.T) {
	t.Setenv("RDPGW_AUTH_HELPER_CONFIG", "/tmp/runtime-auth.yaml")

	cfg := config.Configuration{
		Dashboard: config.DashboardConfig{
			AuthHelperConfigPath: "/var/lib/rdpgw/dashboard/rdpgw-auth.yaml",
		},
	}

	if got := authHelperConfigPath(cfg); got != "/tmp/runtime-auth.yaml" {
		t.Fatalf("auth helper config path = %q", got)
	}
}

func TestValidateManagedDirectAuthConfigRequiresOpenID(t *testing.T) {
	cfg := config.Configuration{
		Server: config.ServerConfig{
			Authentication: []string{config.AuthenticationBasic},
		},
	}

	if err := validateManagedDirectAuthConfig(cfg); err != nil {
		t.Fatalf("expected shared-dashboard direct auth to validate without openid, got %v", err)
	}
}

func TestValidateManagedDirectAuthConfigAllowsOpenIDManagedDirectAuth(t *testing.T) {
	cfg := config.Configuration{
		Server: config.ServerConfig{
			Authentication: []string{config.AuthenticationOpenId, config.AuthenticationBasic, "ntlm"},
		},
	}

	if err := validateManagedDirectAuthConfig(cfg); err != nil {
		t.Fatalf("expected openid-managed direct auth to validate, got %v", err)
	}
}

func TestInitDashboardStateLoadsManagedHostsWithoutOpenID(t *testing.T) {
	t.Cleanup(func() {
		security.ManagedHostList = nil
	})

	tmpDir := t.TempDir()
	cfg := config.Configuration{
		Dashboard: config.DashboardConfig{
			StorePath:     filepath.Join(tmpDir, "dashboard"),
			UploadDir:     filepath.Join(tmpDir, "dashboard", "uploads"),
			AuthUsersPath: filepath.Join(tmpDir, "dashboard", "auth-users.json"),
		},
	}

	store, err := dashboard.NewFileStore(cfg.Dashboard.StorePath, cfg.Dashboard.UploadDir)
	if err != nil {
		t.Fatalf("create dashboard store: %v", err)
	}

	err = store.Put(dashboard.Entry{
		ID:            "host-1",
		Type:          dashboard.EntryTypeHost,
		Name:          "Managed Host",
		AllowedGroups: []string{"ops"},
		Enabled:       true,
		Host:          "managed.internal:3389",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("write dashboard entry: %v", err)
	}

	helperConfigPath := filepath.Join(tmpDir, "dashboard", "rdpgw-auth.yaml")
	dashboardStore, authUserStore, err := initDashboardState(cfg, helperConfigPath)
	if err != nil {
		t.Fatalf("init dashboard state: %v", err)
	}
	if dashboardStore == nil {
		t.Fatal("expected dashboard store to be initialized")
	}
	if authUserStore != nil {
		t.Fatal("expected auth user store to stay disabled without openid")
	}
	if security.ManagedHostList == nil {
		t.Fatal("expected managed host list reader to be initialized")
	}

	hosts, err := security.ManagedHostList()
	if err != nil {
		t.Fatalf("list managed hosts: %v", err)
	}
	if len(hosts) != 1 || hosts[0] != "managed.internal:3389" {
		t.Fatalf("managed hosts = %v", hosts)
	}

	if _, err := os.Stat(helperConfigPath); !os.IsNotExist(err) {
		t.Fatalf("expected helper config to remain untouched without openid, stat err=%v", err)
	}
}
