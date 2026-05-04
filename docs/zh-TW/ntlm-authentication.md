# NTLM Authentication

RDPGW 支援給 Windows `mstsc` 等 direct RDP clients 使用的 NTLM authentication。在目前 HomeRDP Gateway 部署模型中，NTLM 與 `local` 是 *direct listener* auth modes：它們用於專門的 direct-auth listener；OIDC/dashboard listener 則處理 browser login 與 `.rdp` downloads。

## 目前部署模型

內建 sample 使用 split gateway mode：

- OIDC/dashboard listener 在 `8443`
- direct-auth listener 在 `9443`
- 共享 dashboard state：`/var/lib/rdpgw/dashboard`
- 非 container 部署時共享 `rdpgw-auth` helper socket：`/run/rdpgw/rdpgw-auth.sock`；container 部署則使用 container writable `/tmp` runtime directory 下的 socket

當設定 `GatewaySplit.Enabled: true` 時，container entrypoint 會啟動兩個 listeners；當 direct listener 使用 `ntlm` 或 `local` 時，也會自動啟動 `rdpgw-auth`。

## 什麼時候使用 NTLM

當你需要以下能力時使用 NTLM：

- Windows `mstsc` compatibility
- 實際 RDP tunnel 不經 browser OIDC，而是 direct RDP login
- dashboard-managed direct-auth users

如果你的 clients 可以使用 browser SSO，請在 dashboard listener 使用 OIDC；direct listener 則保留給 native RDP access。

## 設定

### 最小 direct-auth config

```yaml
Server:
  Authentication:
    - ntlm
  AuthSocket: /run/rdpgw/rdpgw-auth.sock
  SecureCookies: true
Caps:
  TokenAuth: false
```

注意：

- `Server.AuthSocket` 必須符合 `rdpgw-auth` 使用的 socket path。
- HTTPS reverse proxy 在 rdpgw 前方終止 TLS 時，建議使用 `Server.SecureCookies: true`。
- Pure direct-auth listener 應保持 `Caps.TokenAuth: false`。

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

如果你想要 `local` direct listener，把 `GatewaySplit.Direct.Authentication` 裡的 `ntlm` 換成 `local`。

## 現在 direct-auth user management 如何運作

支援的 workflow 是 dashboard-managed：

1. 在 OIDC/dashboard listener 登入 `/admin`。
2. 在 Direct Auth Users 區塊建立或更新 direct-auth users。
3. Server 會在 dashboard store 下寫入 `auth-users.json`。
4. Server 會自動重新產生 `rdpgw-auth.yaml`。
5. `rdpgw-auth` 會在下一次 auth request reload generated file。

在支援的部署模式中，不要手動編輯 generated helper YAML 並把它當成 source of truth。

## Authentication flow

1. Client 連到 `9443` direct listener。
2. Gateway 透過 Unix socket 把 NTLM handshake forward 給 `rdpgw-auth`。
3. `rdpgw-auth` 根據 generated direct-auth user list 驗證 credentials。
4. 驗證成功後，client 連到 target host。

## Docker deployment

Root `docker-compose.yml` 使用已發布的 GHCR image，以 UID/GID `1001` 執行，並把 `/tmp` mount 成 writable tmpfs。當 config 在 direct listener 啟用 `ntlm` 或 `local` 時，container entrypoint 會自動啟動 `rdpgw-auth`。

如果使用非 container 部署，請另外啟動 `rdpgw-auth`，並讓它使用相同 socket 與 generated helper config path。

## Windows client notes

- 在 `mstsc` 使用 direct listener hostname 與 port `9443`。
- 依提示儲存 gateway credentials。
- 如果你測試的是 TLS-terminating proxy，請保持 `Server.SecureCookies: true`。

## Troubleshooting

### Direct auth 失敗

檢查：

- mounted config 裡是否有 `GatewaySplit.Enabled: true`
- `GatewaySplit.Direct.Authentication` 是否包含 `ntlm` 或 `local`
- direct listener port 是否符合 service 或 port-forward target
- `rdpgw-auth` 是否正在執行，或 container entrypoint 是否有啟動它
- auth socket path 是否可寫入
- `/admin` 是否存在且啟用 direct-auth user
- direct-auth host entry 是否啟用

### Helper config 看起來過期

檢查：

- `Dashboard.StorePath` 是否指向 mounted dashboard volume
- `Dashboard.AuthHelperConfigPath` 是否推導或設定為 helper 讀取的相同 shared path
- dashboard store volume 是否可被 UID `1001` 寫入

## Security notes

- Direct-auth passwords 是敏感資料，應視為 credential data。
- 保護 dashboard store 與 backups。
- 除非是刻意設計，否則不要把 direct listener 直接暴露到 untrusted networks。
- 優先使用 TLS termination 或 trusted private network；當 browser sessions 經過 TLS terminator 時，保持 `Server.SecureCookies: true`。
