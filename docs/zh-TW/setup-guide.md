# HomeRDP Gateway 設定指南

這份指南會帶你使用已發布的 GHCR image 與 repo 內建 sample manifest，完成第一個可運作的 HomeRDP Gateway 部署。

HomeRDP Gateway 主要面向 homelab 與 self-hosted 部署。Sample files 只提供安全 placeholder，不是可直接上 production 的 secret。把 gateway 暴露給其他使用者或網路前，請替換每一個 `CHANGE_ME` 值。

## 部署選擇

你可以從以下其中一條路徑開始：

| 路徑 | 適合情境 | 檔案 |
| --- | --- | --- |
| Docker Compose | 本機測試、單機 homelab 試用 | [`docker-compose.yml`](../../docker-compose.yml), [`dev/docker/rdpgw.yaml`](../../dev/docker/rdpgw.yaml) |
| Kubernetes | 需要 ConfigMap / PVC wiring 的 cluster 部署 | [`k8s/rdpgw.yaml`](../../k8s/rdpgw.yaml) |
| Source build | 開發與自訂 image | [`dev/docker/Dockerfile`](../../dev/docker/Dockerfile), `make build` |

已發布 image：

```text
ghcr.io/tsunheimat/homerdp-gateway:latest
```

Image 以 UID/GID `1001` 執行，只有明確 mount 的 runtime storage 需要可寫入。

## 開始前

1. 決定公開 gateway hostname 與 ports。
   - Browser/OIDC/dashboard listener：sample 通常使用 `8443`。
   - Direct RDP gateway listener：sample 通常使用 `9443`。
2. 在你的 IdP 建立 OpenID Connect client。
   - Redirect/callback URL 要符合外部 OIDC gateway URL。
   - Callback URL 會從對外宣告的 `GatewayAddress` 或 `GatewaySplit.OIDC.Hostname` 推導，例如 `https://rdapp.example.com/callback`。
   - Client secret 要保密。
   - 如果使用 dashboard group filtering，token 需要包含你在 `OpenId.GroupsClaim` 設定的 group claim。
3. 決定 TLS 如何處理。
   - 如果由 HomeRDP Gateway 自己處理 TLS，設定 certificate/key paths。
   - 如果由 reverse proxy 或 ingress 終止 TLS，backend 要保持私有，backend 可用 `Server.Tls: disable`，並設定 `Server.SecureCookies: true`。
4. 為每個 deployment-specific secret 產生新的 32 字元 key。

範例 key generation：

```bash
openssl rand -hex 16
```

分別替以下欄位使用不同值：

- `Server.SessionKey`
- `Server.SessionEncryptionKey`
- `Security.PAATokenSigningKey`
- `Security.PAATokenEncryptionKey`

## Sample configuration 行為

內建 Docker 與 Kubernetes samples 會刻意使用 placeholder OIDC settings 與 secrets。它們用來展示目前 homelab split-gateway topology，不是 production-ready 設定。

重要 sample settings：

- `GatewaySplit.Enabled: true` 會讓 container entrypoint 從同一份 `rdpgw.yaml` 啟動兩個 gateway process：
  - OIDC/dashboard listener：`GatewaySplit.OIDC.Port`，sample 是 `8443`
  - direct-auth listener：`GatewaySplit.Direct.Port`，sample 是 `9443`
- `GatewaySplit` 由 container entrypoint script 消費，Go config schema 不會直接使用它。Entrypoint 啟動各 listener 時，會 export per-listener 的 `RDPGW_SERVER__PORT`、`RDPGW_SERVER__GATEWAYADDRESS`、`RDPGW_SERVER__AUTHENTICATION`、`RDPGW_CAPS__TOKENAUTH` overrides。
- 如果移除 `GatewaySplit`，只會在 `Server.Port` 啟動一個 gateway process。Docker Compose 或 Kubernetes expose `9443` 不會自動建立 direct listener。
- `GatewaySplit.Direct.Authentication` 控制 direct listener auth modes。Dashboard-managed direct-auth users 建議用 `ntlm` 或 `local`；sample 使用 `ntlm`，因為 Windows `mstsc` 支援。
- `Server.SessionStore: file` 會把 session data 存到 `/tmp`；兩個 sample deployments 都把 `/tmp` mount 成 writable runtime storage。如果省略，application default 是 `cookie`。
- `Caps.EnableClipboard`、`Caps.EnableDrive`、`Caps.EnablePrinter`、`Caps.EnablePort`、`Caps.EnablePnp` 會宣告 RDP redirection capabilities。省略時這些 boolean 預設為 `false`。
- `Dashboard.AuthHelperConfigPath` 如果等於 `<Dashboard.StorePath>/rdpgw-auth.yaml` 可以省略；application 與 entrypoint 會自動推導。
- `Security.VerifyClientIp` 預設為 `true`；只有在 trusted proxy 無法提供穩定 client IP 給 token validation 時才考慮關閉。

## Docker Compose 快速開始

1. Clone repo 並進入目錄：

```bash
git clone https://github.com/tsunheimat/HomeRDP_Gateway.git
cd HomeRDP_Gateway
```

2. 編輯 sample config：

```bash
cp dev/docker/rdpgw.yaml /tmp/rdpgw.yaml.edit
# edit /tmp/rdpgw.yaml.edit, then copy reviewed values back intentionally
```

至少替換：

- `Server.GatewayAddress`
- 每一個 `CHANGE_ME...` key 或 secret
- `OpenId.ProviderUrl`
- `OpenId.ClientId`
- `OpenId.ClientSecret`
- `Dashboard.AdminGroups`
- `Server.Hosts`
- `GatewaySplit.OIDC.Hostname`
- `GatewaySplit.Direct.Hostname`
- 如果你想用 `local`、`ntlm` 或其他 direct-auth mode，調整 `GatewaySplit.Direct.Authentication`

確認後把設定寫回 mounted sample path：

```bash
cp /tmp/rdpgw.yaml.edit dev/docker/rdpgw.yaml
```

3. 建立 dashboard data 目錄，並讓 UID `1001` 可寫入：

```bash
mkdir -p data/dashboard
sudo chown -R 1001:1001 data/dashboard
```

如果沒有 `sudo`，請在 container runtime 能以 UID `1001` 寫入的 filesystem 建立該目錄。

4. 從 repo root 啟動 sample：

```bash
docker compose up
```

預期 startup logs 會包含 `Split gateway mode enabled`，一個 `rdpgw-oidc` process 在 port `8443`，以及一個 `rdpgw-direct` process 在 port `9443`。

5. 檢查 listeners：

```bash
curl http://localhost:8443/
curl http://localhost:9443/
```

6. 開啟 admin UI：

```text
http://localhost:8443/admin
```

7. 登入後，至少建立一個 enabled dashboard entry。如果要測試 direct-auth / native RDP，也要在 admin UI 建立 direct-auth user。

## Kubernetes 快速開始

起始 manifest 是 [`k8s/rdpgw.yaml`](../../k8s/rdpgw.yaml)。它包含：

- 內含 `rdpgw.yaml` 的 `ConfigMap`
- 使用 `ghcr.io/tsunheimat/homerdp-gateway:latest` 的 `Deployment`
- 保存 dashboard state 的 `PersistentVolumeClaim`
- 掛載到 `/tmp` 的 `emptyDir`，給 runtime/session files 與 auth helper socket 使用
- expose ports `8443` 與 `9443` 的 `ClusterIP` `Service`

Kubernetes sample 會 mirror split-gateway Docker topology。如果你預期兩個 service ports 都有 live listeners，`GatewaySplit` 必須保持 enabled。如果只需要 OIDC/dashboard listener，請同時移除 direct container port、Service port 與 `GatewaySplit.Direct` block。

1. 複製 manifest：

```bash
cp k8s/rdpgw.yaml /tmp/rdpgw-k8s.yaml
```

2. 編輯 `/tmp/rdpgw-k8s.yaml`，替換 `ConfigMap` 的 `rdpgw.yaml` key 下所有 placeholder config values。

Production 建議用 Kubernetes `Secret` objects 保存 OIDC client secrets、token/session keys 等敏感值。Checked-in ConfigMap 只是為了單檔 starter example 才把內容全部顯示出來。

如果要把既有 running config 轉成 Kubernetes sample，且希望 runtime behavior 一致，請特別搬移以下 behavior-sensitive fields：

- `GatewaySplit.Enabled`, `GatewaySplit.OIDC`, `GatewaySplit.Direct`
- `GatewaySplit.Direct.Authentication`
- `Server.SessionStore`
- `Caps.EnableClipboard`, `Caps.EnableDrive`, `Caps.EnablePrinter`, `Caps.EnablePort`, `Caps.EnablePnp`
- `Server.GatewayAddress` 與兩個 `GatewaySplit.*.Hostname` values
- 真實 OIDC provider/client values 與所有四個 32-character keys

3. Apply manifest：

```bash
kubectl apply -f /tmp/rdpgw-k8s.yaml
```

4. 檢查 rollout 與 pods：

```bash
kubectl rollout status deployment/rdpgw
kubectl get pods -l app.kubernetes.io/name=rdpgw
kubectl logs deployment/rdpgw
```

當 `GatewaySplit.Enabled: true` 存在時，logs 應顯示 split mode 與兩個 gateway processes。

5. 透過你的 ingress、Gateway API、load balancer 或 port-forward path expose service。

Sample manifest 使用 `Server.Tls: disable` 與 `Server.SecureCookies: true`，所以 browser login 最適合由 TLS-terminating ingress 或 load balancer 提供 cluster-facing endpoint。如果只是短暫 port-forward smoke test，可能需要在 throwaway manifest copy 裡把 `Server.SecureCookies` 改成 `false`，或在 port-forward 前放一個本機 TLS terminator。

快速 private test：

```bash
kubectl port-forward service/rdpgw 8443:8443 9443:9443
```

接著開啟外部 HTTPS URL；如果直接用 forwarded HTTP backend，只適合 throwaway smoke testing。當 `Server.SecureCookies: true` 還開著時，不要期待 secure cookies 在 plain HTTP port-forward 正常運作。

```text
https://rdpgw.example.invalid/admin
```

## 既有 production-style config migration checklist

如果你要把已知可運作的非 Kubernetes config 轉到 Kubernetes sample，不要只 copy 明顯的 OIDC values。請 review 每個會改變 listener 行為或 RDP capabilities 的欄位：

| 欄位 | 為什麼重要 |
| --- | --- |
| `GatewaySplit.*` | 啟動並設定分開的 OIDC 與 direct-auth listeners。沒有它時只會啟動 `Server.Port`。 |
| `GatewaySplit.Direct.Authentication` | 選擇 direct listener auth stack，例如 `ntlm`、`local`。 |
| `Server.SessionStore` | 在 filesystem 與 encrypted cookies 間切換 browser/session storage。 |
| `Caps.Enable*` | 控制 clipboard、drive、printer、port、PnP redirection flags。 |
| `Server.SecureCookies` | 當使用者從 HTTPS TLS terminator 存取，但 rdpgw 看到 backend HTTP 時需要。 |
| `Dashboard.StorePath` | 必須指向 mounted writable volume；derived dashboard paths 都在它下面。 |
| `Dashboard.AuthHelperConfigPath` | 等於 `<StorePath>/rdpgw-auth.yaml` 時可省略；只有 custom helper config path 才需要明確設定。 |
| `Security.VerifyClientIp` | 預設 `true`；只有 trusted proxy 讓 forwarded client IP 不穩定時才關閉。 |
| `Server.Hosts` | Optional static host allowlist；homelab UI / direct-auth workflow 可以改用 dashboard-managed hosts。 |

## Native RDP / direct-auth 測試

建立 enabled host entry 與 direct-auth user 後，用 FreeRDP 測試：

```bash
xfreerdp /g:<gateway-host>:9443 /gd:"" /u:<direct-auth-user> /p:<direct-auth-password> /v:<enabled-dashboard-host> /cert-ignore
```

注意：

- Windows `mstsc` 不支援 gateway basic authentication；依部署使用 OpenID Connect 或 NTLM。
- Windows clients 對 TLS 與 certificates 通常比測試工具更嚴格。
- `/v:` 的 host 應符合 enabled dashboard host entry 或其他 configured allow-list target。

## Hardening checklist

從本機測試轉到正式使用前：

- 將所有 placeholder secrets 換成每個 deployment 專用的 random values。
- 外部 HTTPS 存取時保持 `Server.SecureCookies: true`，包含 reverse-proxy TLS termination。
- 把 plain HTTP backend listener 限制在 trusted private networks。
- Writable mounts 限制在 `/tmp`、`/var/lib/rdpgw/dashboard`，以及你有意使用的 certificate cache path。
- 保持 container runtime security settings：non-root UID `1001`、dropped capabilities、no privilege escalation、相容時使用 read-only root filesystem。
- 安全備份 dashboard state；它可能包含 privileged routing policy 與 direct-auth credential material。
- 如果啟用 `/metrics`，請保護它。
- 將 dashboard access 發布給 users 前，review entries 與 group filters。

## Troubleshooting

### Container 立即退出

檢查 logs：

```bash
docker compose logs rdpgw
kubectl logs deployment/rdpgw
```

常見原因：

- YAML indentation 錯誤
- 需要 32 字元的 secrets 長度不正確
- dashboard directory 無法被 UID/GID `1001` 寫入
- OIDC provider URL 或 client settings 不匹配

### `/admin` login loop 或 session 消失

檢查：

- 你是否透過 HTTPS 存取 gateway，但 `Server.SecureCookies` 設成 `false`
- 你是否透過 HTTP port-forward 存取，但 `Server.SecureCookies` 還是 `true`
- `Server.GatewayAddress` 或 `GatewaySplit.OIDC.Hostname` 是否符合實際 browser URL
- reverse proxy 是否保留需要的 host / scheme headers

### Direct listener 沒反應

檢查：

- `GatewaySplit.Enabled: true` 是否存在
- `GatewaySplit.Direct.Port` 是否符合 Docker/Kubernetes expose 的 port
- startup logs 是否顯示 `rdpgw-direct`
- service、ingress 或 port-forward 是否真的把 `9443` 導到 pod/container

## 相關文件

- [`../../README.zh-TW.md`](../../README.zh-TW.md)
- [`openid-authentication.md`](./openid-authentication.md)
- [`ntlm-authentication.md`](./ntlm-authentication.md)
- [`security.md`](./security.md)
