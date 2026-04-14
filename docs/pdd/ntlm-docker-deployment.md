# NTLM Docker Deployment PDD

## Summary
This document details the technical approach for delivering a Docker-based managed direct-auth deployment of `rdpgw`. The supported topology keeps OpenID Connect for the web UI and admin workflow, while `ntlm` and `local` remain direct RDP client authentication methods backed by dashboard-managed state.

## High-Level Design
- **Docker Compose topology**
  - `rdpgw`: single container built from the repository image. It exposes HTTPS on port 8080 (self-signed cert). When the `RDPGW_SERVER__AUTHENTICATION` list includes `ntlm` or `local`, the entrypoint script automatically launches the bundled `rdpgw-auth` binary inside the same container.
- **Inter-process communication**
  - The gateway and helper now share the Unix socket at `/tmp/rdpgw-auth.sock` inside the same container (path configurable via `RDPGW_SERVER__AUTH_SOCKET`).
- **Session continuity across proxies**
  - The gateway now persists NTLM handshake context in an HTTP-only cookie (`rdpgw-ntlm-session`) so that multi-step NTLM flows survive when TLS is terminated upstream and requests arrive over different backend connections/ports.
- **Configuration inputs**
  - A writable dashboard store holds `entries.json`, `auth-users.json`, uploads, and the generated helper YAML.
  - Environment variables on `rdpgw` configure gateway behavior, OpenID Connect, dashboard storage paths, helper output path, TLS mode, and session settings.

## Detailed Design
### Docker Compose File (`dev/docker/docker-compose-ntlm.yml`)
- Defines the `rdpgw` service plus persistent volumes for dashboard state and the helper socket.
- `rdpgw`
  - Image: build from local Dockerfile (ensures consistency with repo code).
  - Environment variables:
    - `RDPGW_SERVER__AUTHENTICATION`: `openid ntlm`.
    - `RDPGW_OPENID__PROVIDERURL`, `RDPGW_OPENID__CLIENTID`, and `RDPGW_OPENID__CLIENTSECRET` for the admin login.
    - `RDPGW_DASHBOARD__ADMINGROUPS` for `/admin` authorization.
  - `RDPGW_SERVER__TLS`: `auto` (self-signed cert baked into the image).
    - `RDPGW_SERVER__PORT`: `8080`.
    - `RDPGW_SERVER__GATEWAY_ADDRESS`: `localhost:8080` (documented for reverse proxy front-end).
    - `RDPGW_CAPS__TOKEN_AUTH`: `false`.
    - `RDPGW_DASHBOARD__STOREPATH`: writable path for managed state.
    - `RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH`: generated helper YAML output path.
    - `RDPGW_AUTH_HELPER_CONFIG`: helper read path, matching the generated output path.
  - `RDPGW_SERVER__SESSION_STORE`: `file`.
    - Provide deterministic `RDPGW_SERVER__SESSION_KEY` and `RDPGW_SERVER__SESSION_ENCRYPTION_KEY` using sample but override-ready 32-char strings.
    - `RDPGW_SECURITY__PAATOKENENCRYPTIONKEY` and `RDPGW_SECURITY__PAATOKENSIGNINGKEY` set to fixed 32-char strings to avoid runtime randomization logs.
  - Volume mounts:
    - Mount a writable dashboard state directory.
    - Mount `auth-socket:/tmp` for the Unix socket.
  - Port mapping `8080:8080` for local access.

### Documentation Updates
- Extend `README.md` (or add a dedicated guide) with:
  - Prerequisites.
  - Steps to launch the managed direct-auth compose file.
  - Steps to sign into `/admin`, create hosts, and create direct-auth users.
  - Verification instructions (curl health check, sample `xfreerdp` command for manual login).
  - Security caveats (plain-text password, writable persistent storage).

## Testing Strategy
- `docker compose -f dev/docker/docker-compose-ntlm.yml up --build` to ensure successful startup.
- Confirm socket creation and generated helper YAML inside the container (document commands).
- Run `curl -k https://localhost:8080/` to validate HTTPS availability (self-signed cert).
- Sign into `/admin`, create an enabled host entry, create a direct-auth user, and then test with an RDP client.

## Risks and Mitigations
- **Plain-text credentials**: Mitigated by security notice and restricting file permissions instructions.
- **Missing managed state**: Mitigated by fail-closed startup validation for unmanaged direct-auth mode and fail-closed host authorization when the managed inventory is absent or empty.
- **Reverse proxy TLS termination**: Document how to disable TLS safely when a reverse proxy terminates HTTPS and ensure the HTTP port stays private.
- **Hard-coded keys**: Highlight in docs that values are defaults and should be rotated for production.

## Rollout Plan
- Merge configuration and docs.
- User runs compose locally.
- Monitor logs for authentication failures; adjust host reachability as needed.
