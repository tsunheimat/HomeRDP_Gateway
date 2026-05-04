# OpenID Connect Authentication

RDPGW 支援 OpenID Connect authentication，可用於 Web UI 與 dashboard administration。在 HomeRDP Gateway 的 homelab dashboard 部署模型中，OIDC 用於 `/` 與 `/admin`；`local` 與 `ntlm` 則保留給原生 RDP clients 使用的 direct gateway authentication。

## 設定

要使用 OpenID Connect，請先在你的 OpenID Connect provider 建立 client ID 與 client secret。Client ID/secret 讓 gateway 向 provider 驗證自己；provider 驗證使用者後，會把 token 回傳給 gateway，gateway 再產生給 RDP host connection 使用的 PAA token。

```yaml
Server:
  Authentication:
    - openid
  SecureCookies: true # HTTPS 由 reverse proxy 終止時建議開啟
OpenId:
  ProviderUrl: https://<provider_url>
  ClientId: <your_client_id>
  ClientSecret: <your_client_secret>
  GroupsClaim: groups
Dashboard:
  StorePath: ./data/dashboard
  UploadDir: ./data/dashboard/uploads
  IconDir: ./data/dashboard/icons
  AuthUsersPath: ./data/dashboard/auth-users.json
  AuthHelperConfigPath: ./data/dashboard/rdpgw-auth.yaml
  AdminGroups:
    - rdpgw-admins
  MaxUploadSizeMb: 5
  MaxTemplateUploads: 100
  MaxIconUploads: 100
  MaxTemplateUploadStorageMb: 100
  MaxIconUploadStorageMb: 100
Caps:
  TokenAuth: true
```

`Server.SecureCookies` 預設為 `false`，方便本機 HTTP 測試。外部 HTTPS deployment 如果透過 Traefik、Gateway API 或其他 TLS-terminating proxy，請設成 `true`，讓 browser session cookies 一律標記為 `Secure`，即使 rdpgw 從 proxy 收到的是 HTTP。

## Dashboard + Group 設定

啟用 OpenID Connect 後，homelab dashboard 會使用 OIDC group membership 控制 entry visibility 與 admin authorization：

- `OpenId.GroupsClaim`：從 ID token 讀取 group memberships 的 claim name。預設：`groups`。
- `Dashboard.StorePath`：dashboard metadata (`entries.json`) 的目錄。預設：`./data/dashboard`。
- `Dashboard.UploadDir`：上傳 `.rdp` templates 的目錄。預設由 `StorePath` 推導為 `<StorePath>/uploads`。
- `Dashboard.IconDir`：從 `/admin` 管理的 global web page icons 目錄。預設由 `StorePath` 推導為 `<StorePath>/icons`。
- `Dashboard.AuthUsersPath`：存放 `/admin` 管理的 direct-auth users JSON 檔案。預設由 `StorePath` 推導為 `<StorePath>/auth-users.json`。
- `Dashboard.AuthHelperConfigPath`：產生給 `rdpgw-auth` 使用的 helper YAML。預設由 `StorePath` 推導為 `<StorePath>/rdpgw-auth.yaml`。
- `Dashboard.AdminGroups`：允許進入 `/admin` 與 admin APIs 的 groups。
- `Dashboard.MaxUploadSizeMb`：template `.rdp` 與 icon file 的最大 request upload size。預設：`5`。
- `Dashboard.MaxTemplateUploads`：保存的 `.rdp` template files 數量上限。預設：`100`；`0` 表示停用此 quota。
- `Dashboard.MaxIconUploads`：保存的 icon files 數量上限。預設：`100`；`0` 表示停用此 quota。
- `Dashboard.MaxTemplateUploadStorageMb`：uploaded `.rdp` templates 的總 storage 上限。預設：`100`；`0` 表示停用此 quota。
- `Dashboard.MaxIconUploadStorageMb`：uploaded icon files 的總 storage 上限。預設：`100`；`0` 表示停用此 quota。

這些 dashboard group controls 是 Web/OIDC controls。Entry `AllowedGroups` 會用於 OIDC dashboard visibility 與 `.rdp` download access；`Dashboard.AdminGroups` 會用於 admin UI。它們不會套用到使用 direct authentication (`local` 或 `ntlm`) 的原生 RDP clients。Direct-auth host authorization 會使用 enabled dashboard host entries 的 addresses 作為 host allowlist，因為這些 auth modes 沒有可靠的 group claims。

範例：

```yaml
OpenId:
  ProviderUrl: https://keycloak.example.com/realms/homelab
  ClientId: rdpgw
  ClientSecret: your-secret
  GroupsClaim: groups
Dashboard:
  StorePath: /var/lib/rdpgw/dashboard
  UploadDir: /var/lib/rdpgw/dashboard/uploads
  IconDir: /var/lib/rdpgw/dashboard/icons
  AuthUsersPath: /var/lib/rdpgw/dashboard/auth-users.json
  AuthHelperConfigPath: /var/lib/rdpgw/dashboard/rdpgw-auth.yaml
  AdminGroups:
    - rdpgw-admins
    - homelab-admins
  MaxUploadSizeMb: 10
  MaxTemplateUploads: 100
  MaxIconUploads: 100
  MaxTemplateUploadStorageMb: 100
  MaxIconUploadStorageMb: 100
```

## Environment variable overrides

你可以用 environment variables override 相同設定：

- `RDPGW_OPENID__GROUPSCLAIM`
- `RDPGW_DASHBOARD__STOREPATH`
- `RDPGW_DASHBOARD__UPLOADDIR`
- `RDPGW_DASHBOARD__ICONDIR`
- `RDPGW_DASHBOARD__AUTHUSERSPATH`
- `RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH`
- `RDPGW_DASHBOARD__ADMINGROUPS`
- `RDPGW_DASHBOARD__MAXUPLOADSIZEMB`
- `RDPGW_DASHBOARD__MAXTEMPLATEUPLOADS`
- `RDPGW_DASHBOARD__MAXICONUPLOADS`
- `RDPGW_DASHBOARD__MAXTEMPLATEUPLOADSTORAGEMB`
- `RDPGW_DASHBOARD__MAXICONUPLOADSTORAGEMB`

注意：

- `RDPGW_DASHBOARD__ADMINGROUPS` 是空白分隔，例如：`rdpgw-admins homelab-admins`。
- 如果沒有設定 `RDPGW_DASHBOARD__UPLOADDIR`、`RDPGW_DASHBOARD__ICONDIR`、`RDPGW_DASHBOARD__AUTHUSERSPATH` 或 `RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH`，它們會由 `StorePath` 推導。
- 如果 runtime 設定了 `RDPGW_AUTH_HELPER_CONFIG`，gateway 會把產生的 helper YAML 寫到該位置，確保 helper read path 與 generated output path 一致。

## Direct Auth Management

Dashboard mode 啟用時，`/admin` 管理兩種 state：

- 給 OIDC users 使用的 published host/template entries
- 給 `ntlm` 與 `local` gateway logins 使用的 direct-auth users

每次 direct-auth user 變更後，server 會重新產生 helper YAML，`rdpgw-auth` 會在下一次 auth request 自動 reload。在 dashboard-managed direct-auth mode 中，enabled dashboard host entries 是 native RDP host allowlist。
Entry `AllowedGroups` 不會再限制這些 direct-auth connections；請只 publish 任何 valid direct-auth user 都應該能連到的 host entries。
如果 managed auth-user state 或 enabled host inventory 缺失或無效，direct `local` 與 `ntlm` auth 會 fail closed。

## Authentication flow

1. 使用者開啟 `https://your-gateway/`
2. Gateway redirect 到 OpenID Connect provider 進行 authentication
3. 使用者在 provider 完成 authentication（可支援 MFA）
4. Provider 帶 authentication token redirect 回 gateway
5. Gateway 驗證 token，並載入 dashboard 或 admin UI
6. Dashboard-managed direct-auth state 會給 native RDP clients 的 `local` 與 `ntlm` 使用

## Multi-Factor Authentication (MFA)

RDPGW 透過 OpenID Connect integration 支援 MFA。請在你的 identity provider 設定 MFA 以提升安全性。

## Provider examples

### Keycloak

```yaml
OpenId:
  ProviderUrl: https://keycloak.example.com/realms/your-realm
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

## Security considerations

- Production deployments 一律使用 HTTPS
- 安全保存 client secrets，並定期 rotate
- 在 provider 設定適當 scopes 與 claims
- 在 identity provider 啟用 MFA
- 在 gateway 與 provider 設定適當 session timeouts

## Troubleshooting

- 確認 gateway 能存取 `ProviderUrl`
- 確認 provider 裡設定的 redirect URI 正確，通常是 `https://your-gateway/callback`
- 檢查 required scopes，例如 openid、profile、email
- 確認 gateway 信任 provider 的 certificate
