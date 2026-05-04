# Security Policy

## 專案範圍

HomeRDP Gateway 是面向 homelab 的 Remote Desktop Gateway fork，適用於個人與 self-hosted 環境。

這個專案處理 authentication、gateway access、host authorization 與 remote desktop connectivity，因此非常歡迎安全回報。

## 支援版本

在專案建立正式 releases 前，安全修正會先在主要開發分支處理。

等正式 release 流程建立後，這一節應再補上 supported-version table。

## 回報漏洞

如果你懷疑發現 security vulnerability，請先私下聯絡 maintainer，不要先公開揭露。

Maintainer：

- GitHub: https://github.com/tsunheimat

建議回報內容：

- 受影響版本或 commit hash
- 部署模式與設定摘要
- 重現步驟
- 預期行為
- 實際行為
- 潛在影響
- 相關 logs、screenshots 或 proof-of-concept 細節

如果漏洞可能涉及 authentication bypass、authorization bypass、token handling 問題、remote code execution、credential leaks 或 host access-control bypasses，請避免開 public issue。

## 安全敏感區域

特別歡迎 review 以下區域：

- OIDC login 與 callback handling
- Gateway token generation 與 validation
- Cookie 與 session handling
- Host authorization 與 access-control logic
- RDP template 與 icon upload handling
- File parsing 與 storage paths
- Cross-gateway permission synchronization
- Reverse proxy 與 TLS deployment assumptions
- mstsc 與 macOS client compatibility flows

## 非目標

這個專案面向 homelab 與個人基礎設施使用情境。

它不打算提供與 Microsoft Windows Server Remote Desktop Gateway 或 Microsoft Remote Desktop Services 相同等級的支援、compliance guarantees、management surface 或 enterprise lifecycle。

## Disclosure process

Maintainer 會盡量確認有效回報，並在可行時協調修復。

如果你打算公開揭露漏洞，請先給合理時間做調查與修補。
