# NTLM Authentication

RDPGW supports NTLM authentication for direct RDP clients such as the default Windows client `mstsc`. In the homelab dashboard deployment, NTLM and `local` direct auth both read from the same helper-managed user list.

## Advantages

- **Easy Setup**: Simple configuration without external dependencies
- **Windows Client Support**: Works with default Windows client `mstsc`
- **No External Services**: Self-contained authentication mechanism
- **Quick Deployment**: Ideal for small teams or testing environments

## Security Warning

**⚠️ Plain Text Storage**: Passwords are currently stored in plain text to support the NTLM authentication protocol. Keep configuration files secure and avoid reusing passwords for other applications.

## Configuration

### 1. Gateway Configuration

Configure RDPGW to use NTLM authentication:

```yaml
Server:
  Authentication:
    - ntlm
  AuthSocket: /tmp/rdpgw-auth/rdpgw-auth.sock
  SecureCookies: true # Recommended when HTTPS is terminated by a reverse proxy
Caps:
  TokenAuth: false
```

Set `Server.AuthSocket` to the same socket path passed to `rdpgw-auth -s`; otherwise the gateway and helper will listen/connect to different Unix sockets.

### 2. Authentication Helper Configuration

The `rdpgw-auth` helper reads user credentials from the YAML file pointed to by `RDPGW_AUTH_HELPER_CONFIG`.

- In the supported managed deployment, this file is generated automatically from `/admin`.
- `local` and `ntlm` direct auth require OpenID Connect to be enabled for the dashboard management surface.
- Do not treat the helper YAML as an admin-editable source of truth.

```yaml
# /var/lib/rdpgw/dashboard/rdpgw-auth.yaml
Users:
  - Username: "alice"
    Password: "secure_password_1"
  - Username: "bob"
    Password: "secure_password_2"
  - Username: "admin"
    Password: "admin_secure_password"
```

### 3. Start Authentication Helper

Run the `rdpgw-auth` helper with NTLM configuration:

```bash
./rdpgw-auth -c /var/lib/rdpgw/dashboard/rdpgw-auth.yaml -s /tmp/rdpgw-auth/rdpgw-auth.sock
```

## Authentication Flow

1. Client initiates NTLM handshake with gateway
2. Gateway forwards NTLM messages to `rdpgw-auth`
3. Helper validates credentials against configured user database
4. Client connects directly on successful authentication

## User Management

### Dashboard-Managed Mode

When OpenID Connect dashboard mode is enabled:

1. Sign in to `/admin` with an OIDC user in `Dashboard.AdminGroups`
2. Create or update direct-auth users in the "Direct Auth Users" section
3. The server writes `auth-users.json`
4. The server regenerates `RDPGW_AUTH_HELPER_CONFIG`
5. `rdpgw-auth` reloads the config automatically on the next auth request

No manual helper restart is required, and manual helper YAML editing is not part of the supported admin workflow.

## Deployment Options

### Systemd Service

Create `/etc/systemd/system/rdpgw-auth.service`:

```ini
[Unit]
Description=RDPGW NTLM Authentication Helper
After=network.target

[Service]
Type=simple
User=rdpgw
ExecStart=/usr/local/bin/rdpgw-auth -c /var/lib/rdpgw/dashboard/rdpgw-auth.yaml -s /tmp/rdpgw-auth/rdpgw-auth.sock
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Docker Deployment

```yaml
# docker-compose.yml
services:
  rdpgw:
    image: rdpgw
    user: "1001:1001"
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    tmpfs:
      - /tmp:rw,noexec,nosuid,nodev,mode=1777
    ports:
      - "8443:8443"
      - "9443:9443"
    volumes:
      - ./rdpgw.yaml:/opt/rdpgw/rdpgw.yaml:ro
      - ./data/dashboard:/var/lib/rdpgw/dashboard:rw
```

The checked-in `dev/docker/docker-compose.yml` uses this hardened runtime model. The helper socket is created below `/tmp/rdpgw-auth/` rather than directly under `/tmp`, so the socket can keep a private parent directory even when `/tmp` is the writable tmpfs.

Use the split gateway topology in one container:

- OIDC web UI, `/admin`, callback handling, and downloaded `.rdp` clients on the OIDC listener
- NTLM or `local` direct RDP clients on the direct listener
- one shared dashboard store and one shared `rdpgw-auth` helper

In the supported topology, `/admin` on the OIDC listener manages direct-auth users and enabled hosts, then the bundled `rdpgw-auth` helper consumes the generated `rdpgw-auth.yaml` from the shared dashboard store. Direct-auth clients connect only to the direct listener hostname.

## Client Configuration

### Windows (mstsc)

NTLM authentication works seamlessly with the default Windows Remote Desktop client:

1. Configure gateway address in RDP settings
2. Save gateway credentials when prompted
3. Connect using domain credentials or local accounts

### Alternative Clients

NTLM is widely supported across RDP clients:

- **mRemoteNG** (Windows)
- **Royal TS/TSX** (Windows/macOS)
- **Remmina** (Linux)
- **FreeRDP** (Cross-platform)

## Security Best Practices

### File Permissions

Secure the configuration file:

```bash
sudo chown rdpgw:rdpgw /var/lib/rdpgw/dashboard/rdpgw-auth.yaml
sudo chmod 600 /var/lib/rdpgw/dashboard/rdpgw-auth.yaml
```

### Password Policy

- Use strong, unique passwords for each user
- Implement regular password rotation
- Avoid reusing passwords from other systems
- Consider minimum password length requirements

### Network Security

- Deploy gateway behind TLS termination
- Set `Server.SecureCookies: true` when external clients reach rdpgw over HTTPS through a TLS-terminating proxy. This marks the NTLM handshake cookie `Secure` even if backend traffic from the proxy to rdpgw is HTTP.
- Use private networks when possible
- Implement network-level access controls
- Monitor authentication logs for suspicious activity

### Access Control

- Limit user accounts to necessary personnel only
- Regularly audit user list and remove inactive accounts
- Use principle of least privilege
- Consider time-based access restrictions

## Migration Path

For production environments, consider migrating to more secure authentication methods:

### To OpenID Connect
- Better password security (hashed storage)
- MFA support
- Centralized user management
- SSO integration

### To Kerberos
- No password storage in gateway
- Enterprise authentication integration
- Stronger cryptographic security
- Seamless Windows domain integration

## Troubleshooting

### Common Issues

1. **Authentication Failed**: Verify username/password in configuration
2. **Helper Not Running**: Check if `rdpgw-auth` process is active
3. **Socket Errors**: Verify socket path and permissions

### Debug Commands

```bash
# Check helper process
ps aux | grep rdpgw-auth

# Verify configuration
cat /var/lib/rdpgw/dashboard/rdpgw-auth.yaml

# Test socket connectivity
ls -la /tmp/rdpgw-auth/rdpgw-auth.sock

# Monitor authentication logs
journalctl -u rdpgw-auth -f
```

### Log Analysis

Enable debug logging in `rdpgw-auth` for detailed NTLM protocol analysis:

```bash
./rdpgw-auth -c /var/lib/rdpgw/dashboard/rdpgw-auth.yaml -s /tmp/rdpgw-auth/rdpgw-auth.sock -v
```

## Future Enhancements

Planned improvements for NTLM authentication:

- **Database Backend**: Support for SQLite/PostgreSQL user storage
- **Password Hashing**: Secure password storage options
- **Group Support**: Role-based access control
- **Audit Logging**: Enhanced security monitoring
