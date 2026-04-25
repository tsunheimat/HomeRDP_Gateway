package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func unsetEnvWithCleanup(t *testing.T, key string) {
	t.Helper()
	value, ok := os.LookupEnv(key)
	if ok {
		t.Cleanup(func() {
			if err := os.Setenv(key, value); err != nil {
				t.Fatalf("failed to restore %s: %v", key, err)
			}
		})
	} else {
		t.Cleanup(func() {
			if err := os.Unsetenv(key); err != nil {
				t.Fatalf("failed to unset %s: %v", key, err)
			}
		})
	}

	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to clear %s: %v", key, err)
	}
}

func TestHeaderEnabled(t *testing.T) {
	cases := []struct {
		name           string
		authentication []string
		expected       bool
	}{
		{
			name:           "header_enabled",
			authentication: []string{"header"},
			expected:       true,
		},
		{
			name:           "header_with_others",
			authentication: []string{"openid", "header", "local"},
			expected:       true,
		},
		{
			name:           "header_not_enabled",
			authentication: []string{"openid", "local"},
			expected:       false,
		},
		{
			name:           "empty_authentication",
			authentication: []string{},
			expected:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := &ServerConfig{
				Authentication: tc.authentication,
			}

			result := config.HeaderEnabled()
			if result != tc.expected {
				t.Errorf("expected HeaderEnabled(): %v, got: %v", tc.expected, result)
			}
		})
	}
}

func TestAuthenticationConstants(t *testing.T) {
	// Test that the header authentication constant is correct
	if AuthenticationHeader != "header" {
		t.Errorf("incorrect authentication header constant: %v", AuthenticationHeader)
	}
}

func TestHeaderConfigValidation(t *testing.T) {
	cases := []struct {
		name        string
		headerConf  HeaderConfig
		shouldError bool
	}{
		{
			name: "valid_config",
			headerConf: HeaderConfig{
				UserHeader: "X-Forwarded-User",
			},
			shouldError: false,
		},
		{
			name: "missing_user_header",
			headerConf: HeaderConfig{
				EmailHeader: "X-Forwarded-Email",
			},
			shouldError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the configuration struct
			if tc.headerConf.UserHeader == "" && !tc.shouldError {
				t.Error("expected user header to be set")
			}
			if tc.headerConf.UserHeader != "" && tc.shouldError {
				t.Error("expected configuration to be invalid")
			}
		})
	}
}

func TestConfigurationValidateRejectsHeaderAuthWithoutTrustedProxyCIDRs(t *testing.T) {
	cfg := Configuration{
		Server: ServerConfig{
			Authentication: []string{AuthenticationHeader},
		},
		Header: HeaderConfig{
			UserHeader: "X-Forwarded-User",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected header auth without trusted proxy CIDRs to be invalid")
	}
}

func TestConfigurationValidateAcceptsHeaderAuthWithTrustedProxyCIDRs(t *testing.T) {
	cfg := Configuration{
		Server: ServerConfig{
			Authentication:       []string{AuthenticationHeader},
			TrustedProxyCIDRs:    []string{"10.0.0.0/24"},
			SessionKey:           "testsessionkeytestsessionkey1234",
			SessionEncryptionKey: "testencryptionkeytestencrypt1234",
		},
		Header: HeaderConfig{
			UserHeader: "X-Forwarded-User",
		},
		Caps: CapsConfigWithTokenAuth(),
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid header auth config, got %v", err)
	}
}

func TestConfigurationValidateRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	cfg := Configuration{
		Server: ServerConfig{
			Authentication:    []string{AuthenticationOpenId},
			TrustedProxyCIDRs: []string{"not-a-cidr"},
		},
		Caps: CapsConfigWithTokenAuth(),
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid trusted proxy CIDR to be rejected")
	}
}

func TestConfigurationValidateSignedHostSelectionQueryTokenSigningKey(t *testing.T) {
	cases := []struct {
		name        string
		selection   string
		key         string
		shouldError bool
	}{
		{
			name:        "signed empty key rejected",
			selection:   HostSelectionSigned,
			key:         "",
			shouldError: true,
		},
		{
			name:        "signed short key rejected",
			selection:   HostSelectionSigned,
			key:         "short-query-token-signing-key",
			shouldError: true,
		},
		{
			name:        "signed 32 byte key accepted",
			selection:   HostSelectionSigned,
			key:         "12345678901234567890123456789012",
			shouldError: false,
		},
		{
			name:        "non signed host selection unaffected",
			selection:   HostSelectionRoundRobin,
			key:         "",
			shouldError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Configuration{
				Server: ServerConfig{
					HostSelection: tc.selection,
				},
				Security: SecurityConfig{
					QueryTokenSigningKey: tc.key,
				},
			}

			err := cfg.Validate()
			if tc.shouldError && err == nil {
				t.Fatal("expected config validation to fail")
			}
			if !tc.shouldError && err != nil {
				t.Fatalf("expected config validation to pass, got %v", err)
			}
		})
	}
}

func TestServerSecureCookiesDefaultsFalse(t *testing.T) {
	unsetEnvWithCleanup(t, "RDPGW_SERVER__SECURECOOKIES")

	cfg := Load("/definitely-missing.yaml")
	if cfg.Server.SecureCookies {
		t.Fatal("expected Server.SecureCookies to default to false")
	}
}

func TestServerAllowTLSKeyLogDefaultsFalse(t *testing.T) {
	unsetEnvWithCleanup(t, "RDPGW_SERVER__ALLOWTLSKEYLOG")

	cfg := Load("/definitely-missing.yaml")
	if cfg.Server.AllowTLSKeyLog {
		t.Fatal("expected Server.AllowTLSKeyLog to default to false")
	}
}

func TestServerAllowTLSKeyLogCanBeEnabledFromEnv(t *testing.T) {
	t.Setenv("RDPGW_SERVER__ALLOWTLSKEYLOG", "true")

	cfg := Load("/definitely-missing.yaml")
	if !cfg.Server.AllowTLSKeyLog {
		t.Fatal("expected Server.AllowTLSKeyLog to be enabled from environment")
	}
}

func TestServerAllowTLSKeyLogCanBeEnabledFromFile(t *testing.T) {
	unsetEnvWithCleanup(t, "RDPGW_SERVER__ALLOWTLSKEYLOG")
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	configBody := "Server:\n  AllowTLSKeyLog: true\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := Load(configPath)
	if !cfg.Server.AllowTLSKeyLog {
		t.Fatal("expected Server.AllowTLSKeyLog to be enabled from config file")
	}
}

func TestServerAllowTLSKeyLogDoesNotPersistAcrossLoad(t *testing.T) {
	unsetEnvWithCleanup(t, "RDPGW_SERVER__ALLOWTLSKEYLOG")
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	configBody := "Server:\n  AllowTLSKeyLog: true\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := Load(configPath)
	if !cfg.Server.AllowTLSKeyLog {
		t.Fatal("expected Server.AllowTLSKeyLog to be enabled from config file")
	}

	cfg = Load("/definitely-missing.yaml")
	if cfg.Server.AllowTLSKeyLog {
		t.Fatal("expected Server.AllowTLSKeyLog to reset to the false default when absent")
	}
}

func CapsConfigWithTokenAuth() RDGCapsConfig {
	return RDGCapsConfig{TokenAuth: true}
}

func TestLoadDashboardSettings(t *testing.T) {
	unsetEnvWithCleanup(t, "RDPGW_OPENID__GROUPSCLAIM")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__STOREPATH")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__UPLOADDIR")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__ICONDIR")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__AUTHUSERSPATH")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__ADMINGROUPS")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__MAXUPLOADSIZEMB")

	cfg := Load("/definitely-missing.yaml")

	if cfg.OpenId.GroupsClaim != "groups" {
		t.Fatalf("expected default OpenId.GroupsClaim to be groups, got %q", cfg.OpenId.GroupsClaim)
	}

	if cfg.Dashboard.StorePath != "./data/dashboard" {
		t.Fatalf("expected default Dashboard.StorePath to be ./data/dashboard, got %q", cfg.Dashboard.StorePath)
	}

	if cfg.Dashboard.UploadDir != "./data/dashboard/uploads" {
		t.Fatalf("expected default Dashboard.UploadDir to be ./data/dashboard/uploads, got %q", cfg.Dashboard.UploadDir)
	}
	if cfg.Dashboard.IconDir != "./data/dashboard/icons" {
		t.Fatalf("expected default Dashboard.IconDir to be ./data/dashboard/icons, got %q", cfg.Dashboard.IconDir)
	}
	if cfg.Dashboard.AuthUsersPath != "./data/dashboard/auth-users.json" {
		t.Fatalf("expected default Dashboard.AuthUsersPath to be ./data/dashboard/auth-users.json, got %q", cfg.Dashboard.AuthUsersPath)
	}
	if cfg.Dashboard.AuthHelperConfigPath != "./data/dashboard/rdpgw-auth.yaml" {
		t.Fatalf("expected default Dashboard.AuthHelperConfigPath to be ./data/dashboard/rdpgw-auth.yaml, got %q", cfg.Dashboard.AuthHelperConfigPath)
	}

	if cfg.Dashboard.MaxUploadSizeMb != 5 {
		t.Fatalf("expected default Dashboard.MaxUploadSizeMb to be 5, got %d", cfg.Dashboard.MaxUploadSizeMb)
	}

	t.Setenv("RDPGW_OPENID__GROUPSCLAIM", "ak_groups")
	t.Setenv("RDPGW_DASHBOARD__STOREPATH", "/tmp/rdpgw-dashboard")
	t.Setenv("RDPGW_DASHBOARD__UPLOADDIR", "/tmp/rdpgw-dashboard/uploads")
	t.Setenv("RDPGW_DASHBOARD__ICONDIR", "/tmp/rdpgw-dashboard/icons")
	t.Setenv("RDPGW_DASHBOARD__AUTHUSERSPATH", "/tmp/rdpgw-dashboard/auth-users.json")
	t.Setenv("RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH", "/tmp/rdpgw-dashboard/rdpgw-auth.yaml")
	t.Setenv("RDPGW_DASHBOARD__ADMINGROUPS", "rdpgw-admins homelab-admins")
	t.Setenv("RDPGW_DASHBOARD__MAXUPLOADSIZEMB", "7")

	cfg = Load("/definitely-missing.yaml")

	if cfg.OpenId.GroupsClaim != "ak_groups" {
		t.Fatalf("expected OpenId.GroupsClaim to be ak_groups, got %q", cfg.OpenId.GroupsClaim)
	}

	if cfg.Dashboard.StorePath != "/tmp/rdpgw-dashboard" {
		t.Fatalf("expected Dashboard.StorePath to be /tmp/rdpgw-dashboard, got %q", cfg.Dashboard.StorePath)
	}

	if cfg.Dashboard.UploadDir != "/tmp/rdpgw-dashboard/uploads" {
		t.Fatalf("expected Dashboard.UploadDir to be /tmp/rdpgw-dashboard/uploads, got %q", cfg.Dashboard.UploadDir)
	}
	if cfg.Dashboard.IconDir != "/tmp/rdpgw-dashboard/icons" {
		t.Fatalf("expected Dashboard.IconDir to be /tmp/rdpgw-dashboard/icons, got %q", cfg.Dashboard.IconDir)
	}
	if cfg.Dashboard.AuthUsersPath != "/tmp/rdpgw-dashboard/auth-users.json" {
		t.Fatalf("expected Dashboard.AuthUsersPath to be /tmp/rdpgw-dashboard/auth-users.json, got %q", cfg.Dashboard.AuthUsersPath)
	}
	if cfg.Dashboard.AuthHelperConfigPath != "/tmp/rdpgw-dashboard/rdpgw-auth.yaml" {
		t.Fatalf("expected Dashboard.AuthHelperConfigPath to be /tmp/rdpgw-dashboard/rdpgw-auth.yaml, got %q", cfg.Dashboard.AuthHelperConfigPath)
	}

	if cfg.Dashboard.MaxUploadSizeMb != 7 {
		t.Fatalf("expected Dashboard.MaxUploadSizeMb to be 7, got %d", cfg.Dashboard.MaxUploadSizeMb)
	}

	expectedGroups := []string{"rdpgw-admins", "homelab-admins"}
	if !reflect.DeepEqual(cfg.Dashboard.AdminGroups, expectedGroups) {
		t.Fatalf("expected Dashboard.AdminGroups to be %v, got %v", expectedGroups, cfg.Dashboard.AdminGroups)
	}
}

func TestLoadDashboardSettingsDoesNotLeakAdminGroups(t *testing.T) {
	const key = "RDPGW_DASHBOARD__ADMINGROUPS"
	previousValue, hadPreviousValue := os.LookupEnv(key)
	if err := os.Setenv(key, "admins ops"); err != nil {
		t.Fatalf("failed to set %s: %v", key, err)
	}

	cfg := Load("/definitely-missing.yaml")
	expectedGroups := []string{"admins", "ops"}
	if !reflect.DeepEqual(cfg.Dashboard.AdminGroups, expectedGroups) {
		t.Fatalf("expected Dashboard.AdminGroups to be %v, got %v", expectedGroups, cfg.Dashboard.AdminGroups)
	}

	if hadPreviousValue {
		if err := os.Setenv(key, previousValue); err != nil {
			t.Fatalf("failed to restore %s: %v", key, err)
		}
	} else {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("failed to unset %s: %v", key, err)
		}
	}

	cfg = Load("/definitely-missing.yaml")
	if len(cfg.Dashboard.AdminGroups) != 0 {
		t.Fatalf("expected Dashboard.AdminGroups to be cleared on reload, got %v", cfg.Dashboard.AdminGroups)
	}
}

func TestLoadDashboardSettingsDerivesUploadDirFromStorePath(t *testing.T) {
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__UPLOADDIR")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__ICONDIR")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__AUTHUSERSPATH")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH")
	t.Setenv("RDPGW_DASHBOARD__STOREPATH", "/tmp/rdpgw-dashboard")

	cfg := Load("/definitely-missing.yaml")
	if cfg.Dashboard.UploadDir != "/tmp/rdpgw-dashboard/uploads" {
		t.Fatalf("expected derived Dashboard.UploadDir to be /tmp/rdpgw-dashboard/uploads, got %q", cfg.Dashboard.UploadDir)
	}
	if cfg.Dashboard.IconDir != "/tmp/rdpgw-dashboard/icons" {
		t.Fatalf("expected derived Dashboard.IconDir to be /tmp/rdpgw-dashboard/icons, got %q", cfg.Dashboard.IconDir)
	}
	if cfg.Dashboard.AuthUsersPath != "/tmp/rdpgw-dashboard/auth-users.json" {
		t.Fatalf("expected derived Dashboard.AuthUsersPath to be /tmp/rdpgw-dashboard/auth-users.json, got %q", cfg.Dashboard.AuthUsersPath)
	}
	if cfg.Dashboard.AuthHelperConfigPath != "/tmp/rdpgw-dashboard/rdpgw-auth.yaml" {
		t.Fatalf("expected derived Dashboard.AuthHelperConfigPath to be /tmp/rdpgw-dashboard/rdpgw-auth.yaml, got %q", cfg.Dashboard.AuthHelperConfigPath)
	}
}

func TestCombinedDockerSampleConfig(t *testing.T) {
	unsetEnvWithCleanup(t, "RDPGW_SERVER__AUTHENTICATION")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__STOREPATH")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH")
	unsetEnvWithCleanup(t, "RDPGW_DASHBOARD__ADMINGROUPS")

	configPath := filepath.Join("..", "..", "..", "dev", "docker", "rdpgw.yaml")
	cfg := Load(configPath)

	expectedAuth := []string{"openid"}
	if !reflect.DeepEqual(cfg.Server.Authentication, expectedAuth) {
		t.Fatalf("expected combined docker auth modes %v, got %v", expectedAuth, cfg.Server.Authentication)
	}

	if cfg.Server.Port != 8443 {
		t.Fatalf("expected combined docker sample port 8443, got %d", cfg.Server.Port)
	}

	if cfg.Dashboard.StorePath != "/var/lib/rdpgw/dashboard" {
		t.Fatalf("expected Dashboard.StorePath to be /var/lib/rdpgw/dashboard, got %q", cfg.Dashboard.StorePath)
	}

	if cfg.Dashboard.AuthHelperConfigPath != "/var/lib/rdpgw/dashboard/rdpgw-auth.yaml" {
		t.Fatalf("expected Dashboard.AuthHelperConfigPath to be /var/lib/rdpgw/dashboard/rdpgw-auth.yaml, got %q", cfg.Dashboard.AuthHelperConfigPath)
	}

	expectedGroups := []string{"admin"}
	if !reflect.DeepEqual(cfg.Dashboard.AdminGroups, expectedGroups) {
		t.Fatalf("expected Dashboard.AdminGroups to be %v, got %v", expectedGroups, cfg.Dashboard.AdminGroups)
	}

	if !cfg.Caps.TokenAuth {
		t.Fatal("expected combined docker sample to keep token auth enabled for OIDC")
	}
}
