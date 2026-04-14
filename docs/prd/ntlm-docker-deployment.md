# NTLM Docker Deployment PRD

## Overview
Deploy the `rdpgw` gateway using Docker with managed direct authentication so that OpenID Connect protects the web UI while `ntlm` and `local` remain native RDP client authentication methods. The gateway terminates TLS by default (self-signed cert) to satisfy Windows clients but can be switched to HTTP-only when an external reverse proxy handles TLS.

## Problem Statement
The current Docker samples need to reflect the managed direct-auth model instead of a static helper YAML. The user needs a reproducible container setup where `/admin` manages direct-auth users and allowed hosts, the gateway writes the helper YAML automatically, and native RDP clients authenticate against that generated state.

## Goals
- Provide a Docker-based deployment that runs `rdpgw` with OpenID Connect plus NTLM or local direct auth enabled.
- Store hosts and direct-auth users in the managed dashboard state.
- Keep the helper YAML generated from managed users instead of hand-edited.
- Keep TLS enabled by default for compatibility while documenting how to disable it safely when another reverse proxy handles HTTPS, ensuring NTLM auth still works when HTTP traffic is forwarded through a proxy.
- Document how to start and verify the deployment.

## Non-Goals
- Managing TLS certificates or reverse proxy configuration.
- Automating RDP host provisioning or health checks.
- Adding a separate database or generic config editor.
- Adding support for other authentication methods.

## Assumptions
- The deployment runs on a host with Docker and Docker Compose v1.28+ installed.
- Network access from the gateway container to the managed target hosts is available.
- Storing credentials in plain text is acceptable for this environment.
- Linux paths and permissions can be managed on the host as needed.

## Functional Requirements
1. Provide a Docker Compose file that starts a single `rdpgw` service which also launches the bundled `rdpgw-auth` helper when `local` or `ntlm` authentication is enabled.
2. Keep the NTLM helper accessible over the Unix socket defined by `RDPGW_SERVER__AUTH_SOCKET` (default `/tmp/rdpgw-auth.sock`).
3. Mount a writable dashboard state directory and keep `RDPGW_AUTH_HELPER_CONFIG` aligned with the generated helper output path.
4. Configure the gateway with OpenID Connect plus `ntlm` or `local`, and require `/admin` to create the allowed hosts and direct-auth users before native RDP clients connect.
5. Expose the gateway over HTTPS on a configurable port (default 8080) and document how to switch to HTTP-only when placing a reverse proxy in front without breaking NTLM authentication flows.
6. Document start-up, verification steps, and any follow-up actions required for productionization (e.g., TLS via reverse proxy).

## Acceptance Criteria
- Running `docker compose -f <new-file> up` starts the gateway without runtime configuration errors once valid OpenID settings are supplied.
- The gateway listens on the documented HTTPS port and returns a health response (`/` endpoint) with a self-signed certificate (or HTTP if explicitly disabled for a reverse proxy).
- After an admin creates an enabled host entry and a direct-auth user from `/admin`, a manual direct-auth attempt succeeds and connects toward that managed host.
- Documentation clearly states security considerations for plain-text credentials, writable managed state, and when disabling TLS for reverse proxy deployments.

## Open Questions
- Should additional logging or volumes be exposed for auditing needs? (Assume no changes unless requested.)
- Do we need to support scaling beyond a single gateway instance? (Assume single instance for now.)
