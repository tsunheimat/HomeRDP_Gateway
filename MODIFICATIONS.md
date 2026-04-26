# Modifications from upstream rdpgw

This repository is an independent modified fork of rdpgw:

https://github.com/bolkedebruin/rdpgw

The fork is maintained by tsunheimat and is focused on homelab and self-hosted use cases.

## Major modifications

Compared with the original rdpgw project, this fork includes the following major changes:

1. WebUI-assisted OIDC selection

   This fork adds a WebUI flow that allows users to select an OIDC login option.

   The goal is to make homelab identity-provider setups easier to use, especially when multiple OIDC choices or login paths are available.

2. RemoteApp tool file upload support

   This fork adds support for uploading files used by the RemoteApp tool workflow.

   This is intended to make RemoteApp-related setup and operation easier for homelab users.

3. mstsc and macOS Remote Desktop client workflows

   This fork is designed to support connection workflows using Windows mstsc and macOS Remote Desktop clients.

4. Unified host access control across two gateway components

   This fork introduces or adjusts permission-control behavior so that two gateway components can share a unified host access-control model.

   The intended result is that the gateways work together as a single control surface for host authorization.

5. Homelab-oriented positioning

   This fork is designed for personal infrastructure, self-hosted environments, and homelab users.

   It is not intended to be a full replacement for Microsoft Windows Server Remote Desktop Gateway, Microsoft Remote Desktop Services, or any official Microsoft server product.

## Maintenance status

This fork is maintained independently by tsunheimat.

It is not affiliated with, endorsed by, sponsored by, or maintained by the original rdpgw project or its maintainers.

## License

This fork is licensed under the Apache License 2.0.

Original copyright, license, and attribution notices from upstream rdpgw are retained where applicable.
