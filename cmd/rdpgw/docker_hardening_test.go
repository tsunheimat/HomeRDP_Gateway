package main

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readRepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", ".."}, parts...)...)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}

func moduleGoVersion(t *testing.T) string {
	t.Helper()
	goMod := readRepoFile(t, "go.mod")
	for _, line := range strings.Split(goMod, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go "))
		}
	}
	t.Fatal("go.mod does not declare a go version")
	return ""
}

func TestDockerfileRuntimePrivilegeHardened(t *testing.T) {
	dockerfile := readRepoFile(t, "dev", "docker", "Dockerfile")
	goVersion := moduleGoVersion(t)

	if !regexp.MustCompile(`(?m)^FROM\s+golang:` + regexp.QuoteMeta(goVersion) + `\s+AS\s+builder\s*$`).MatchString(dockerfile) {
		t.Fatalf("Dockerfile builder image must match go.mod version %s", goVersion)
	}
	if regexp.MustCompile(`(?m)^USER\s+(0|root)\s*$`).MatchString(dockerfile) {
		t.Fatal("Dockerfile must not leave the runtime container running as root")
	}
	if !regexp.MustCompile(`(?m)^USER\s+(rdpgw|1001)(?::(rdpgw|1001))?\s*$`).MatchString(dockerfile) {
		t.Fatal("Dockerfile must set final USER to the non-root rdpgw identity")
	}
	for _, forbidden := range []string{"chmod u+s", "chmod +s", "chmod 4", "setuid"} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Dockerfile must not grant rdpgw-auth setuid privileges by default; found %q", forbidden)
		}
	}
	for _, forbidden := range []string{
		"COPY --chown=1001 dev/docker/run.sh",
		"COPY --chown=1001 dev/docker/run.lib.sh",
		"COPY --from=builder --chown=1001 /out/rdpgw",
		"COPY --from=builder --chown=1001 /out/rdpgw-auth",
		"COPY --chown=1001 cmd/rdpgw/templates",
		"COPY --chown=1001 assets",
		"chown -R rdpgw:rdpgw /opt/rdpgw",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Dockerfile must keep runtime application files root-owned/non-writable by the rdpgw user; found %q", forbidden)
		}
	}
	for _, required := range []string{
		"chown root:root /opt/rdpgw /opt/rdpgw/templates /opt/rdpgw/assets",
		"chmod 0755 /opt/rdpgw /opt/rdpgw/templates /opt/rdpgw/assets",
		"chmod 0555 /run.sh /run.lib.sh /opt/rdpgw/rdpgw /opt/rdpgw/rdpgw-auth",
		"chown root:rdpgw /opt/rdpgw/key.pem /opt/rdpgw/server.pem",
		"chmod 0640 /opt/rdpgw/key.pem",
		"chown -R rdpgw:rdpgw /var/lib/rdpgw",
	} {
		if !strings.Contains(dockerfile, required) {
			t.Fatalf("Dockerfile missing runtime ownership/permission hardening %q", required)
		}
	}
}

func TestDockerRunScriptRunsAsCurrentNonRootUser(t *testing.T) {
	runScript := readRepoFile(t, "dev", "docker", "run.sh")

	for _, forbidden := range []string{"su -c", "exec su", " sh -c \"${AUTH_CMD}\""} {
		if strings.Contains(runScript, forbidden) {
			t.Fatalf("run.sh must not require a root supervisor or shell-built auth command; found %q", forbidden)
		}
	}
	if strings.Contains(runScript, "USER=rdpgw") {
		t.Fatal("run.sh should not hard-code privilege dropping to rdpgw; the image USER already selects the runtime identity")
	}
	if !strings.Contains(runScript, "exec /opt/rdpgw/rdpgw") {
		t.Fatal("run.sh should exec rdpgw directly as the current user")
	}
	authSocketDefaultRE := regexp.MustCompile(`(?m)^AUTH_SOCKET=\$\{RDPGW_SERVER__AUTH_SOCKET:-([^}]+)\}$`)
	authSocketDefault := authSocketDefaultRE.FindStringSubmatch(runScript)
	if authSocketDefault == nil {
		t.Fatal("run.sh must default AUTH_SOCKET from RDPGW_SERVER__AUTH_SOCKET")
	}
	defaultSocket := authSocketDefault[1]
	if defaultSocket == "/tmp/rdpgw-auth.sock" || path.Dir(defaultSocket) == "/tmp" {
		t.Fatalf("run.sh default auth socket must be under a dedicated parent directory, got %q", defaultSocket)
	}
	if defaultSocket != "/tmp/rdpgw-auth/rdpgw-auth.sock" {
		t.Fatalf("run.sh default auth socket = %q, want /tmp/rdpgw-auth/rdpgw-auth.sock", defaultSocket)
	}
	if !strings.Contains(runScript, "export RDPGW_SERVER__AUTH_SOCKET=\"${AUTH_SOCKET}\"") {
		t.Fatal("run.sh must export the resolved auth socket so rdpgw and rdpgw-auth use the same dedicated socket path")
	}
	if strings.Index(runScript, "export RDPGW_SERVER__AUTH_SOCKET=\"${AUTH_SOCKET}\"") > strings.Index(runScript, "exec /opt/rdpgw/rdpgw") {
		t.Fatal("run.sh must export RDPGW_SERVER__AUTH_SOCKET before starting rdpgw")
	}
	if !strings.Contains(runScript, "--socket-mode 0660") || !strings.Contains(runScript, "--socket-dir-mode 0750") || !strings.Contains(runScript, "--socket-group rdpgw") {
		t.Fatal("run.sh must keep restrictive auth-helper socket mode, directory mode, and group settings")
	}
	if !strings.Contains(runScript, "-c \"${AUTH_CONFIG}\"") {
		t.Fatal("run.sh must pass the derived auth helper config path to rdpgw-auth")
	}
	if strings.Contains(runScript, "--socket-group rdpgw &") {
		t.Fatal("run.sh must not start rdpgw-auth without -c ${AUTH_CONFIG}, even while waiting for a generated config")
	}
}

func TestDockerComposeRuntimeHardening(t *testing.T) {
	compose := readRepoFile(t, "dev", "docker", "docker-compose.yml")

	for _, required := range []string{
		`user: "1001:1001"`,
		"no-new-privileges:true",
		"cap_drop:",
		"- ALL",
		"read_only: true",
		"tmpfs:",
		"/tmp",
	} {
		if !strings.Contains(compose, required) {
			t.Fatalf("docker-compose.yml missing runtime hardening setting %q", required)
		}
	}
}

func TestDockerAndKubernetesHardeningDocumented(t *testing.T) {
	readme := readRepoFile(t, "README.md")

	for _, required := range []string{
		"runAsNonRoot",
		"allowPrivilegeEscalation: false",
		"drop: [\"ALL\"]",
		"readOnlyRootFilesystem",
		"rdpgw-auth helper runs as the same non-root user",
		"does not set the setuid bit",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README.md missing Docker/Kubernetes hardening guidance %q", required)
		}
	}
}
