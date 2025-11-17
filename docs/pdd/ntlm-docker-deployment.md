# NTLM Docker Deployment PDD

## Summary
This document details the technical approach for delivering a Docker-based NTLM deployment of `rdpgw`. The solution introduces a dedicated Docker Compose file plus supporting assets that start the gateway and NTLM authentication helper with the required configuration.

## High-Level Design
- **Docker Compose topology**
  - `rdpgw`: single container built from the repository image. It exposes HTTPS on port 8080 (self-signed cert) and forwards sessions to `10.0.30.14:3389`. When the `RDPGW_SERVER__AUTHENTICATION` list includes `ntlm` or `local`, the entrypoint script automatically launches the bundled `rdpgw-auth` binary inside the same container.
- **Inter-process communication**
  - The gateway and helper now share the Unix socket at `/tmp/rdpgw-auth.sock` inside the same container (path configurable via `RDPGW_SERVER__AUTH_SOCKET`).
- **Session continuity across proxies**
  - The gateway now persists NTLM handshake context in an HTTP-only cookie (`rdpgw-ntlm-session`) so that multi-step NTLM flows survive when TLS is terminated upstream and requests arrive over different backend connections/ports.
- **Configuration inputs**
  - New file `dev/docker/rdpgw-auth-ntlm.yaml` contains the static credential list with `matthew` / `matthewchiu`.
  - Environment variables on `rdpgw` configure gateway behavior (authentication list, host targets, TLS mode, session settings).

## Detailed Design
### Docker Compose File (`dev/docker/docker-compose-ntlm.yml`)
- Defines two services (`rdpgw`, `rdpgw-auth`) and the shared volume.
- `rdpgw`
  - Image: build from local Dockerfile (ensures consistency with repo code).
  - Environment variables:
    - `RDPGW_SERVER__AUTHENTICATION`: `ntlm`.
    - `RDPGW_SERVER__HOSTS`: `10.0.30.14:3389`.
  - `RDPGW_SERVER__TLS`: `auto` (self-signed cert baked into the image).
    - `RDPGW_SERVER__PORT`: `8080`.
    - `RDPGW_SERVER__GATEWAY_ADDRESS`: `localhost:8080` (documented for reverse proxy front-end).
    - `RDPGW_CAPS__TOKEN_AUTH`: `false`.
  - `RDPGW_SERVER__SESSION_STORE`: `file`.
    - Provide deterministic `RDPGW_SERVER__SESSION_KEY` and `RDPGW_SERVER__SESSION_ENCRYPTION_KEY` using sample but override-ready 32-char strings.
    - `RDPGW_SECURITY__PAATOKENENCRYPTIONKEY` and `RDPGW_SECURITY__PAATOKENSIGNINGKEY` set to fixed 32-char strings to avoid runtime randomization logs.
  - Volume mounts:
    - Mount `auth-socket:/tmp` for the Unix socket.
  - Port mapping `8080:8080` for local access.
### Authentication Config (`dev/docker/rdpgw-auth-ntlm.yaml`)
Bind-mounted into the container at `/opt/rdpgw/rdpgw-auth.yaml` (overridable via `RDPGW_AUTH_HELPER_CONFIG`).

### Authentication Config (`dev/docker/rdpgw-auth-ntlm.yaml`)
```yaml
Users:
  - Username: "matthew"
    Password: "matthewchiu"
```

### Documentation Updates
- Extend `README.md` (or add a dedicated guide) with:
  - Prerequisites.
  - Steps to launch the NTLM compose file.
  - Verification instructions (curl health check, sample `xfreerdp` command for manual login).
  - Security caveats (plain-text password, HTTP).

## Testing Strategy
- `docker compose -f dev/docker/docker-compose-ntlm.yml up --build` to ensure successful startup.
- Confirm socket creation inside containers (document commands).
- Run `curl -k https://localhost:8080/` to validate HTTPS availability (self-signed cert).
- Document manual RDP client test (e.g., `xfreerdp /g:localhost:8080 /gd:"" /u:matthew /p:matthewchiu /v:10.0.30.14`).

## Risks and Mitigations
- **Plain-text credentials**: Mitigated by security notice and restricting file permissions instructions.
- **Reverse proxy TLS termination**: Document how to disable TLS safely when a reverse proxy terminates HTTPS and ensure the HTTP port stays private.
- **Hard-coded keys**: Highlight in docs that values are defaults and should be rotated for production.

## Rollout Plan
- Merge configuration and docs.
- User runs compose locally.
- Monitor logs for authentication failures; adjust host reachability as needed.
