# Microsoft Azure Application Proxy Deployment Guide

This guide describes the current Azure Application Proxy / Entra ID pattern for HomeRDP Gateway.

The important distinction is:

- Azure Application Proxy is used to front the *web/OIDC listener*.
- The browser dashboard still belongs to rdpgw.
- Direct RDP clients should usually use the direct-auth listener separately.

## What this deployment is for

Use this pattern when you want:

- Entra ID / Azure AD authentication in front of the web UI
- trusted header propagation from Azure Application Proxy
- OIDC-backed dashboard login and `.rdp` download generation
- a private backend rdpgw service that is not exposed directly to the Internet

## What it is not for

This is *not* the recommended way to publish the native direct-auth listener for `mstsc`/FreeRDP. For direct RDP use, prefer the split gateway direct listener on `9443` and keep it on a trusted private path unless you explicitly want to publish it another way.

## Current auth flow

### Web path

- User browses to the published App Proxy URL.
- Azure App Proxy authenticates the user.
- App Proxy forwards trusted identity headers to rdpgw.
- rdpgw uses OIDC for the dashboard session and server-side RDP file generation.
- `/connect` serves the download path from the server-side allow list.

### Direct browser-host selection

Do *not* rely on browser-supplied host parameters. The current application chooses from server-side configured entries, and per-entry downloads are served from `/connect/entries/{id}.rdp`.

## Configuration

A current configuration for this pattern usually looks like this:

```yaml
Server:
  Authentication:
    - openid
    - header
  Tls: disable
  GatewayAddress: https://rdpgw.yourdomain.com
  TrustedProxyCIDRs:
    - "10.0.0.0/24" # Replace with the connector or internal source range rdpgw sees as RemoteAddr
  SecureCookies: true
  Hosts:
    - server1.internal.domain:3389
    - server2.internal.domain:3389

Header:
  UserHeader: "X-MS-CLIENT-PRINCIPAL-NAME"
  UserIdHeader: "X-MS-CLIENT-PRINCIPAL-ID"
  EmailHeader: "X-MS-CLIENT-PRINCIPAL-EMAIL"

OpenId:
  ProviderUrl: https://login.microsoftonline.com/{tenant-id}/v2.0
  ClientId: <rdpgw-oidc-client-id>
  ClientSecret: <rdpgw-oidc-client-secret>
  GroupsClaim: groups

Security:
  VerifyClientIp: true
  PAATokenSigningKey: <32-char-signing-key>
  PAATokenEncryptionKey: <32-char-encryption-key>

Caps:
  TokenAuth: true

GatewaySplit:
  Enabled: true
  OIDC:
    Hostname: https://rdpgw.yourdomain.com
    Port: 8443
  Direct:
    Hostname: https://rdpgw-direct.yourdomain.com
    Port: 9443
    Authentication:
      - ntlm
```

### Notes on the config

- `Server.Authentication` must include `openid` for the web download path.
- `header` is only trusted from configured `Server.TrustedProxyCIDRs`.
- `Server.SecureCookies: true` is recommended when App Proxy terminates HTTPS externally and rdpgw sees HTTP internally.
- `Security.VerifyClientIp` should stay enabled only if the proxy provides a stable forwarded client IP; disable it explicitly if the proxy NAT makes that unstable.
- `GatewaySplit` is what makes the sample listener split work in the current container entrypoint.

## Azure setup

### 1. Register the application

Create an Entra ID / Azure AD application for the published URL.

Use the external URL that users will visit, for example:

- `https://rdpgw.yourdomain.com`

### 2. Configure Application Proxy

Set the internal target to the private rdpgw web listener, for example:

- internal URL: `http://rdpgw-web:8443`
- external URL: `https://rdpgw.yourdomain.com`
- pre-authentication: Azure Active Directory / Entra ID

Make sure the connector or internal source IP range matches `Server.TrustedProxyCIDRs`.

### 3. Configure claims

Forward the identity headers rdpgw expects:

- `X-MS-CLIENT-PRINCIPAL-NAME`
- `X-MS-CLIENT-PRINCIPAL-ID`
- `X-MS-CLIENT-PRINCIPAL-EMAIL`

If you use different claim names in your tenant or proxy policy, update the `Header.*` fields accordingly.

### 4. Keep the backend private

Do not expose the backend rdpgw service publicly if App Proxy is the front door.

Recommended controls:

- private network only
- Kubernetes `NetworkPolicy` if you are on a cluster
- reverse proxy / connector source ranges limited to the App Proxy connector
- `Server.TrustedProxyCIDRs` narrowed to the real connector or load-balancer source range

## Testing

### Test the web flow

```bash
curl -v https://rdpgw.yourdomain.com/
curl -v https://rdpgw.yourdomain.com/admin
curl -v https://rdpgw.yourdomain.com/connect
```

Expected result:

- App Proxy redirects/challenges to Entra ID as needed
- rdpgw accepts the trusted headers only from the proxy
- the dashboard session and RDP download path work after login

### Test a per-entry download

```bash
curl -v https://rdpgw.yourdomain.com/connect/entries/<id>.rdp
```

### Do not test header auth by spoofing headers directly

Direct requests to rdpgw with fake identity headers should be rejected unless they originate from a configured trusted proxy CIDR.

## Troubleshooting

### Browser login loops

Check:

- `Server.GatewayAddress` matches the externally visible URL
- the App Proxy external URL matches the OIDC callback URL
- `Server.SecureCookies` is `true` when the frontend is HTTPS and the backend is HTTP
- the proxy forwards the expected host and scheme

### Header auth fails

Check:

- the immediate source IP is inside `Server.TrustedProxyCIDRs`
- the proxy strips client-supplied identity headers before forwarding
- `Header.UserHeader` matches the header actually injected by App Proxy

### Dashboard works but downloads fail

Check:

- `Server.Authentication` includes `openid`
- `Caps.TokenAuth` is enabled
- the configured host entries exist and are enabled
- the OIDC provider settings are correct

## Recommended current pattern

For most current deployments, this is the safest split:

- App Proxy publishes the OIDC/dashboard listener
- direct RDP clients use the direct listener on `9443`
- both listeners read the same dashboard state
- only the proxy-facing web path is exposed through App Proxy

This keeps the web SSO and direct-auth flows separate while still using the current split-gateway design.
