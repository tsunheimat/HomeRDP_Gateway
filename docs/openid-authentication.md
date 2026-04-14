# OpenID Connect Authentication

![OpenID Connect](images/flow-openid.svg)

RDPGW supports OpenID Connect authentication for the web UI and dashboard administration. In the homelab dashboard deployment, OIDC is used for `/` and `/admin`, while `local` and `ntlm` remain direct gateway authentication methods for native RDP clients.

## Configuration

To use OpenID Connect, ensure you have properly configured your OpenID Connect provider with a client ID and secret. The client ID and secret authenticate the gateway to the OpenID Connect provider. The provider authenticates the user and provides the gateway with a token, which generates a PAA token for RDP host connections.

```yaml
Server:
  Authentication:
    - openid
OpenId:
  ProviderUrl: https://<provider_url>
  ClientId: <your_client_id>
  ClientSecret: <your_client_secret>
  GroupsClaim: groups
Dashboard:
  StorePath: ./data/dashboard
  UploadDir: ./data/dashboard/uploads
  AuthUsersPath: ./data/dashboard/auth-users.json
  AuthHelperConfigPath: ./data/dashboard/rdpgw-auth.yaml
  AdminGroups:
    - rdpgw-admins
  MaxUploadSizeMb: 5
Caps:
  TokenAuth: true
```

### Dashboard + Group Configuration

When OpenID Connect is enabled, the homelab dashboard uses OIDC group membership for entry visibility and admin authorization:

- `OpenId.GroupsClaim`: claim name to read group memberships from the ID token. Default: `groups`.
- `Dashboard.StorePath`: directory for dashboard metadata (`entries.json`). Default: `./data/dashboard`.
- `Dashboard.UploadDir`: directory for uploaded `.rdp` templates. Default: derived from `StorePath` as `<StorePath>/uploads`.
- `Dashboard.AuthUsersPath`: JSON file storing direct-auth users managed from `/admin`. Default: derived from `StorePath` as `<StorePath>/auth-users.json`.
- `Dashboard.AuthHelperConfigPath`: generated helper YAML consumed by `rdpgw-auth`. Default: derived from `StorePath` as `<StorePath>/rdpgw-auth.yaml`.
- `Dashboard.AdminGroups`: groups allowed to access `/admin` and admin APIs.
- `Dashboard.MaxUploadSizeMb`: max upload size for template `.rdp` files. Default: `5`.

Example:

```yaml
OpenId:
  ProviderUrl: https://keycloak.example.com/realms/homelab
  ClientId: rdpgw
  ClientSecret: your-secret
  GroupsClaim: groups
Dashboard:
  StorePath: /var/lib/rdpgw/dashboard
  UploadDir: /var/lib/rdpgw/dashboard/uploads
  AuthUsersPath: /var/lib/rdpgw/dashboard/auth-users.json
  AuthHelperConfigPath: /var/lib/rdpgw/dashboard/rdpgw-auth.yaml
  AdminGroups:
    - rdpgw-admins
    - homelab-admins
  MaxUploadSizeMb: 10
```

### Environment Variable Overrides

You can override the same settings via environment variables:

- `RDPGW_OPENID__GROUPSCLAIM`
- `RDPGW_DASHBOARD__STOREPATH`
- `RDPGW_DASHBOARD__UPLOADDIR`
- `RDPGW_DASHBOARD__AUTHUSERSPATH`
- `RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH`
- `RDPGW_DASHBOARD__ADMINGROUPS`
- `RDPGW_DASHBOARD__MAXUPLOADSIZEMB`

Notes:

- `RDPGW_DASHBOARD__ADMINGROUPS` is space-separated (for example: `rdpgw-admins homelab-admins`).
- If `RDPGW_DASHBOARD__UPLOADDIR`, `RDPGW_DASHBOARD__AUTHUSERSPATH`, or `RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH` are not set, they are derived from `StorePath`.

### Direct Auth Management

When dashboard mode is enabled, `/admin` manages two kinds of state:

- published host and template entries for OIDC users
- direct-auth users for `ntlm` and `local` gateway logins

The server regenerates the helper YAML after every direct-auth user change, and `rdpgw-auth` reloads that file automatically. Allowed hosts for direct gateway auth are also read from enabled dashboard host entries instead of only `Server.Hosts`.

## Authentication Flow

1. User navigates to `https://your-gateway/connect`
2. Gateway redirects to OpenID Connect provider for authentication
3. User authenticates with the provider (supports MFA)
4. Provider redirects back to gateway with authentication token
5. Gateway validates token and generates RDP file with temporary credentials
6. User downloads RDP file and connects using remote desktop client

## Multi-Factor Authentication (MFA)

RDPGW provides multi-factor authentication out of the box with OpenID Connect integration. Configure MFA in your identity provider to enhance security.

## Provider Examples

### Keycloak
```yaml
OpenId:
  ProviderUrl: https://keycloak.example.com/auth/realms/your-realm
  ClientId: rdpgw
  ClientSecret: your-keycloak-secret
```

### Azure AD
```yaml
OpenId:
  ProviderUrl: https://login.microsoftonline.com/{tenant-id}/v2.0
  ClientId: your-azure-app-id
  ClientSecret: your-azure-secret
```

### Google
```yaml
OpenId:
  ProviderUrl: https://accounts.google.com
  ClientId: your-google-client-id.googleusercontent.com
  ClientSecret: your-google-secret
```

## Security Considerations

- Always use HTTPS for production deployments
- Store client secrets securely and rotate them regularly
- Configure appropriate scopes and claims in your provider
- Enable MFA in your identity provider for enhanced security
- Set appropriate session timeouts in both gateway and provider

## Troubleshooting

- Ensure `ProviderUrl` is accessible from the gateway
- Verify redirect URI is configured in your provider (usually `https://your-gateway/callback`)
- Check that required scopes (openid, profile, email) are configured
- Validate that the provider's certificate is trusted by the gateway
