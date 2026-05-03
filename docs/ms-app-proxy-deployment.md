# Microsoft Azure Application Proxy Deployment Guide

This guide provides step-by-step instructions for deploying RDPGW behind Microsoft Azure Application Proxy with Conditional Access Policy enforcement.

## Architecture Overview

```
Internet → Azure AD (Auth + CAP) → App Proxy → RDPGW (Internal) → RDP Hosts
```

**Authentication Flow:**
- **Web requests** (`/connect`): Azure/App Proxy supplies trusted headers, and RDPGW still requires its own OIDC session before generating the RDP file from configured allow-listed hosts. Browser-supplied targets such as `/connect?host=...` are rejected.
- **RDP protocol** (`/remoteDesktopGateway/`): Passthrough with OIDC-backed token validation

## Prerequisites

- Azure AD Premium P1/P2 (for Conditional Access)
- Azure AD Application Proxy connector installed
- RDPGW deployed internally
- Network connectivity from connector to RDPGW

## Step 1: Azure AD App Registration

```powershell
# Create app registration
$app = New-AzADApplication -DisplayName "RDPGW-AppProxy" `
    -HomePage "https://rdpgw.yourdomain.com" `
    -IdentifierUris "https://rdpgw.yourdomain.com"

# Note the Application ID
Write-Host "Application ID: $($app.ApplicationId)"
```

## Step 2: Configure Application Proxy

### Portal Configuration

1. **Navigate to**: Azure AD → Enterprise Applications → New Application
2. **Select**: On-premises application
3. **Configure**:
   - **Name**: RDPGW
   - **Internal URL**: `http://rdpgw-server:80`
   - **External URL**: `https://rdpgw.yourdomain.com`
   - **Pre-authentication**: Azure Active Directory
   - **Connector Group**: Select appropriate connector

### Advanced Configuration

```json
{
  "application": {
    "name": "RDPGW",
    "internalUrl": "http://rdpgw-server",
    "externalUrl": "https://rdpgw.yourdomain.com",
    "preAuthentication": "aadPreAuthentication",
    "externalAuthenticationType": "aadPreAuthentication",
    "applicationProxyUrlSettings": {
      "externalUrl": "https://rdpgw.yourdomain.com",
      "internalUrl": "http://rdpgw-server",
      "isTranslateHostHeaderEnabled": true,
      "isTranslateLinksInBodyEnabled": false,
      "isOnPremPublishingEnabled": true
    }
  }
}
```

## Step 3: Configure Passthrough for RDP Endpoint

**Critical**: Configure App Proxy to bypass authentication for RDP connections:

### PowerShell Configuration

```powershell
# Get the application
$app = Get-AzADApplication -DisplayName "RDPGW-AppProxy"

# Configure passthrough paths (if available via API)
# Note: This may need to be configured via Support ticket
$passthroughPaths = @("/remoteDesktopGateway/*")
```

### Support Request

If passthrough configuration isn't available in portal:

1. **Open Azure Support Ticket**
2. **Request**: Passthrough configuration for `/remoteDesktopGateway/*` path
3. **Provide**: Application ID and external URL
4. **Reason**: RDP client compatibility requirements

## Step 4: RDPGW Configuration

### Complete Configuration File

```yaml
# rdpgw.yaml
Server:
  Authentication:
    - openid
    - header
  Tls: disable
  GatewayAddress: https://rdpgw.yourdomain.com
  Port: 80
  # App Proxy terminates HTTPS externally; force Secure cookies on browser sessions.
  SecureCookies: true
  # Replace this with the narrow connector/source CIDR that rdpgw sees as RemoteAddr.
  # Header auth will fail closed unless the immediate peer is inside this range.
  TrustedProxyCIDRs:
    - "10.0.0.0/24"
  Hosts:
    - server1.internal.domain:3389
    - server2.internal.domain:3389
    - "{{ preferred_username }}-desktop:3389"  # Dynamic host mapping

Header:
  UserHeader: "X-MS-CLIENT-PRINCIPAL-NAME"
  UserIdHeader: "X-MS-CLIENT-PRINCIPAL-ID"
  EmailHeader: "X-MS-CLIENT-PRINCIPAL-EMAIL"

OpenId:
  # Configure RDPGW as an OIDC client for the same Entra ID/Azure AD tenant.
  # Web RDP downloads and PAA/user-token flows are OIDC-bound; header-only mode
  # disables TokenAuth/EnableUserToken during configuration load.
  ProviderUrl: https://login.microsoftonline.com/{tenant-id}/v2.0
  ClientId: <rdpgw-oidc-client-id>
  ClientSecret: <rdpgw-oidc-client-secret>

Security:
  # Keep true only if App Proxy provides a stable forwarded client IP from a trusted
  # connector/source. If App Proxy NAT makes client IP unstable, disable this explicitly
  # and rely on short token lifetime plus normal session controls.
  VerifyClientIp: false
  PAATokenSigningKey: "your-32-character-signing-key-here"
  PAATokenEncryptionKey: "your-32-character-encryption-key"

Caps:
  TokenAuth: true  # Effective only when Server.Authentication includes openid
  IdleTimeout: 60

Client:
  UsernameTemplate: "{{ username }}\x1f{{ token }}"
```

### Docker Deployment

```yaml
# docker-compose.yml
services:
  rdpgw:
    image: ghcr.io/tsunheimat/homerdp-gateway:latest
    user: "1001:1001"
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    tmpfs:
      - /tmp:rw,noexec,nosuid,nodev,mode=1777
    ports:
      - "80:80"
    volumes:
      - ./rdpgw.yaml:/opt/rdpgw/rdpgw.yaml:ro
    networks:
      - internal

networks:
  internal:
    driver: bridge
```

## Step 5: Conditional Access Policy

### Create CAP for RDPGW

```powershell
# PowerShell example (simplified)
$conditions = @{
    "applications" = @{
        "includeApplications" = @($app.ApplicationId)
    }
    "users" = @{
        "includeGroups" = @("rdp-users-group-id")
    }
    "locations" = @{
        "includeLocations" = @("AllTrusted")
    }
}

$grantControls = @{
    "operator" = "OR"
    "builtInControls" = @("mfa", "compliantDevice")
}
```

### Portal Configuration

1. **Navigate to**: Azure AD → Security → Conditional Access
2. **Create Policy**:
   - **Name**: RDPGW Access Control
   - **Users**: Select appropriate groups
   - **Cloud apps**: Select RDPGW application
   - **Conditions**: Configure as needed (device, location, etc.)
   - **Grant**: Require MFA + Compliant Device
   - **Session**: Configure session lifetime

## Step 6: Testing

### Test Web Authentication

```bash
# Test /connect endpoint
curl -v https://rdpgw.yourdomain.com/connect
# Should establish/require the RDPGW OIDC session before downloading the RDP file
```

### Test RDP Connection

1. **Access web interface**: `https://rdpgw.yourdomain.com/`
2. **Authenticate**: Complete the RDPGW OIDC login backed by Azure AD/Entra ID
3. **Download RDP file**: Should contain token-based credentials
4. **Connect via RDP client**: Should work without additional authentication

### Verify Headers

Check that App Proxy forwards correct headers by testing through the published App Proxy URL after Azure AD authentication. Do not validate header authentication by curling rdpgw directly with identity headers; direct requests should return `401 Unauthorized` unless they originate from the configured `Server.TrustedProxyCIDRs` range.

```bash
# Test through Azure Application Proxy, not directly against rdpgw.
curl -v https://rdpgw.yourdomain.com/connect
```

## Troubleshooting

### Common Issues

1. **RDP Client Won't Connect**:
   - Verify passthrough configuration for `/remoteDesktopGateway/*`
   - Check token generation in downloaded RDP file
   - Ensure `Server.Authentication` includes `openid`; without OIDC, RDPGW disables `TokenAuth`/user tokens
   - Ensure `TokenAuth: true` remains configured for the OIDC-backed web download path

2. **Authentication Loop**:
   - Verify header configuration matches App Proxy headers
   - Confirm `Server.TrustedProxyCIDRs` contains the App Proxy connector/source CIDR that rdpgw sees as `RemoteAddr`
   - If `Security.VerifyClientIp` is true, confirm App Proxy provides a stable trusted forwarded client IP; otherwise disable it explicitly for App Proxy NAT
   - Validate App Proxy connector connectivity

3. **CAP Not Enforced**:
   - Verify policy targets correct application
   - Check user/group assignments
   - Review conditional access logs

### Debug Commands

```bash
# Check RDPGW logs
docker logs rdpgw-container

# Test internal connectivity without spoofed identity headers; a direct request should
# return 401 unless it comes from a configured trusted proxy CIDR.
curl -v http://rdpgw-internal/connect

# Verify OIDC-backed token generation
curl -v https://rdpgw.yourdomain.com/connect
```

### Azure AD Logs

Monitor these logs for authentication issues:

- **Sign-ins**: User authentication events
- **Conditional Access**: Policy evaluation results
- **Application Proxy**: Connector and application events

## Security Considerations

- **Network Isolation**: Deploy RDPGW in private network and allow inbound access only from the App Proxy connector/source CIDR configured in `Server.TrustedProxyCIDRs`
- **Connector Security**: Ensure App Proxy connector is secured and strips or overwrites identity and forwarding headers before reaching rdpgw
- **Secure Cookies**: Set `Server.SecureCookies: true` when App Proxy terminates HTTPS externally
- **Token Validation**: Monitor for token replay attacks; keep `Security.VerifyClientIp` enabled only when forwarded client IPs are stable and trusted
- **Audit Logging**: Enable comprehensive logging for compliance
- **Certificate Management**: Ensure proper TLS certificate chain
