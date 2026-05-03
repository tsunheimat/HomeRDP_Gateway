# HomeRDP Gateway

[![Go](https://github.com/tsunheimat/HomeRDP_Gateway/actions/workflows/go.yml/badge.svg)](https://github.com/tsunheimat/HomeRDP_Gateway/actions/workflows/go.yml)
[![CodeQL](https://github.com/tsunheimat/HomeRDP_Gateway/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/tsunheimat/HomeRDP_Gateway/actions/workflows/codeql-analysis.yml)
[![Container Image CI](https://github.com/tsunheimat/HomeRDP_Gateway/actions/workflows/docker-image.yml/badge.svg)](https://github.com/tsunheimat/HomeRDP_Gateway/actions/workflows/docker-image.yml)

HomeRDP Gateway is a homelab-oriented Remote Desktop Gateway based on the open-source [rdpgw](https://github.com/bolkedebruin/rdpgw) project. It lets users connect to RDP hosts through HTTPS with Microsoft Remote Desktop clients, while adding a browser dashboard and self-hosted management workflows that are useful for small private deployments.

This repository is an **independent modified fork** maintained by [tsunheimat](https://github.com/tsunheimat). It is not affiliated with, endorsed by, sponsored by, or maintained by the original rdpgw project, its maintainers, Microsoft, Windows Server, or Remote Desktop Services.

HomeRDP Gateway is designed for homelab, self-hosted, and personal infrastructure use cases. It is not intended to be a full enterprise replacement for Microsoft Windows Server Remote Desktop Gateway or Microsoft Remote Desktop Services.

## Status

- License: Apache License 2.0.
- Repository: <https://github.com/tsunheimat/HomeRDP_Gateway>.
- Upstream: <https://github.com/bolkedebruin/rdpgw>.
- Upstream sync status: this fork is independently maintained and is not currently guaranteed to match the latest upstream release.
- Container publishing target: GitHub Container Registry (`ghcr.io`), not Docker Hub.

See also:

- [docs/setup-guide.md](./docs/setup-guide.md) for first-time Docker Compose and Kubernetes setup.
- [MODIFICATIONS.md](./MODIFICATIONS.md) for major changes from upstream.
- [NOTICE](./NOTICE) for attribution and non-affiliation notices.
- [SECURITY.md](./SECURITY.md) for vulnerability reporting.
- [CONTRIBUTING.md](./CONTRIBUTING.md) for contribution guidance.

## Main features

HomeRDP Gateway keeps the core rdpgw gateway behavior and focuses on these fork-specific areas:

- HTTPS Remote Desktop Gateway for Microsoft RDP clients.
- OpenID Connect login flow for browser-based authentication.
- Homelab dashboard for presenting allowed RDP entries after OIDC login.
- Admin UI/API for managing dashboard entries, uploaded RDP templates, icons, and direct-auth users.
- Split gateway mode: one OIDC/browser listener plus one direct-auth listener for native RDP clients.
- Managed host allowlist for direct `local` and `ntlm` RDP gateway traffic.
- Support for Windows `mstsc`, macOS Microsoft Remote Desktop, iOS/Android Microsoft clients, and FreeRDP where compatible.
- Header-based authentication for deployments behind trusted identity-aware reverse proxies.
- Kerberos, PAM/local, NTLM, and OpenID Connect authentication modes inherited or extended from upstream rdpgw.
- Hardened container defaults: non-root runtime user, dropped capabilities, read-only root filesystem support, and explicit writable data paths.

## Architecture overview

A typical homelab deployment has two externally visible roles:

1. **OIDC/dashboard listener**
   - Serves `/` and `/admin`.
   - Handles browser login through OpenID Connect.
   - Lets admins manage hosts, uploaded RDP templates, icons, and direct-auth users.
   - Generates downloadable `.rdp` files for dashboard entries.

2. **Direct-auth listener**
   - Serves native RDP gateway traffic for clients that use `local` or `ntlm` authentication.
   - Reuses the dashboard-managed host inventory and direct-auth user list.
   - Intended for clients such as Windows `mstsc` or FreeRDP when browser-driven OIDC is not the desired path.

The checked-in Docker sample runs both roles in one container for local testing. Production deployments can split or front these listeners differently, but plain HTTP backends should only be reachable from trusted reverse proxies or private networks.

## Authentication modes

### OpenID Connect

Use OpenID Connect when you want browser login, MFA through an identity provider, and dashboard entry filtering by group claims.

Relevant docs:

- [docs/openid-authentication.md](./docs/openid-authentication.md)

### Dashboard authorization

With OpenID Connect enabled, `/` serves a dashboard after login. Entries are filtered by groups extracted from the OIDC token using the configured `OpenId.GroupsClaim`. Users only see entries where at least one of their groups matches the entry `AllowedGroups`.

`Dashboard.AdminGroups` controls access to `/admin` and admin API endpoints. Admin users can create host entries, upload `.rdp` templates, update/delete entries, manage direct-auth users, and upload/select/delete the web page icon.

Dashboard group checks apply to OIDC web sessions. For native RDP clients using direct `local`, `ntlm`, or `kerberos` authentication, enabled dashboard host entry addresses are used as the allowlist; entry `AllowedGroups` are not evaluated because those modes do not reliably provide group claims to the gateway.

### Local/PAM and NTLM

Use direct authentication when a native RDP client should authenticate directly to the gateway instead of going through browser OIDC.

- PAM/local authentication uses the bundled `rdpgw-auth` helper.
- NTLM authentication uses the direct-auth user configuration managed by the dashboard workflow.

Relevant docs:

- [docs/ntlm-authentication.md](./docs/ntlm-authentication.md)

### Kerberos

Kerberos mode requires client-side Kerberos setup, a KDC reachable by the gateway, and appropriate keytab/krb5 configuration.

Relevant docs:

- [docs/kerberos-authentication.md](./docs/kerberos-authentication.md)

### Header authentication

Header authentication is intended for deployments behind a trusted reverse proxy or identity-aware access proxy. Only enable it when the gateway can trust the proxy source addresses.

Relevant docs:

- [docs/header-authentication.md](./docs/header-authentication.md)
- [docs/ms-app-proxy-deployment.md](./docs/ms-app-proxy-deployment.md)

## TLS and security notes

Remote Desktop clients normally require a valid TLS certificate whose hostname matches the gateway address. You can provide a certificate/key pair, use ACME/Let's Encrypt where supported by your deployment, or terminate TLS at a trusted reverse proxy.

Security reminders:

- Do not expose a plain HTTP backend directly to untrusted networks. If TLS terminates at a reverse proxy, keep the backend reachable only from that proxy/private network and set `Server.SecureCookies: true`.
- Replace all `CHANGE_ME` sample keys and secrets before production use. Do not use placeholder values even when they happen to be the right length.
- Keep `SessionKey`, `SessionEncryptionKey`, `PAATokenSigningKey`, and `PAATokenEncryptionKey` random and deployment-specific. These values must be exactly 32 characters; generate suitable hex strings with a command such as `openssl rand -hex 16`.
- Treat dashboard entries as privileged routing policy: an entry controls which internal `host:port` a user can reach through the gateway. Limit admin access and review entries before publishing them.
- RDP template uploads are treated as admin-managed content. The gateway overwrites the final target, username, gateway token, and configured redirection settings before serving a generated `.rdp` file.
- Keep dashboard data directories writable only by the gateway runtime user.
- Direct-auth passwords are sensitive. Treat dashboard state and backups as credential-bearing data, exclude them from Docker build contexts, and encrypt backups.
- `/metrics` is disabled by default. Enable it only with `Server.EnableMetrics: true`, and do not expose it publicly unless your reverse proxy or network policy protects it.
- `SSLKEYLOGFILE` can decrypt captured TLS traffic. HomeRDP Gateway fails closed if it is set unless `Server.AllowTLSKeyLog: true` is explicitly configured. Enable it only for short-lived debugging.

## Container images

Docker Hub publishing is intentionally disabled for this fork. Container images are intended to be published to GitHub Container Registry:

```text
ghcr.io/tsunheimat/homerdp-gateway
```

After the GitHub repository is created and the workflow is enabled, expected tags are:

```text
ghcr.io/tsunheimat/homerdp-gateway:latest
ghcr.io/tsunheimat/homerdp-gateway:<git-ref-or-version>
```

For local builds without a registry:

```bash
docker build -f dev/docker/Dockerfile -t homerdp-gateway:local .
```

## Build from source

Prerequisites:

- Go 1.25.x compatible with `go.mod`.
- `make`.
- PAM development headers on Linux when building PAM-related code, for example `libpam0g-dev` on Debian/Ubuntu.

Build and test:

```bash
make build
go test ./...
```

The default `make` target builds both gateway binaries into `bin/`.

## Local Docker test

The local Docker Compose sample starts a TLS-ready split gateway setup on ports `8443` and `9443`.

For a step-by-step walkthrough, see the [setup guide](./docs/setup-guide.md).

1. Review and replace the placeholder values in [`dev/docker/rdpgw.yaml`](./dev/docker/rdpgw.yaml).
2. Start the sample from the repository root using the published GHCR image:

```bash
docker compose up
```

3. Check the listeners:

```bash
curl -k https://localhost:8443/
curl -k https://localhost:9443/
```

4. Sign into the admin UI:

```text
https://localhost:8443/admin
```

5. Create at least one enabled host entry and, if testing direct authentication, create a direct-auth user.

Example FreeRDP direct-auth test:

```bash
xfreerdp /g:localhost:9443 /gd:"" /u:<direct-auth-user> /p:<direct-auth-password> /v:<enabled-dashboard-host> /cert-ignore
```

The sample container runs as UID/GID `1001`, drops Linux capabilities, uses `no-new-privileges`, supports a read-only root filesystem, and stores dashboard state under `./data/dashboard` in the compose environment. The rdpgw-auth helper runs as the same non-root user as the gateway and the image does not set the setuid bit. Keep the dashboard directory writable by UID `1001`.

For Kubernetes deployments, use an equivalent pod/container security context where it is compatible with your auth mode:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1001
  runAsGroup: 1001
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true
```

Mount writable volumes only where needed, for example a private auth-socket runtime directory such as `/run/rdpgw/rdpgw-auth.sock` for non-Docker deployments, the Docker sample's `/tmp` tmpfs for `/tmp/rdpgw-auth/rdpgw-auth.sock`, a dashboard state volume such as `/var/lib/rdpgw/dashboard`, and `/var/lib/rdpgw/certs` when ACME certificate caching is enabled.

A Kubernetes starter manifest is available at [`k8s/rdpgw.yaml`](./k8s/rdpgw.yaml). It uses the published GHCR image, provides `rdpgw.yaml` through a `ConfigMap`, mounts that key at `/opt/rdpgw/rdpgw.yaml`, and mirrors the Docker sample's non-root/read-only runtime posture.

## Minimal configuration sketch

This is a shortened configuration sketch. See [`dev/docker/rdpgw.yaml`](./dev/docker/rdpgw.yaml) and the documents under [`docs/`](./docs/) for fuller examples.

```yaml
Server:
  Authentication:
    - openid
  GatewayAddress: https://your-oidc-gateway.example.com
  Port: 8443
  Tls: enable
  CertFile: /path/to/server.pem
  KeyFile: /path/to/key.pem
  Hosts:
    - example-host:3389
  SessionKey: REPLACE_WITH_RANDOM_SESSION_KEY
  SessionEncryptionKey: REPLACE_WITH_RANDOM_SESSION_ENCRYPTION_KEY
  SecureCookies: true

OpenId:
  ProviderUrl: https://idp.example.com/realms/example
  ClientId: homerdp-gateway
  ClientSecret: CHANGE_ME_USE_ENV_OR_SECRET_STORE
  GroupsClaim: groups

Dashboard:
  StorePath: /var/lib/rdpgw/dashboard/store.json
  UploadDir: /var/lib/rdpgw/dashboard/uploads
  IconDir: /var/lib/rdpgw/dashboard/icons
  AdminGroups:
    - admin

Security:
  PAATokenSigningKey: REPLACE_WITH_RANDOM_PAA_SIGNING_KEY
  PAATokenEncryptionKey: REPLACE_WITH_RANDOM_PAA_ENCRYPTION_KEY
  VerifyClientIp: true

Caps:
  TokenAuth: true
  EnableClipboard: false
  EnableDrive: false
  EnablePrinter: false
  EnablePort: false
  EnablePnp: false
```

## Client caveats

Microsoft Remote Desktop clients differ in gateway behavior:

- Windows `mstsc` does not support basic authentication for the gateway; use OpenID Connect, Kerberos, or NTLM.
- Windows `mstsc` often requires saved gateway credentials or a domain-style username such as `.\username`.
- Windows `mstsc` requires a valid certificate and is stricter about TLS/cipher configuration than some other clients.
- Host entries should normally include hostnames and ports, for example `myserver.example.com:3389`.
- The Microsoft Remote Desktop client for macOS is generally more flexible and can use different gateway and RDP-host credentials.
- FreeRDP can be useful for testing and automation.

When using a reverse proxy for TLS termination, verify that your cipher suites and forwarded headers are compatible with the clients you support.

## Upstream relationship

HomeRDP Gateway is derived from rdpgw:

```text
https://github.com/bolkedebruin/rdpgw
```

The upstream project is licensed under Apache License 2.0. This fork preserves the Apache License 2.0 license text and relevant attribution. See [NOTICE](./NOTICE) and [MODIFICATIONS.md](./MODIFICATIONS.md).

Please report issues for this fork at <https://github.com/tsunheimat/HomeRDP_Gateway/issues>. For upstream rdpgw issues that are not specific to this fork, consider reporting them upstream.

## Acknowledgements

- The original rdpgw project by bolkedebruin and contributors.
- Software developed by Thomson Reuters Global Resources: [go-ntlm](https://github.com/m7913d/go-ntlm) under BSD-4-Clause.
