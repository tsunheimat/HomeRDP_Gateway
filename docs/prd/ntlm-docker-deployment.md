# NTLM Docker Deployment PRD

## Overview
Deploy the `rdpgw` gateway using Docker with NTLM authentication so that users can access an existing RDP host through the gateway. The gateway terminates TLS by default (self-signed cert) to satisfy Windows clients but can be switched to HTTP-only when an external reverse proxy handles TLS.

## Problem Statement
The current Docker samples focus on OpenID Connect or local authentication and do not provide a ready-to-run NTLM configuration. The user needs a reproducible container setup that enables NTLM authentication against a small static user list and forwards sessions to an existing RDP server at `10.0.30.14:3389`.

## Goals
- Provide a Docker-based deployment that runs `rdpgw` with NTLM authentication enabled.
- Configure the gateway to forward traffic to `10.0.30.14:3389`.
- Supply NTLM credentials (`matthew` / `matthewchiu`) via the authentication helper configuration.
- Keep TLS enabled by default for compatibility while documenting how to disable it safely when another reverse proxy handles HTTPS, ensuring NTLM auth still works when HTTP traffic is forwarded through a proxy.
- Document how to start and verify the deployment.

## Non-Goals
- Managing TLS certificates or reverse proxy configuration.
- Automating RDP host provisioning or health checks.
- Implementing persistent credential storage beyond the provided YAML file.
- Adding support for other authentication methods.

## Assumptions
- The deployment runs on a host with Docker and Docker Compose v1.28+ installed.
- Network access from the gateway container to `10.0.30.14:3389` is available.
- Storing credentials in plain text is acceptable for this environment.
- Linux paths and permissions can be managed on the host as needed.

## Functional Requirements
1. Provide a Docker Compose file that starts both the `rdpgw` service and the `rdpgw-auth` helper.
2. Share a Unix socket between both containers so the gateway can send NTLM challenges.
3. Mount a configuration file containing the specified username and password into the auth helper.
4. Configure the gateway with NTLM as the only authentication mechanism, targeting the RDP host `10.0.30.14:3389`.
5. Expose the gateway over HTTPS on a configurable port (default 8080) and document how to switch to HTTP-only when placing a reverse proxy in front without breaking NTLM authentication flows.
6. Document start-up, verification steps, and any follow-up actions required for productionization (e.g., TLS via reverse proxy).

## Acceptance Criteria
- Running `docker compose -f <new-file> up` starts the gateway without runtime configuration errors.
- The gateway listens on the documented HTTPS port and returns a health response (`/` endpoint) with a self-signed certificate (or HTTP if explicitly disabled for a reverse proxy).
- A manual NTLM authentication attempt using the provided credentials succeeds and connects toward the remote host (smoke-tested instructions).
- Documentation clearly states security considerations for plain-text credentials and when disabling TLS for reverse proxy deployments.

## Open Questions
- Should additional logging or volumes be exposed for auditing needs? (Assume no changes unless requested.)
- Do we need to support scaling beyond a single gateway instance? (Assume single instance for now.)
