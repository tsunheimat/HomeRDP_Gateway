# HomeRDP Gateway setup guide

This guide walks through a first working HomeRDP Gateway deployment using the published GHCR image and the checked-in sample manifests.

HomeRDP Gateway is intended for homelab and self-hosted deployments. The sample files are safe placeholders, not production-ready secrets. Replace every `CHANGE_ME` value before exposing the gateway to other users or networks.

## Deployment choices

Use one of these starter paths:

| Path | Best for | Files |
| --- | --- | --- |
| Docker Compose | Local testing, single-host homelab trials | [`docker-compose.yml`](../docker-compose.yml), [`dev/docker/rdpgw.yaml`](../dev/docker/rdpgw.yaml) |
| Kubernetes | Cluster deployment with ConfigMap/PVC wiring | [`k8s/rdpgw.yaml`](../k8s/rdpgw.yaml) |
| Source build | Development and custom image work | [`dev/docker/Dockerfile`](../dev/docker/Dockerfile), `make build` |

The published image is:

```text
ghcr.io/tsunheimat/homerdp-gateway:latest
```

The image runs as UID/GID `1001` and expects writable runtime storage only where explicitly mounted.

## Before you start

1. Pick public gateway hostnames and ports.
   - Browser/OIDC/dashboard listener: usually `8443` in the samples.
   - Direct RDP gateway listener: usually `9443` in the samples.
2. Prepare an OpenID Connect client in your IdP.
   - Configure redirect/callback URL to match your external OIDC gateway URL.
   - The callback URL is derived from the listener's externally advertised `GatewayAddress` or `GatewaySplit.OIDC.Hostname`; for example `https://rdapp.example.com/callback`.
   - Keep the client secret private.
   - Ensure the token contains the group claim you configure as `OpenId.GroupsClaim` if you use group-based dashboard filtering.
3. Decide how TLS is handled.
   - If HomeRDP Gateway handles TLS directly, configure certificate/key paths.
   - If a reverse proxy or ingress terminates TLS, keep the backend private, use `Server.Tls: disable` for that backend, and set `Server.SecureCookies: true`.
4. Generate fresh 32-character keys for every deployment-specific secret.

Example key generation:

```bash
openssl rand -hex 16
```

Use separate values for:

- `Server.SessionKey`
- `Server.SessionEncryptionKey`
- `Security.PAATokenSigningKey`
- `Security.PAATokenEncryptionKey`

## Sample configuration behavior

The checked-in Docker and Kubernetes samples intentionally use placeholder OIDC settings and secrets. They are designed to show the current homelab split-gateway topology, not to be production-ready.

Important sample settings:

- `GatewaySplit.Enabled: true` makes the container entrypoint start two gateway processes from the same `rdpgw.yaml`:
  - an OIDC/dashboard listener on `GatewaySplit.OIDC.Port` (`8443` in the samples)
  - a direct-auth listener on `GatewaySplit.Direct.Port` (`9443` in the samples)
- `GatewaySplit` is consumed by the container entrypoint script. The Go configuration schema does not use it directly. When the entrypoint starts each process, it exports per-listener `RDPGW_SERVER__PORT`, `RDPGW_SERVER__GATEWAYADDRESS`, `RDPGW_SERVER__AUTHENTICATION`, and `RDPGW_CAPS__TOKENAUTH` overrides.
- If you remove `GatewaySplit`, only one gateway process is started on `Server.Port`. Exposing port `9443` in Docker Compose or Kubernetes does not by itself start a direct listener.
- `GatewaySplit.Direct.Authentication` controls the direct listener auth modes. Use `ntlm` or `local` for dashboard-managed direct-auth users; the sample uses `ntlm` because Windows `mstsc` supports it.
- `Server.SessionStore: file` stores session data under `/tmp`; both sample deployments mount `/tmp` as writable runtime storage. The application default is `cookie` if this setting is omitted.
- `Caps.EnableClipboard`, `Caps.EnableDrive`, `Caps.EnablePrinter`, `Caps.EnablePort`, and `Caps.EnablePnp` advertise RDP redirection capabilities. If omitted, those booleans default to `false`.
- `Dashboard.AuthHelperConfigPath` can be omitted when it should be `<Dashboard.StorePath>/rdpgw-auth.yaml`; the application and entrypoint derive that path automatically.
- `Security.VerifyClientIp` defaults to `true`; keep it enabled unless your trusted proxy product cannot provide stable client IPs for token validation.

## Docker Compose quick start

1. Clone the repository and enter it:

```bash
git clone https://github.com/tsunheimat/HomeRDP_Gateway.git
cd HomeRDP_Gateway
```

2. Edit the sample config:

```bash
cp dev/docker/rdpgw.yaml /tmp/rdpgw.yaml.edit
# edit /tmp/rdpgw.yaml.edit, then copy reviewed values back intentionally
```

At minimum, replace:

- `Server.GatewayAddress`
- every `CHANGE_ME...` key or secret
- `OpenId.ProviderUrl`
- `OpenId.ClientId`
- `OpenId.ClientSecret`
- `Dashboard.AdminGroups`
- `Server.Hosts`
- `GatewaySplit.OIDC.Hostname`
- `GatewaySplit.Direct.Hostname`
- `GatewaySplit.Direct.Authentication` if you want `local`, `ntlm`, or another direct-auth mode

Then write the reviewed config back to the mounted sample path:

```bash
cp /tmp/rdpgw.yaml.edit dev/docker/rdpgw.yaml
```

3. Create the dashboard data directory and make it writable by UID `1001`:

```bash
mkdir -p data/dashboard
sudo chown -R 1001:1001 data/dashboard
```

If you do not have `sudo`, create the directory on a filesystem where your container runtime can write as UID `1001`.

4. Start the sample from the repository root:

```bash
docker compose up
```

Expected startup logs include `Split gateway mode enabled`, one `rdpgw-oidc` process on port `8443`, and one `rdpgw-direct` process on port `9443`.

5. Check the listeners:

```bash
curl http://localhost:8443/
curl http://localhost:9443/
```

6. Open the admin UI:

```text
http://localhost:8443/admin
```

7. After login, create at least one enabled dashboard entry. If you want direct-auth/native RDP testing, also create a direct-auth user in the admin UI.

## Kubernetes quick start

The starter manifest is [`k8s/rdpgw.yaml`](../k8s/rdpgw.yaml). It includes:

- a `ConfigMap` containing `rdpgw.yaml`
- a `Deployment` using `ghcr.io/tsunheimat/homerdp-gateway:latest`
- a `PersistentVolumeClaim` for dashboard state
- an `emptyDir` mounted at `/tmp` for runtime/session files and the auth helper socket
- a `ClusterIP` `Service` exposing ports `8443` and `9443`

The Kubernetes sample mirrors the split-gateway Docker topology. `GatewaySplit` must remain enabled if you expect both service ports to have live listeners. If you only want the OIDC/dashboard listener, remove the direct port from the container and Service as well as the `GatewaySplit.Direct` block.

1. Copy the manifest for your environment:

```bash
cp k8s/rdpgw.yaml /tmp/rdpgw-k8s.yaml
```

2. Edit `/tmp/rdpgw-k8s.yaml` and replace all placeholder config values under the `ConfigMap` `rdpgw.yaml` key.

For production, prefer Kubernetes `Secret` objects for sensitive values such as OIDC client secrets and token/session keys. The checked-in ConfigMap keeps everything visible for a single-file starter example only.

When adapting an existing running config, carry over these behavior-sensitive fields if you need the same runtime behavior:

- `GatewaySplit.Enabled`, `GatewaySplit.OIDC`, and `GatewaySplit.Direct`
- `GatewaySplit.Direct.Authentication`
- `Server.SessionStore`
- `Caps.EnableClipboard`, `Caps.EnableDrive`, `Caps.EnablePrinter`, `Caps.EnablePort`, and `Caps.EnablePnp`
- `Server.GatewayAddress` and the two `GatewaySplit.*.Hostname` values
- real OIDC provider/client values and all four 32-character keys

3. Apply the manifest:

```bash
kubectl apply -f /tmp/rdpgw-k8s.yaml
```

4. Check rollout and pods:

```bash
kubectl rollout status deployment/rdpgw
kubectl get pods -l app.kubernetes.io/name=rdpgw
kubectl logs deployment/rdpgw
```

The logs should show split mode and both gateway processes when `GatewaySplit.Enabled: true` is present.

5. Expose the service through your preferred ingress, gateway API, load balancer, or port-forwarding path.

The sample manifest uses `Server.Tls: disable` and `Server.SecureCookies: true`, so browser login works best when the cluster-facing endpoint is fronted by a TLS-terminating ingress or load balancer. If you are only doing a temporary port-forward smoke test, you may need to flip `Server.SecureCookies` to `false` in a throwaway copy of the manifest or place a local TLS terminator in front of the port-forward.

For a quick private test:

```bash
kubectl port-forward service/rdpgw 8443:8443 9443:9443
```

Then open the externally terminated HTTPS URL, or use the forwarded HTTP backend directly only for throwaway smoke testing. Do not expect secure cookies to survive a plain HTTP port-forward when `Server.SecureCookies: true` is still set.

```text
https://rdpgw.example.invalid/admin
```

## Existing production-style config migration checklist

If you are translating a known-good non-Kubernetes config into the Kubernetes sample, do not blindly copy only the obvious OIDC values. Review every field that changes listener behavior or RDP capabilities:

| Field | Why it matters |
| --- | --- |
| `GatewaySplit.*` | Starts and configures separate OIDC and direct-auth listeners. Without it, only `Server.Port` starts. |
| `GatewaySplit.Direct.Authentication` | Selects the direct listener auth stack (`ntlm`, `local`, etc.). |
| `Server.SessionStore` | Changes browser/session storage between filesystem and encrypted cookies. |
| `Caps.Enable*` | Controls clipboard, drive, printer, port, and PnP redirection flags sent to RDP clients. |
| `Server.SecureCookies` | Required when users access HTTPS through a TLS terminator but rdpgw sees backend HTTP. |
| `Dashboard.StorePath` | Must point at a mounted writable volume; derived dashboard paths live below it. |
| `Dashboard.AuthHelperConfigPath` | Optional when it equals `<StorePath>/rdpgw-auth.yaml`; required only for a custom helper config path. |
| `Security.VerifyClientIp` | Defaults to `true`; disable only when a trusted proxy makes forwarded client IP unstable. |
| `Server.Hosts` | Optional static host allowlist; dashboard-managed hosts may be used instead for the homelab UI/direct-auth workflow. |

## Native RDP/direct-auth test

After creating an enabled host entry and a direct-auth user, test with FreeRDP:

```bash
xfreerdp /g:<gateway-host>:9443 /gd:"" /u:<direct-auth-user> /p:<direct-auth-password> /v:<enabled-dashboard-host> /cert-ignore
```

Notes:

- Windows `mstsc` does not support basic authentication for the gateway; use OpenID Connect or NTLM depending on your deployment.
- Windows clients are stricter about TLS and certificates than many test tools.
- The host in `/v:` should match an enabled dashboard host entry or another configured allow-list target.

## Hardening checklist

Before moving beyond local testing:

- Replace all placeholder secrets with random per-deployment values.
- Keep `Server.SecureCookies: true` for HTTPS external access, including reverse-proxy TLS termination.
- Restrict any plain HTTP backend listener to trusted private networks only.
- Limit writeable mounts to `/tmp`, `/var/lib/rdpgw/dashboard`, and any certificate cache path you intentionally use.
- Keep container runtime security settings enabled: non-root UID `1001`, dropped capabilities, no privilege escalation, and read-only root filesystem where compatible.
- Back up dashboard state securely; it can contain privileged routing policy and direct-auth credential material.
- Protect `/metrics` if you enable it.
- Review entries and group filters before publishing dashboard access to users.

## Troubleshooting

### Container exits immediately

Check logs:

```bash
docker compose logs rdpgw
kubectl logs deployment/rdpgw
```

Common causes:

- invalid YAML indentation
- secrets not exactly 32 characters where required
- dashboard directory not writable by UID/GID `1001`
- OIDC provider URL or client settings mismatch

### Browser login loops or callback fails

Check:

- `Server.GatewayAddress` or `GatewaySplit.OIDC.Hostname` matches the externally visible OIDC URL
- IdP redirect URL matches the gateway callback URL
- reverse proxy forwards the expected scheme/host
- `Server.SecureCookies` is true when users access the gateway over HTTPS but the backend receives HTTP

### Service exposes 9443 but direct RDP does not connect

Check:

- `GatewaySplit.Enabled: true` is present in the mounted `rdpgw.yaml`
- `GatewaySplit.Direct.Port` is `9443` or matches the Service `targetPort`
- `GatewaySplit.Direct.Authentication` contains the expected direct-auth mode, such as `ntlm`
- container logs include `Starting rdpgw-direct`
- a direct-auth user exists and is enabled
- the dashboard host entry is enabled
- the helper socket path is writable in `/tmp/rdpgw-auth/` for container deployments or in the configured runtime directory for non-container deployments

### Dashboard works but RDP download/connect fails

Check:

- `Caps.TokenAuth: true`
- OIDC is included in `Server.Authentication` for the web download path
- target hosts are server-side configured and enabled
- client certificate/TLS requirements are satisfied
- `Security.VerifyClientIp` is compatible with your proxy's forwarded client IP behavior

### Clipboard, drive, printer, or device redirection is missing

Check:

- `Caps.EnableClipboard`, `Caps.EnableDrive`, `Caps.EnablePrinter`, `Caps.EnablePort`, and `Caps.EnablePnp`
- whether `Caps.DisableRedirect` or `Caps.RedirectAll` is also set
- generated `.rdp` files or gateway capability negotiation for the expected redirect flags

## Related documents

- [README](../README.md)
- [OpenID authentication](./openid-authentication.md)
- [NTLM authentication](./ntlm-authentication.md)
