# Header Authentication

RDPGW supports header-based authentication for integration with reverse proxy services that handle authentication upstream.

Web RDP downloads and PAA/user-token flows remain OIDC-bound: when `Server.Authentication` does not include `openid`, RDPGW disables `Caps.TokenAuth` and `Security.EnableUserToken` during configuration load. Use header-only mode only for trusted-proxy identity propagation that does not need the web token download path; use `openid` + `header` for the web UI/RDP download design.

## Configuration

```yaml
Server:
  Authentication:
    - openid
    - header
  Tls: disable  # Proxy handles TLS termination
  TrustedProxyCIDRs:
    - "10.42.0.0/16" # Replace with the proxy pod/service CIDR rdpgw sees as RemoteAddr
  SecureCookies: true # Set when external users access rdpgw over HTTPS

Header:
  UserHeader: "X-Forwarded-User"        # Required: Username header
  UserIdHeader: "X-Forwarded-User-Id"   # Optional: User ID header
  EmailHeader: "X-Forwarded-Email"      # Optional: Email header
  DisplayNameHeader: "X-Forwarded-Name" # Optional: Display name header

Caps:
  TokenAuth: true

Security:
  VerifyClientIp: true
```

Header authentication fails closed unless `Server.TrustedProxyCIDRs` is configured. RDPGW only accepts identity headers when the immediate peer `RemoteAddr` is inside one of those CIDRs. `X-Forwarded-For` is also trusted only from those peers; direct clients and malformed forwarded values fall back to the socket `RemoteAddr`.

## Proxy Service Examples

### Microsoft Azure Application Proxy

```yaml
Server:
  Authentication:
    - openid
    - header
  Tls: disable  # App Proxy handles TLS termination
  TrustedProxyCIDRs:
    - "10.0.0.0/24" # Replace with the connector or internal load-balancer source range
  SecureCookies: true

Header:
  UserHeader: "X-MS-CLIENT-PRINCIPAL-NAME"
  UserIdHeader: "X-MS-CLIENT-PRINCIPAL-ID"
  EmailHeader: "X-MS-CLIENT-PRINCIPAL-EMAIL"

Security:
  VerifyClientIp: true

Caps:
  TokenAuth: true  # Effective for OIDC-backed RDP client connections
```

**Azure Configuration:**

1. **Create App Registration** in Azure AD:
   ```bash
   # Note the Application ID for App Proxy configuration
   az ad app create --display-name "RDPGW-AppProxy"
   ```

2. **Configure Application Proxy**:
   - **Internal URL**: `http://rdpgw-internal:80` (or your internal RDPGW address)
   - **External URL**: `https://rdpgw.yourdomain.com`
   - **Pre-authentication**: Azure Active Directory
   - **Pass through**: Enabled for `/remoteDesktopGateway/`

3. **Configure Conditional Access Policies**:
   - Target the RDPGW App Proxy application
   - Set device compliance, location restrictions, MFA requirements
   - Enable session controls as needed

**Important App Proxy Configuration:**

```json
{
  "name": "RDPGW",
  "internalUrl": "http://rdpgw-internal",
  "externalUrl": "https://rdpgw.yourdomain.com",
  "preAuthenticatedApplication": {
    "preAuthenticationType": "AzureActiveDirectory",
    "passthroughPaths": [
      "/remoteDesktopGateway/*"
    ]
  }
}
```

**Authentication Flow:**

1. **Web Authentication** (`/connect` endpoint):
   ```
   User Browser → App Proxy (trusted headers) → RDPGW OIDC session → Downloads RDP file
   ```

2. **RDP Client Connection** (`/remoteDesktopGateway/` endpoint):
   ```
   RDP Client → App Proxy (passthrough) → RDPGW (OIDC-backed token validation) → RDP Host
   ```

**Key Requirements:**
- **OIDC enabled in RDPGW** for `/connect` web downloads and PAA/user-token generation. Header-only mode does not provide the web token download path.
- **Passthrough configuration** for `/remoteDesktopGateway/` path
- **Header authentication** only from configured trusted proxy CIDRs
- **OIDC-backed token auth** for actual RDP connections
- **Do not pass target hosts in the browser URL**: `/connect?host=...` is intentionally rejected. `/connect` chooses from the server-side configured allow list; OIDC/dashboard per-host downloads use `/connect/entries/{id}.rdp`.
- **Keep IP verification enabled** only when App Proxy supplies stable forwarded client IPs from a trusted source; otherwise disable it explicitly for App Proxy NAT

### Google Cloud Identity-Aware Proxy (IAP)

```yaml
Server:
  Authentication:
    - openid
    - header
  Tls: disable  # IAP/load balancer handles TLS termination
  TrustedProxyCIDRs:
    - "10.128.0.0/20" # Replace with the load-balancer/backend proxy source range rdpgw sees
  SecureCookies: true

Header:
  UserHeader: "X-Goog-Authenticated-User-Email"
  UserIdHeader: "X-Goog-Authenticated-User-ID"
  EmailHeader: "X-Goog-Authenticated-User-Email"

Caps:
  TokenAuth: true

Security:
  VerifyClientIp: true
```

**Setup**: Enable IAP on your Cloud Load Balancer pointing to RDPGW. Configure OAuth consent screen and authorized users/groups.

### AWS Application Load Balancer (ALB) with Cognito

```yaml
Server:
  Authentication:
    - openid
    - header
  Tls: disable  # ALB handles TLS termination
  TrustedProxyCIDRs:
    - "10.0.0.0/24" # Replace with the ALB subnet or target-group source range rdpgw sees
  SecureCookies: true

Header:
  UserHeader: "X-Amzn-Oidc-Subject"
  EmailHeader: "X-Amzn-Oidc-Email"
  DisplayNameHeader: "X-Amzn-Oidc-Name"

Caps:
  TokenAuth: true

Security:
  VerifyClientIp: true
```

**Setup**: Configure ALB with Cognito User Pool authentication. Enable OIDC headers forwarding to RDPGW target group.

### Traefik with ForwardAuth

For Traefik + Kubernetes Gateway API, rdpgw should be reachable only from Traefik, not directly from the Internet or other workloads.

```yaml
Server:
  Authentication:
    - openid
    - header
  Tls: disable
  TrustedProxyCIDRs:
    - "10.42.0.0/16" # Replace with the Traefik pod/service CIDR
  SecureCookies: true

Header:
  UserHeader: "X-Forwarded-User"
  EmailHeader: "X-Forwarded-Email"
  DisplayNameHeader: "X-Forwarded-Name"

Caps:
  TokenAuth: true

Security:
  VerifyClientIp: true
```

**Setup**: Use Traefik ForwardAuth middleware with external auth service (e.g., OAuth2 Proxy, Authelia) that sets headers.

**Gateway API checklist:**

- Publish only Traefik/Gateway API hostnames externally; keep the rdpgw Service internal and do not expose it with a public `LoadBalancer`, `NodePort`, or host network binding.
- Add a Kubernetes `NetworkPolicy` that allows inbound rdpgw traffic only from the Traefik namespace or pods and denies direct pod-to-pod access from other workloads.
- Configure Traefik middleware to strip client-supplied identity headers and set fresh values from the authenticated ForwardAuth response.
- Configure Traefik to overwrite forwarding headers such as `X-Forwarded-For`, `X-Forwarded-Host`, and `X-Forwarded-Proto` instead of passing arbitrary client-supplied values through.
- Set `Server.TrustedProxyCIDRs` to the narrow Traefik pod CIDR, service CIDR, or load-balancer source CIDR that rdpgw sees as `RemoteAddr`; do not use `0.0.0.0/0`.
- Set `Server.SecureCookies: true` for HTTPS external deployments where Traefik terminates TLS and rdpgw receives HTTP.
- Keep `Security.VerifyClientIp: true` when Traefik supplies stable forwarded client IPs from a trusted source. Disable it only for proxy products that cannot provide a stable client IP.

### nginx with auth_request

```yaml
Server:
  Authentication:
    - openid
    - header
  Tls: disable  # nginx handles TLS termination
  TrustedProxyCIDRs:
    - "10.0.0.0/24" # Replace with the nginx source range rdpgw sees as RemoteAddr
  SecureCookies: true

Header:
  UserHeader: "X-Auth-User"
  EmailHeader: "X-Auth-Email"

Caps:
  TokenAuth: true

Security:
  VerifyClientIp: true
```

**nginx config**:
```nginx
upstream rdpgw {
    # rdpgw listens on plain HTTP because Server.Tls is disabled above.
    server rdpgw:443;
}

upstream auth-service {
    server auth-service:80;
}

server {
    listen 443 ssl http2;
    server_name your-gateway.example.com;

    # SSL configuration
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # Auth endpoint (internal)
    location /auth {
        internal;
        proxy_pass http://auth-service;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header X-Original-URI $request_uri;
        proxy_set_header X-Original-Method $request_method;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Main location with auth and WebSocket support
    location / {
        # Authentication
        auth_request /auth;
        auth_request_set $user $upstream_http_x_auth_user;
        auth_request_set $email $upstream_http_x_auth_email;

        # Forward user headers to RDPGW
        proxy_set_header X-Auth-User $user;
        proxy_set_header X-Auth-Email $email;

        # WebSocket and HTTP upgrade support
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts for long-lived connections
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;

        # Disable buffering for real-time protocols
        proxy_buffering off;

        proxy_pass http://rdpgw;
    }
}

# WebSocket upgrade mapping
map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}
```

This nginx example overwrites `X-Forwarded-For` with `$remote_addr` instead of appending `$proxy_add_x_forwarded_for`. Do not append client-supplied `X-Forwarded-For` unless rdpgw is configured for a trusted proxy chain where every hop is validated before the header reaches rdpgw.

## Security Considerations

- **Trust Boundary**: RDPGW trusts identity headers only from `Server.TrustedProxyCIDRs`. Header auth startup fails if no trusted proxy CIDRs are configured.
- **Forwarded Client IP**: RDPGW uses `X-Forwarded-For` only when the immediate peer is trusted and the full comma-separated list contains only valid, non-empty IP addresses. Otherwise it uses `RemoteAddr`.
- **Header Validation**: Configure the proxy to strip or overwrite user identity headers from client requests before it forwards to RDPGW.
- **Network Security**: Deploy RDPGW in a private network accessible only via the proxy. In Kubernetes, use `NetworkPolicy` to allow only Traefik to reach the rdpgw Service.
- **TLS and Cookies**: Enable `Server.SecureCookies: true` when users access rdpgw over HTTPS through a TLS-terminating proxy. This marks browser session cookies and NTLM session cookies as `Secure` even when backend `r.TLS` is nil.
- **Backend TLS**: If the backend network is not fully trusted, enable TLS between the proxy and RDPGW as well.

## Validation

Validate header authentication through the real proxy authentication flow. Do not send identity headers manually from an external client; the proxy or ForwardAuth service should strip client-supplied identity headers and inject fresh authenticated values.

```bash
# Expected: redirect/challenge through the configured proxy auth flow and RDPGW OIDC flow, then a generated RDP download after both authenticated sessions are available.
curl -v https://your-proxy/connect
```

For isolated local tests, use a trusted-proxy-only test harness or loopback proxy that overwrites the identity headers before forwarding to rdpgw.

Do not validate header authentication by curling rdpgw directly with identity headers. Direct requests should return `401 Unauthorized` unless they originate from a configured trusted proxy CIDR.
