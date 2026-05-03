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
   - Configure redirect/callback URL to match your external gateway URL.
   - Keep the client secret private.
   - Ensure the token contains the group claim you configure as `OpenId.GroupsClaim` if you use group-based dashboard filtering.
3. Decide how TLS is handled.
   - If HomeRDP Gateway handles TLS directly, configure certificate/key paths.
   - If a reverse proxy terminates TLS, keep the backend private and set `Server.SecureCookies: true`.
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

5. Check the listeners:

```bash
curl -k https://localhost:8443/
curl -k https://localhost:9443/
```

6. Open the admin UI:

```text
https://localhost:8443/admin
```

7. After login, create at least one enabled dashboard entry. If you want direct-auth/native RDP testing, also create a direct-auth user in the admin UI.

## Kubernetes quick start

The starter manifest is [`k8s/rdpgw.yaml`](../k8s/rdpgw.yaml). It includes:

- a `ConfigMap` containing `rdpgw.yaml`
- a `Deployment` using `ghcr.io/tsunheimat/homerdp-gateway:latest`
- a `PersistentVolumeClaim` for dashboard state
- a `ClusterIP` `Service` exposing ports `8443` and `9443`

1. Copy the manifest for your environment:

```bash
cp k8s/rdpgw.yaml /tmp/rdpgw-k8s.yaml
```

2. Edit `/tmp/rdpgw-k8s.yaml` and replace all placeholder config values under the `ConfigMap` `rdpgw.yaml` key.

For production, prefer Kubernetes `Secret` objects for sensitive values such as OIDC client secrets and token/session keys. The checked-in ConfigMap keeps everything visible for a single-file starter example only.

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

5. Expose the service through your preferred ingress, gateway API, load balancer, or port-forwarding path. For a quick private test:

```bash
kubectl port-forward service/rdpgw 8443:8443 9443:9443
```

Then open:

```text
https://localhost:8443/admin
```

## Native RDP/direct-auth test

After creating an enabled host entry and a direct-auth user, test with FreeRDP:

```bash
xfreerdp /g:<gateway-host>:9443 /gd:"" /u:<direct-auth-user> /p:<direct-auth-password> /v:<enabled-dashboard-host> /cert-ignore
```

Notes:

- Windows `mstsc` does not support basic authentication for the gateway; use OpenID Connect, Kerberos, or NTLM depending on your deployment.
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

- `Server.GatewayAddress` matches the externally visible URL
- IdP redirect URL matches the gateway callback URL
- reverse proxy forwards the expected scheme/host
- `Server.SecureCookies` is true when users access the gateway over HTTPS but the backend receives HTTP

### Dashboard works but RDP download/connect fails

Check:

- `Caps.TokenAuth: true`
- OIDC is included in `Server.Authentication` for the web download path
- target hosts are server-side configured and enabled
- client certificate/TLS requirements are satisfied

### Direct RDP auth fails

Check:

- the direct listener is reachable on the expected hostname/port
- the direct-auth user exists and is enabled
- the dashboard host entry is enabled
- the helper socket path is writable in the configured runtime directory

## Related documents

- [README](../README.md)
- [OpenID authentication](./openid-authentication.md)
- [NTLM authentication](./ntlm-authentication.md)
- [Header authentication](./header-authentication.md)
- [Microsoft App Proxy deployment](./ms-app-proxy-deployment.md)
- [Kerberos authentication](./kerberos-authentication.md)
