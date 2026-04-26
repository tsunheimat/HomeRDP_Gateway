# Security Policy

## Project scope

HomeRDP Gateway is a homelab-oriented Remote Desktop Gateway fork intended for personal and self-hosted environments.

Because this project handles authentication, gateway access, host authorization, and remote desktop connectivity, security reports are welcome and appreciated.

## Supported versions

Until the project publishes formal releases, security fixes are handled on the main development branch.

After releases are established, this section should be updated with a supported-version table.

## Reporting a vulnerability

Please report suspected security vulnerabilities privately to the maintainer before public disclosure.

Maintainer:

- GitHub: https://github.com/tsunheimat

Recommended report content:

- Affected version or commit hash
- Deployment mode and configuration summary
- Steps to reproduce
- Expected behavior
- Actual behavior
- Potential impact
- Any relevant logs, screenshots, or proof-of-concept details

Please avoid posting public issues for vulnerabilities that may expose authentication bypasses, authorization bypasses, token handling problems, remote code execution, credential leaks, or host access-control bypasses.

## Security-sensitive areas

Security reviews are especially welcome in the following areas:

- OIDC login and callback handling
- Gateway token generation and validation
- Cookie and session handling
- Host authorization and access-control logic
- RemoteApp file upload handling
- File parsing and storage paths
- Cross-gateway permission synchronization
- Reverse proxy and TLS deployment assumptions
- mstsc and macOS client compatibility flows

## Non-goals

This project is intended for homelab and personal infrastructure use cases.

It is not intended to provide the same support, compliance guarantees, management surface, or enterprise lifecycle as Microsoft Windows Server Remote Desktop Gateway or Microsoft Remote Desktop Services.

## Disclosure process

The maintainer will try to acknowledge valid reports and coordinate a fix when possible.

If you plan to disclose a vulnerability publicly, please allow reasonable time for investigation and remediation first.
