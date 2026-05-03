# NTLM Authentication

RDPGW supports NTLM authentication for direct RDP clients such as Windows `mstsc`. In the current HomeRDP Gateway deployment model, NTLM and `local` are *direct listener* auth modes: they are meant for the dedicated direct-auth listener, while the OIDC/dashboard listener handles browser login and downloads.

## Current deployment model

The checked-in sample uses split gateway mode:

- OIDC/dashboard listener on `8443`
- direct-auth listener on `9443`
- shared dashboard state under `/var/lib/rdpgw/dashboard`
- shared `rdpgw-auth` helper socket under `/run/rdpgw/rdpgw-auth.sock` in non-container deployments, or under the container's writable `/tmp` runtime directory

When `GatewaySplit.Enabled: true` is set, the container entrypoint starts both listeners and starts `rdpgw-auth` automatically when the direct listener uses `ntlm` or `local`.

## When to use NTLM

Use NTLM when you want:

- Windows `mstsc` compatibility
- direct RDP login without browser OIDC for the actual RDP tunnel
- dashboard-managed direct-auth users

If your clients can use browser SSO instead, use OIDC on the dashboard listener and leave the direct listener for native RDP access only.

## Configuration

### Minimal direct-auth config

```yaml
Server:
  Authentication:
    - ntlm
  AuthSocket: /run/rdpgw/rdpgw-auth.sock
  SecureCookies: true
Caps:
  TokenAuth: false
```

Notes:

- `Server.AuthSocket` must match the socket path used by `rdpgw-auth`.
- `Server.SecureCookies: true` is recommended when an HTTPS reverse proxy terminates TLS before rdpgw.
- `Caps.TokenAuth` should remain `false` for a pure direct-auth listener.

### Split gateway config

```yaml
Server:
  Authentication:
    - openid
Dashboard:
  StorePath: /var/lib/rdpgw/dashboard
  AdminGroups:
    - admin
GatewaySplit:
  Enabled: true
  OIDC:
    Hostname: https://rdpgw.example.com
    Port: 8443
  Direct:
    Hostname: https://rdpgw-direct.example.com
    Port: 9443
    Authentication:
      - ntlm
```

If you want a `local` direct listener instead, replace `ntlm` with `local` in `GatewaySplit.Direct.Authentication`.

## How direct-auth user management works now

The supported workflow is dashboard-managed:

1. Sign in to `/admin` on the OIDC/dashboard listener.
2. Create or update direct-auth users in the Direct Auth Users section.
3. The server writes `auth-users.json` under the dashboard store.
4. The server regenerates `rdpgw-auth.yaml` automatically.
5. `rdpgw-auth` reloads the generated file on the next auth request.

Do not hand-edit the generated helper YAML as the source of truth in the supported deployment.

## Authentication flow

1. Client connects to the direct listener on `9443`.
2. Gateway forwards the NTLM handshake to `rdpgw-auth` over the Unix socket.
3. `rdpgw-auth` validates the credentials against the generated direct-auth user list.
4. The client connects to the target host after successful authentication.

## Docker deployment

The root `docker-compose.yml` uses the published GHCR image, runs as UID/GID `1001`, and mounts `/tmp` as writable tmpfs. The container entrypoint starts `rdpgw-auth` automatically when the config enables `ntlm` or `local` on the direct listener.

If you use a non-container deployment, start `rdpgw-auth` separately and point it at the same socket and generated helper config path.

## Windows client notes

- Use the direct listener hostname and port `9443` in `mstsc`.
- Save gateway credentials when prompted.
- If you are testing against a TLS-terminating proxy, keep `Server.SecureCookies: true`.

## Troubleshooting

### Direct auth fails

Check:

- `GatewaySplit.Enabled: true` is present in the mounted config
- `GatewaySplit.Direct.Authentication` includes `ntlm` or `local`
- the direct listener port matches the service or port-forward target
- `rdpgw-auth` is running or the container entrypoint is starting it
- the auth socket path is writable
- a direct-auth user exists and is enabled in `/admin`
- the direct-auth host entry is enabled

### Helper config looks stale

Check:

- `Dashboard.StorePath` points at the mounted dashboard volume
- `Dashboard.AuthHelperConfigPath` is derived or set to the same shared path the helper reads
- the dashboard store volume is writable by UID `1001`

## Security notes

- Direct-auth passwords are sensitive and should be treated as credential data.
- Keep the dashboard store and backups protected.
- Do not expose the direct listener directly to untrusted networks unless that is the intended design.
- Prefer TLS termination or a trusted private network, and keep `Server.SecureCookies: true` when browser sessions cross a TLS terminator.
