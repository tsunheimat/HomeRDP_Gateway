# Contributing

Thank you for your interest in contributing to HomeRDP Gateway.

HomeRDP Gateway is an independent modified fork of rdpgw, maintained by tsunheimat and focused on homelab-oriented Remote Desktop Gateway use cases.

## Project positioning

This project is intended for:

- homelab users,
- self-hosted environments,
- personal infrastructure,
- small private deployments,
- OIDC-backed dashboard, split gateway, and direct-auth workflows.

This project is not intended to be a full replacement for Microsoft Windows Server Remote Desktop Gateway, Microsoft Remote Desktop Services, or any official Microsoft server product.

Please keep this positioning in mind when proposing changes.

## Contribution license

Unless you explicitly state otherwise, any contribution intentionally submitted to this project is submitted under the Apache License 2.0.

By submitting a contribution, you agree that it may be included in this project under the Apache License 2.0.

## Contribution guidelines

Before opening a pull request, please consider:

- Is the change useful for homelab or self-hosted deployments?
- Does the change preserve compatibility with Windows mstsc and macOS Remote Desktop clients where relevant?
- Does the change affect OIDC login, session handling, token handling, file upload, or host authorization?
- Does the change require documentation updates?
- Does the change preserve attribution and license notices from upstream rdpgw?

## Security-sensitive changes

Please be especially careful with changes involving:

- authentication,
- authorization,
- OIDC provider handling,
- gateway tokens,
- cookies and sessions,
- uploaded files,
- host access-control rules,
- cross-gateway permission behavior,
- TLS or reverse proxy assumptions.

For suspected vulnerabilities, please follow SECURITY.md instead of opening a public issue.

## Relationship to upstream

This project is derived from rdpgw:

https://github.com/bolkedebruin/rdpgw

This fork is maintained independently and is not affiliated with, endorsed by, or maintained by the original rdpgw project.
