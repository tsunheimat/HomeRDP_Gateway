package main

import (
	"bytes"
	"crypto/tls"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestConfigureTLSKeyLogRejectsSSLKEYLOGFILEWithoutOptIn(t *testing.T) {
	cfg := &tls.Config{}
	keyLogPath := filepath.Join(t.TempDir(), "tls.keys")

	err := configureTLSKeyLog(cfg, config.Configuration{}, keyLogPath)

	if err == nil {
		t.Fatal("expected SSLKEYLOGFILE without Server.AllowTLSKeyLog to fail closed")
	}
	if !strings.Contains(err.Error(), "Server.AllowTLSKeyLog") {
		t.Fatalf("expected error to name Server.AllowTLSKeyLog, got %v", err)
	}
	if cfg.KeyLogWriter != nil {
		t.Fatal("expected key log writer to remain unset")
	}
	if _, statErr := os.Stat(keyLogPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected key log file not to be created, stat err=%v", statErr)
	}
}

func TestConfigureTLSKeyLogHonorsSSLKEYLOGFILEWithOptIn(t *testing.T) {
	cfg := &tls.Config{}
	keyLogPath := filepath.Join(t.TempDir(), "tls.keys")

	err := configureTLSKeyLog(cfg, config.Configuration{
		Server: config.ServerConfig{
			AllowTLSKeyLog: true,
		},
	}, keyLogPath)

	if err != nil {
		t.Fatalf("expected SSLKEYLOGFILE with Server.AllowTLSKeyLog to be honored, got %v", err)
	}
	if cfg.KeyLogWriter == nil {
		t.Fatal("expected key log writer to be configured")
	}
	if closer, ok := cfg.KeyLogWriter.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			t.Fatalf("close key log writer: %v", err)
		}
	}
	if _, statErr := os.Stat(keyLogPath); statErr != nil {
		t.Fatalf("expected key log file to be created, stat err=%v", statErr)
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

func TestBuildGatewayWithTokenAuthUsesPAAValidation(t *testing.T) {
	cfg := config.Configuration{
		Caps: config.RDGCapsConfig{
			TokenAuth: true,
		},
	}

	gw := buildGateway(cfg, true)

	if !gw.TokenAuth {
		t.Fatal("expected token-auth gateway to require token auth")
	}
	if gw.CheckPAACookie == nil {
		t.Fatal("expected token-auth gateway to validate PAA cookies")
	}
	if gw.CheckHost == nil {
		t.Fatal("expected token-auth gateway to validate hosts from session token")
	}
}

func TestBuildGatewayForDirectAuthSkipsPAAValidation(t *testing.T) {
	cfg := config.Configuration{
		Caps: config.RDGCapsConfig{
			TokenAuth: true,
		},
	}

	gw := buildGateway(cfg, false)

	if gw.TokenAuth {
		t.Fatal("expected direct-auth gateway to skip token auth")
	}
	if gw.CheckPAACookie != nil {
		t.Fatal("expected direct-auth gateway not to require PAA cookies")
	}
	if gw.CheckHost == nil {
		t.Fatal("expected direct-auth gateway to validate hosts directly")
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
	dashboardStore, authUserStore, iconStore, err := initDashboardState(cfg, helperConfigPath)
	if err != nil {
		t.Fatalf("init dashboard state: %v", err)
	}
	if iconStore == nil {
		t.Fatalf("expected icon store to be initialized")
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

func TestDockerRunScriptStartsAuthHelperWhenConfigEnablesNTLM(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	if err := os.WriteFile(configPath, []byte("Server:\n  Authentication:\n    - ntlm\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scriptPath := filepath.Join("..", "..", "dev", "docker", "run.lib.sh")
	cmd := exec.Command(
		"sh",
		"-c",
		`. "$1"; rdpgw_should_start_auth_helper --conf "$2"`,
		"sh",
		scriptPath,
		configPath,
	)
	cmd.Env = append(os.Environ(), "RDPGW_SERVER__AUTHENTICATION=")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("run shell helper: %v stderr=%q", err, stderr.String())
	}

	if got := stdout.String(); got != "true\n" {
		t.Fatalf("expected helper startup decision true, got %q", got)
	}
}

func TestDockerRunScriptSkipsAuthHelperForOpenIDOnlyConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	if err := os.WriteFile(configPath, []byte("Server:\n  Authentication:\n    - openid\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scriptPath := filepath.Join("..", "..", "dev", "docker", "run.lib.sh")
	cmd := exec.Command(
		"sh",
		"-c",
		`. "$1"; rdpgw_should_start_auth_helper --conf "$2"`,
		"sh",
		scriptPath,
		configPath,
	)
	cmd.Env = append(os.Environ(), "RDPGW_SERVER__AUTHENTICATION=")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("run shell helper: %v stderr=%q", err, stderr.String())
	}

	if got := stdout.String(); got != "false\n" {
		t.Fatalf("expected helper startup decision false, got %q", got)
	}
}

func TestDockerRunScriptReadsShortConfigFlag(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	if err := os.WriteFile(configPath, []byte("Server:\n  Authentication:\n    - local\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scriptPath := filepath.Join("..", "..", "dev", "docker", "run.lib.sh")
	cmd := exec.Command(
		"sh",
		"-c",
		`. "$1"; printf '%s\n' "$(rdpgw_config_path "$@")" "$(rdpgw_should_start_auth_helper "$@")"`,
		"sh",
		scriptPath,
		"-c",
		configPath,
	)
	cmd.Env = append(os.Environ(), "RDPGW_SERVER__AUTHENTICATION=")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("run shell helper: %v stderr=%q", err, stderr.String())
	}

	want := configPath + "\ntrue\n"
	if got := stdout.String(); got != want {
		t.Fatalf("expected config path and startup decision %q, got %q", want, got)
	}
}

func TestDockerRunScriptUsesConfiguredAuthHelperPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	configBody := "Dashboard:\n  AuthHelperConfigPath: /var/lib/rdpgw/custom-auth.yaml\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scriptPath := filepath.Join("..", "..", "dev", "docker", "run.lib.sh")
	cmd := exec.Command(
		"sh",
		"-c",
		`. "$1"; rdpgw_auth_helper_config_path --conf "$2"`,
		"sh",
		scriptPath,
		configPath,
	)
	cmd.Env = append(os.Environ(),
		"RDPGW_AUTH_HELPER_CONFIG=",
		"RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH=",
		"RDPGW_DASHBOARD__STOREPATH=",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("run shell helper: %v stderr=%q", err, stderr.String())
	}

	if got := stdout.String(); got != "/var/lib/rdpgw/custom-auth.yaml\n" {
		t.Fatalf("expected configured auth helper path, got %q", got)
	}
}

func TestDockerRunScriptDerivesAuthHelperPathFromStorePath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	configBody := "Dashboard:\n  StorePath: /var/lib/rdpgw/dashboard\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scriptPath := filepath.Join("..", "..", "dev", "docker", "run.lib.sh")
	cmd := exec.Command(
		"sh",
		"-c",
		`. "$1"; rdpgw_auth_helper_config_path --conf "$2"`,
		"sh",
		scriptPath,
		configPath,
	)
	cmd.Env = append(os.Environ(),
		"RDPGW_AUTH_HELPER_CONFIG=",
		"RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH=",
		"RDPGW_DASHBOARD__STOREPATH=",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("run shell helper: %v stderr=%q", err, stderr.String())
	}

	if got := stdout.String(); got != "/var/lib/rdpgw/dashboard/rdpgw-auth.yaml\n" {
		t.Fatalf("expected derived auth helper path, got %q", got)
	}
}

func TestDockerRunScriptStartsAuthHelperForSplitDirectAuth(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	configBody := strings.Join([]string{
		"Server:",
		"  Authentication:",
		"    - openid",
		"GatewaySplit:",
		"  Enabled: true",
		"  OIDC:",
		"    Hostname: rdapp.example.com",
		"    Port: 8443",
		"  Direct:",
		"    Hostname: rdp-direct.example.com",
		"    Port: 9443",
		"    Authentication:",
		"      - ntlm",
	}, "\n") + "\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scriptPath := filepath.Join("..", "..", "dev", "docker", "run.lib.sh")
	cmd := exec.Command(
		"sh",
		"-c",
		`. "$1"; rdpgw_should_start_auth_helper --conf "$2"`,
		"sh",
		scriptPath,
		configPath,
	)
	cmd.Env = append(os.Environ(), "RDPGW_SERVER__AUTHENTICATION=")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("run shell helper: %v stderr=%q", err, stderr.String())
	}

	if got := stdout.String(); got != "true\n" {
		t.Fatalf("expected helper startup decision true for split direct auth, got %q", got)
	}
}

func TestDockerRunScriptRuntimeEnvForSplitOIDCListener(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	configBody := strings.Join([]string{
		"GatewaySplit:",
		"  Enabled: true",
		"  OIDC:",
		"    Hostname: rdapp.example.com",
		"    Port: 8443",
		"  Direct:",
		"    Hostname: rdp-direct.example.com",
		"    Port: 9443",
		"    Authentication:",
		"      - ntlm",
		"      - local",
	}, "\n") + "\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scriptPath := filepath.Join("..", "..", "dev", "docker", "run.lib.sh")
	cmd := exec.Command(
		"sh",
		"-c",
		`. "$1"; rdpgw_runtime_exports oidc --conf "$2"`,
		"sh",
		scriptPath,
		configPath,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("run shell helper: %v stderr=%q", err, stderr.String())
	}

	want := strings.Join([]string{
		"export RDPGW_SERVER__PORT='8443'",
		"export RDPGW_SERVER__GATEWAYADDRESS='rdapp.example.com'",
		"export RDPGW_SERVER__AUTHENTICATION='openid'",
		"export RDPGW_CAPS__TOKENAUTH='true'",
		"",
	}, "\n")
	if got := stdout.String(); got != want {
		t.Fatalf("expected oidc runtime exports %q, got %q", want, got)
	}
}

func TestDockerRunScriptRuntimeEnvForSplitDirectListener(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "rdpgw.yaml")
	configBody := strings.Join([]string{
		"GatewaySplit:",
		"  Enabled: true",
		"  OIDC:",
		"    Hostname: rdapp.example.com",
		"    Port: 8443",
		"  Direct:",
		"    Hostname: rdp-direct.example.com",
		"    Port: 9443",
		"    Authentication:",
		"      - ntlm",
		"      - local",
	}, "\n") + "\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scriptPath := filepath.Join("..", "..", "dev", "docker", "run.lib.sh")
	cmd := exec.Command(
		"sh",
		"-c",
		`. "$1"; rdpgw_runtime_exports direct --conf "$2"`,
		"sh",
		scriptPath,
		configPath,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("run shell helper: %v stderr=%q", err, stderr.String())
	}

	want := strings.Join([]string{
		"export RDPGW_SERVER__PORT='9443'",
		"export RDPGW_SERVER__GATEWAYADDRESS='rdp-direct.example.com'",
		"export RDPGW_SERVER__AUTHENTICATION='ntlm local'",
		"export RDPGW_CAPS__TOKENAUTH='false'",
		"",
	}, "\n")
	if got := stdout.String(); got != want {
		t.Fatalf("expected direct runtime exports %q, got %q", want, got)
	}
}
