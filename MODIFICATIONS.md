# Modifications from upstream rdpgw

This repository is an independent modified fork of rdpgw:

https://github.com/bolkedebruin/rdpgw

HomeRDP Gateway is maintained by tsunheimat and is focused on homelab and self-hosted use cases.

## Major modifications

Compared with the original rdpgw project, this fork includes the following major changes:

1. OIDC-backed dashboard and admin UI

   This fork adds a browser dashboard and `/admin` UI backed by OpenID Connect.

   The goal is to make homelab identity-provider setups easier to use while keeping host publishing and direct-auth user management inside a small self-hosted control surface.

2. Dashboard entry, icon, and RDP template upload support

   This fork adds support for managing host entries, uploaded `.rdp` templates, uploaded web icons, and generated client downloads.

   This is intended to make RDP entry publishing and client-file generation easier for homelab users.

3. mstsc and macOS Remote Desktop client workflows

   This fork is designed to support connection workflows using Windows mstsc and macOS Remote Desktop clients.

4. Unified host access control across OIDC and direct-auth gateway listeners

   This fork introduces split gateway mode so one container can run an OIDC/dashboard listener and a direct-auth listener from the same mounted configuration.

   The intended result is that dashboard-managed enabled host entries become the shared allowlist for downloaded OIDC `.rdp` files and direct `local`/`ntlm` gateway traffic.

5. Hardened container and Kubernetes starter deployment

   This fork ships GHCR-based Docker Compose and Kubernetes starter manifests with non-root UID/GID `1001`, dropped Linux capabilities, no privilege escalation, read-only root filesystem support, explicit writable volumes, and split-listener examples.

6. Homelab-oriented positioning

   This fork is designed for personal infrastructure, self-hosted environments, and homelab users.

   It is not intended to be a full replacement for Microsoft Windows Server Remote Desktop Gateway, Microsoft Remote Desktop Services, or any official Microsoft server product.

## Maintenance status

This project is maintained independently by tsunheimat.

It is not affiliated with, endorsed by, sponsored by, or maintained by the original rdpgw project or its maintainers.

## License

This fork is licensed under the Apache License 2.0.

Original copyright, license, and attribution notices from upstream rdpgw are retained where applicable.
