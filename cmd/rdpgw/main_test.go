package main

import (
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/config"
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

	if err := validateManagedDirectAuthConfig(cfg); err == nil {
		t.Fatal("expected direct auth without openid to fail validation")
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
