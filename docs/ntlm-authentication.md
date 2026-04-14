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
Caps:
  TokenAuth: false
```

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
./rdpgw-auth -c /var/lib/rdpgw/dashboard/rdpgw-auth.yaml -s /tmp/rdpgw-auth.sock
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
ExecStart=/usr/local/bin/rdpgw-auth -c /var/lib/rdpgw/dashboard/rdpgw-auth.yaml -s /tmp/rdpgw-auth.sock
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
    environment:
      RDPGW_SERVER__AUTHENTICATION: openid ntlm
      RDPGW_DASHBOARD__STOREPATH: /var/lib/rdpgw/dashboard
      RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH: /var/lib/rdpgw/dashboard/rdpgw-auth.yaml
      RDPGW_AUTH_HELPER_CONFIG: /var/lib/rdpgw/dashboard/rdpgw-auth.yaml
    volumes:
      - ./data/dashboard:/var/lib/rdpgw/dashboard
      - auth-socket:/tmp

volumes:
  auth-socket:
```

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
ls -la /tmp/rdpgw-auth.sock

# Monitor authentication logs
journalctl -u rdpgw-auth -f
```

### Log Analysis

Enable debug logging in `rdpgw-auth` for detailed NTLM protocol analysis:

```bash
./rdpgw-auth -c /var/lib/rdpgw/dashboard/rdpgw-auth.yaml -s /tmp/rdpgw-auth.sock -v
```

## Future Enhancements

Planned improvements for NTLM authentication:

- **Database Backend**: Support for SQLite/PostgreSQL user storage
- **Password Hashing**: Secure password storage options
- **Group Support**: Role-based access control
- **Audit Logging**: Enhanced security monitoring
