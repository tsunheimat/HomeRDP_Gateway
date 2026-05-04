# HomeRDP Gateway

HomeRDP Gateway is a simple Remote Desktop Gateway for homelab and self-hosted environments.

It gives you:

- a browser dashboard for your RDP hosts
- OpenID Connect login for the web UI
- an admin page for host entries, icons, RDP templates, and direct-auth users
- a direct RDP gateway listener for clients such as Windows `mstsc`
- Docker Compose and Kubernetes starter manifests

This project is for personal infrastructure and small self-hosted networks.

## How it works

A typical setup has two listeners:

| Listener | Default port | Purpose |
| --- | --- | --- |
| Web / OIDC dashboard | `8443` | Browser login, dashboard, admin UI, `.rdp` downloads |
| Direct RDP gateway | `9443` | Native RDP clients using direct auth such as NTLM |

The dashboard controls the host list. Direct-auth users and enabled host entries are managed from `/admin`.

## Quick start with Docker Compose

1. Edit the local sample config:

```bash
$EDITOR dev/docker/rdpgw.yaml
```

At minimum, replace:

- OIDC provider URL, client ID, and client secret
- all `CHANGE_ME...` keys
- `GatewayAddress` / `GatewaySplit` hostnames
- admin groups
- target RDP hosts

2. Create the dashboard data directory:

```bash
mkdir -p data/dashboard
sudo chown -R 1001:1001 data/dashboard
```

3. Start the gateway:

```bash
docker compose up
```

4. Open the admin UI:

```text
http://localhost:8443/admin
```

The local sample disables backend TLS for easy testing. For real access, put it behind HTTPS and set `Server.SecureCookies: true`.

## Container image

```text
ghcr.io/tsunheimat/homerdp-gateway:latest
```

## Configuration files

Important starter files:

- `dev/docker/rdpgw.yaml` — local Docker sample config
- `docker-compose.yml` — local compose deployment
- `k8s/rdpgw.yaml` — Kubernetes starter manifest

Important config areas:

- `Server` — listener, auth, session, TLS/cookie settings
- `OpenId` — OIDC provider and client settings
- `Dashboard` — dashboard storage and admin groups
- `Security` — token signing/encryption keys
- `Caps` — RDP redirection capabilities such as clipboard and drive
- `GatewaySplit` — container entrypoint helper for starting web + direct listeners

For the full walkthrough, see [`docs/setup-guide.md`](./docs/setup-guide.md).

## Native RDP clients

After creating an enabled host entry and a direct-auth user in `/admin`, test with FreeRDP:

```bash
xfreerdp /g:<gateway-host>:9443 /gd:"" /u:<direct-auth-user> /p:<direct-auth-password> /v:<enabled-dashboard-host> /cert-ignore
```

Windows `mstsc` generally works best with NTLM direct auth and saved gateway credentials.

## Runtime hardening

The included Docker Compose and Kubernetes samples run the container with homelab-safe defaults:

- non-root UID/GID `1001:1001`
- `readOnlyRootFilesystem` / read-only root filesystem
- `allowPrivilegeEscalation: false`
- Linux capabilities dropped with `drop: ["ALL"]`
- Kubernetes `runAsNonRoot: true`
- the `rdpgw-auth helper runs as the same non-root user` and the image `does not set the setuid bit`

Only the dashboard data directory needs to be writable.

## Security notes

Before exposing the gateway beyond local testing:

- replace every placeholder secret
- keep all 32-character keys stable and private
- use HTTPS through a reverse proxy, ingress, or load balancer
- set `Server.SecureCookies: true` when users access the gateway over HTTPS
- protect the dashboard data directory and backups
- treat direct-auth passwords as sensitive credential data

## Build from source

```bash
make build
go test ./...
```

The default `make` target builds both gateway binaries into `bin/`.

## More docs

- [`docs/setup-guide.md`](./docs/setup-guide.md)
- [`docs/openid-authentication.md`](./docs/openid-authentication.md)
- [`docs/ntlm-authentication.md`](./docs/ntlm-authentication.md)
- [`SECURITY.md`](./SECURITY.md)
- [`CONTRIBUTING.md`](./CONTRIBUTING.md)

## Translations

- [繁體中文 README](./README.zh-TW.md)
- [繁體中文文件索引](./docs/zh-TW/README.md)

## License and attribution

HomeRDP Gateway is licensed under Apache License 2.0. See [`NOTICE`](./NOTICE) and [`MODIFICATIONS.md`](./MODIFICATIONS.md) for attribution and project history.
