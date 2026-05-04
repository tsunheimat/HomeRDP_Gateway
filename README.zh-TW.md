# HomeRDP Gateway

HomeRDP Gateway 是給 homelab 與 self-hosted 環境使用的簡單 Remote Desktop Gateway。

它提供：

- 用來管理 RDP 主機的瀏覽器 dashboard
- Web UI 的 OpenID Connect 登入
- 管理主機項目、圖示、RDP template、direct-auth 使用者的 admin 頁面
- 給 Windows `mstsc` 這類原生 RDP client 使用的 direct RDP gateway listener
- Docker Compose 與 Kubernetes 起始範例

這個專案是給個人基礎設施與小型 self-hosted 網路使用。

## 運作方式

典型部署會有兩個 listener：

| Listener | 預設 port | 用途 |
| --- | --- | --- |
| Web / OIDC dashboard | `8443` | 瀏覽器登入、dashboard、admin UI、下載 `.rdp` 檔案 |
| Direct RDP gateway | `9443` | 使用 NTLM 等 direct auth 的原生 RDP client |

Dashboard 控制主機清單。Direct-auth 使用者與啟用的主機項目會在 `/admin` 管理。

## Docker Compose 快速開始

1. 編輯本機範例設定：

```bash
$EDITOR dev/docker/rdpgw.yaml
```

至少需要替換：

- OIDC provider URL、client ID、client secret
- 所有 `CHANGE_ME...` key
- `GatewayAddress` / `GatewaySplit` hostname
- admin groups
- 目標 RDP hosts

2. 建立 dashboard data 目錄：

```bash
mkdir -p data/dashboard
sudo chown -R 1001:1001 data/dashboard
```

3. 啟動 gateway：

```bash
docker compose up
```

4. 開啟 admin UI：

```text
http://localhost:8443/admin
```

本機範例為了方便測試會停用 backend TLS。正式使用時，請放在 HTTPS reverse proxy / ingress / load balancer 後面，並設定 `Server.SecureCookies: true`。

## Container image

```text
ghcr.io/tsunheimat/homerdp-gateway:latest
```

## 設定檔

重要起始檔案：

- `dev/docker/rdpgw.yaml` — 本機 Docker 範例設定
- `docker-compose.yml` — 本機 compose 部署
- `k8s/rdpgw.yaml` — Kubernetes 起始 manifest

重要設定區塊：

- `Server` — listener、auth、session、TLS/cookie 設定
- `OpenId` — OIDC provider 與 client 設定
- `Dashboard` — dashboard storage 與 admin groups
- `Security` — token signing/encryption keys
- `Caps` — clipboard、drive 等 RDP redirection capabilities
- `GatewaySplit` — container entrypoint 用來啟動 web + direct listeners 的輔助設定

完整 walkthrough 請看 [`docs/zh-TW/setup-guide.md`](./docs/zh-TW/setup-guide.md)。英文版在 [`docs/setup-guide.md`](./docs/setup-guide.md)。

## 原生 RDP client

在 `/admin` 建立啟用的 host entry 與 direct-auth user 後，可以用 FreeRDP 測試：

```bash
xfreerdp /g:<gateway-host>:9443 /gd:"" /u:<direct-auth-user> /p:<direct-auth-password> /v:<enabled-dashboard-host> /cert-ignore
```

Windows `mstsc` 通常最適合搭配 NTLM direct auth 與儲存的 gateway credentials。

## Runtime hardening

內建 Docker Compose 與 Kubernetes 範例使用 homelab-friendly 的安全預設：

- non-root UID/GID `1001:1001`
- `readOnlyRootFilesystem` / read-only root filesystem
- `allowPrivilegeEscalation: false`
- Linux capabilities 使用 `drop: ["ALL"]` 全部移除
- Kubernetes `runAsNonRoot: true`
- `rdpgw-auth helper runs as the same non-root user`，image `does not set the setuid bit`

只有 dashboard data 目錄需要可寫入。

## 安全注意事項

在把 gateway 暴露到本機測試以外的環境前：

- 替換每一個 placeholder secret
- 所有 32 字元 key 都要穩定保存且保密
- 使用 reverse proxy、ingress 或 load balancer 提供 HTTPS
- 當使用者透過 HTTPS 存取 gateway 時，設定 `Server.SecureCookies: true`
- 保護 dashboard data 目錄與備份
- 將 direct-auth password 視為敏感 credential data

## 從 source build

```bash
make build
go test ./...
```

預設 `make` target 會把兩個 gateway binaries build 到 `bin/`。

## 更多文件

- [`docs/zh-TW/README.md`](./docs/zh-TW/README.md) — 繁體中文文件索引
- [`docs/zh-TW/setup-guide.md`](./docs/zh-TW/setup-guide.md)
- [`docs/zh-TW/openid-authentication.md`](./docs/zh-TW/openid-authentication.md)
- [`docs/zh-TW/ntlm-authentication.md`](./docs/zh-TW/ntlm-authentication.md)
- [`docs/zh-TW/security.md`](./docs/zh-TW/security.md)

英文文件：

- [`docs/setup-guide.md`](./docs/setup-guide.md)
- [`docs/openid-authentication.md`](./docs/openid-authentication.md)
- [`docs/ntlm-authentication.md`](./docs/ntlm-authentication.md)
- [`SECURITY.md`](./SECURITY.md)
- [`CONTRIBUTING.md`](./CONTRIBUTING.md)

## License 與 attribution

HomeRDP Gateway 使用 Apache License 2.0。Attribution 與專案歷史請看 [`NOTICE`](./NOTICE) 與 [`MODIFICATIONS.md`](./MODIFICATIONS.md)。
