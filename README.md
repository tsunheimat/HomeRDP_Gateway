GO Remote Desktop Gateway
=========================

> Fork status: this repository is an independent modified fork of
> [rdpgw](https://github.com/bolkedebruin/rdpgw), maintained by
> [tsunheimat](https://github.com/tsunheimat). It is not affiliated with,
> endorsed by, or maintained by the original rdpgw project.
>
> The final project name is still being evaluated. Until then, documents may
> refer to the project as `PROJECT_NAME` or as this homelab-oriented rdpgw fork.
>
> This fork focuses on homelab and self-hosted use cases: WebUI-assisted OIDC
> selection, RemoteApp tool file uploads, Windows `mstsc` and macOS Remote
> Desktop client workflows, and unified host access control across two gateway
> components. It is not intended to be a full replacement for Microsoft Windows
> Server Remote Desktop Gateway or Microsoft Remote Desktop Services.
>
> See [MODIFICATIONS.md](./MODIFICATIONS.md) and [NOTICE](./NOTICE) for fork
> attribution, modification details, and product positioning context.

![Go](https://github.com/bolkedebruin/rdpgw/workflows/Go/badge.svg)
[![Docker Pulls](https://badgen.net/docker/pulls/bolkedebruin/rdpgw?icon=docker&label=pulls)](https://hub.docker.com/r/bolkedebruin/rdpgw/)
[![Docker Stars](https://badgen.net/docker/stars/bolkedebruin/rdpgw?icon=docker&label=stars)](https://hub.docker.com/r/bolkedebruin/rdpgw/)
[![Docker Image Size](https://badgen.net/docker/size/bolkedebruin/rdpgw?icon=docker&label=image%20size)](https://hub.docker.com/r/bolkedebruin/rdpgw/)


:star: Star us on GitHub — it helps!

RDPGW is an implementation of the [Remote Desktop Gateway protocol](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-tsgu/0007d661-a86d-4e8f-89f7-7f77f8824188).
This allows you to connect with the official Microsoft clients to remote desktops over HTTPS. 
These desktops could be, for example, [XRDP](http://www.xrdp.org) desktops running in containers
on Kubernetes.

# AIM
RDPGW aims to provide a full open source replacement for MS Remote Desktop Gateway, 
including access policies.

# Security requirements

Several security requirements are stipulated by the client that is connecting to it and some are
enforced by the gateway. The client requires that the server's TLS certificate is valid and that
it is signed by a trusted authority. In addition, the common name in the certificate needs to
match the DNS hostname of the gateway. If these requirements are not met the client will refuse
to connect.

The gateway has several security phases. In the authentication phase the client's credentials are
verified. Depending the authentication mechanism used, the client's credentials are verified against
an OpenID Connect provider, Kerberos, a local PAM service, a local database, or extracted from HTTP headers
provided by upstream proxy services.

If OpenID Connect is used the user will
need to connect to a webpage provided by the gateway to authenticate, which in turn will redirect
the user to the OpenID Connect provider. If the authentication is successful the browser will download
a RDP file with temporary credentials that allow the user to connect to the gateway by using a remote
desktop client.

If Kerberos is used the client will need to have a valid ticket granting ticket (TGT). The gateway
will proxy the TGT request to the KDC. Therefore, the gateway needs to be able to connect to the KDC
and a krb5.conf file needs to be provided. The proxy works without the need for an RDP file and thus
the client can connect directly to the gateway.

If local authentication is used the client will need to provide a username and password that is verified
against PAM. This requires, to ensure privilege separation, that ```rdpgw-auth``` is also running and a
valid PAM configuration is provided per typical configuration.

If NTLM authentication is used, the allowed user credentials for the gateway should be configured in the 
configuration file of `rdpgw-auth`.

Finally, RDP hosts that the client wants to connect to are verified against what was provided by / allowed by
the server. Next to that the client's ip address needs to match the one it obtained the gateway token with if
using OpenID Connect. Due to proxies and NAT this is not always possible and thus can be disabled. However, this
is a security risk.

# Configuration
The configuration is done through a YAML file. The configuration file is read from `rdpgw.yaml` by default.
At the bottom of this README is an example configuration file. In these sections you will find the most important
settings.

## Authentication

RDPGW wants to be secure when you set it up from the start. It supports several authentication
mechanisms such as OpenID Connect, Kerberos, PAM, NTLM, and header-based authentication for proxy integration.

Technically, cookies are encrypted and signed on the client side relying
on [Gorilla Sessions](https://www.gorillatoolkit.org/pkg/sessions). PAA tokens (gateway access tokens)
are generated and signed according to the JWT spec by using [jwt-go](https://github.com/dgrijalva/jwt-go)
signed with a 256 bit HMAC. 

### Multi Factor Authentication (MFA)
RDPGW provides multi-factor authentication out of the box with OpenID Connect integration. Thus
you can integrate your remote desktops with Keycloak, Okta, Google, Azure, Apple or Facebook
if you want.

### Mixing authentication mechanisms

It is technically possible to mix authentication mechanisms. Currently, you can mix local with Kerberos or NTLM. If you enable 
OpenID Connect it is not possible to mix it with local or Kerberos at the moment.

### OpenID Connect

For detailed OpenID Connect setup with providers like Keycloak, Azure AD, Google, and others, see the [OpenID Connect Authentication Documentation](docs/openid-authentication.md).

### Homelab Dashboard (OpenID)

With OpenID Connect enabled, `/` serves a dashboard UI after OIDC login. Dashboard entries are filtered by the groups extracted from your OIDC token (claim configured by `OpenId.GroupsClaim`), so users only see entries where at least one of their groups matches the entry `AllowedGroups`.

Dashboard group checks are evaluated for OIDC web sessions only. For native RDP clients using direct authentication (`local`, `ntlm`, or `kerberos`), enabled dashboard host entry addresses are used as the host allowlist; entry `AllowedGroups` is not evaluated because those authentication modes do not provide reliable group claims to the gateway.

`Dashboard.AdminGroups` controls access to `/admin` and the admin API endpoints. Users in those groups can:

- Create host entries (host/port-backed connections).
- Upload `.rdp` template files and create template-backed entries.
- Update and delete existing entries.
- Upload, select, and delete the global web page icon used by browser tabs and the page header.

The dashboard catalog is stored on local disk (`Dashboard.StorePath`), uploaded templates are written to `Dashboard.UploadDir`, and uploaded web icons are written to `Dashboard.IconDir`. Use `Dashboard.MaxTemplateUploads`, `Dashboard.MaxIconUploads`, `Dashboard.MaxTemplateUploadStorageMb`, and `Dashboard.MaxIconUploadStorageMb` to bound uploaded file count and total storage. The current storage model is intended for single-node/homelab deployments unless you provide shared storage and routing affinity externally.

### Kerberos

For detailed Kerberos setup including keytab generation, DNS requirements, and KDC proxy configuration, see the [Kerberos Authentication Documentation](docs/kerberos-authentication.md).


### PAM/Local Authentication

For detailed PAM setup including LDAP integration, container deployment, and compatible clients, see the [PAM Authentication Documentation](docs/pam-authentication.md).

### NTLM Authentication

For detailed NTLM setup including user management, security considerations, and deployment options, see the [NTLM Authentication Documentation](docs/ntlm-authentication.md).

### Header Authentication (Proxy Integration)

RDPGW supports header-based authentication for integration with reverse proxy services (Azure App Proxy, Google IAP, AWS ALB, etc.) that handle authentication upstream and pass user identity via HTTP headers.

For detailed configuration and examples, see the [Header Authentication Documentation](docs/header-authentication.md).

## TLS

The gateway requires a valid TLS certificate. This means a certificate that is signed by a valid CA that is in the store 
of your clients. If this is not the case particularly Windows clients will fail to connect. You can either provide a 
certificate and key file or let the gateway obtain a certificate from letsencrypt. If you want to use letsencrypt make 
sure that the host is reachable on port 80 from the letsencrypt servers.

For letsencrypt:

```yaml
Tls: auto
```

for your own certificate:
```yaml
Tls: enable
CertFile: server.pem 
KeyFile: key.pem
```

__NOTE__: You can disable TLS on the gateway, but you will then need to make sure a proxy is run in front of it that does
TLS termination. 

`SSLKEYLOGFILE` writes TLS session keys that can decrypt captured gateway traffic. RDPGW fails closed if this environment
variable is set unless `Server.AllowTLSKeyLog: true` is configured. Enable it only for short-lived debugging and remove
the key log file afterwards.


## Example configuration file for Open ID Connect

```yaml
# web server configuration. 
Server:
 # can be set to openid, kerberos, local and ntlm. If openid is used rdpgw expects
 # a configured openid provider, make sure to set caps.tokenauth to true. If local
 # auth is enabled, rdpgw connects to rdpgw-auth over a socket to verify users and
 # passwords. The Docker image runs rdpgw-auth as the same non-root user as rdpgw
 # and does not set the setuid bit. If a future PAM deployment requires a privileged
 # helper, keep that helper isolated behind the restricted socket permissions below.
 # If kerberos is used a keytab and krb5conf need to be supplied. local can be
 # kerberos or ntlm authentication, so that the clients selects what it wants.
 Authentication:
  # - kerberos
  # - local
  - openid
  # - ntlm
 # The socket to connect to if using local auth. Ensure rdpgw auth is configured to
 # use the same socket. rdpgw-auth creates the socket as 0600 by default; if the
 # auth helper and gateway run as different users, run rdpgw-auth with an explicit
 # shared group, for example: --socket-mode 0660 --socket-group rdpgw, and place
 # the socket in a dedicated directory both users can traverse (for example,
 # /tmp/rdpgw-auth created with --socket-dir-mode 0750). Do not grant world
 # access to the auth socket.
 # AuthSocket: /tmp/rdpgw-auth/rdpgw-auth.sock
 # Basic auth timeout (in seconds). Useful if you're planning on waiting for MFA
 BasicAuthTimeout: 5
 # The default option 'auto' uses a certificate file if provided and found otherwise
 # it uses letsencrypt to obtain a certificate, the latter requires that the host is reachable
 # from letsencrypt servers. If TLS termination happens somewhere else (e.g. a load balancer)
 # set this option to 'disable'. This is mutually exclusive with 'authentication: local'
 # Note: rdp connections over a gateway require TLS
 Tls: auto
 # Allows SSLKEYLOGFILE to write TLS session keys for debugging. Keep this false in production.
 AllowTLSKeyLog: false
 # gateway address advertised in the rdp files and browser
 GatewayAddress: localhost
 # port to listen on (change to 80 or equivalent if not using TLS)
 Port: 443
 # list of acceptable desktop hosts to connect to
 Hosts:
  - localhost:3389
  - my-{{ preferred_username }}-host:3389
 # if true the server randomly selects a host to connect to
 # valid options are: 
 #  - roundrobin, which selects a random host from the list (default)
 #  - signed, a listed host specified in the signed query parameter
 #  - unsigned, a listed host specified in the query parameter
 #  - any, insecurely allow any host specified in the query parameter
 HostSelection: roundrobin 
 # a random strings of at least 32 characters to secure cookies on the client
 # make sure to share this across the different pods
 SessionKey: thisisasessionkeyreplacethisjetzt
 SessionEncryptionKey: thisisasessionkeyreplacethisnunu!
  # where to store session details. This can be either file or cookie (default: cookie)
  # if a file store is chosen, it is required to have clients 'keep state' to the rdpgw
  # instance they are connected to.
 SessionStore: cookie
  # tries to set the receive / send buffer of the connections to the client
 # in case of high latency high bandwidth the defaults set by the OS might
 # be to low for a good experience
 # ReceiveBuf: 12582912
 # SendBuf: 12582912 
# Open ID Connect specific settings
OpenId:
 ProviderUrl: http://keycloak/auth/realms/test
 ClientId: rdpgw
 ClientSecret: your-secret
# Kerberos:
#  Keytab: /etc/keytabs/rdpgw.keytab
#  Krb5conf: /etc/krb5.conf
#  enabled / disabled capabilities
Caps:
 SmartCardAuth: false
 # required for openid connect
 TokenAuth: true
 # connection timeout in minutes, 0 is limitless
 IdleTimeout: 10
 EnablePrinter: true
 EnablePort: true
 EnablePnp: true
 EnableDrive: true
 EnableClipboard: true
Client:
  # template rdp file to use for clients
  # rdp file settings and their defaults see here: 
  # https://docs.microsoft.com/en-us/windows-server/remote/remote-desktop-services/clients/rdp-files
  defaults: /etc/rdpgw/default.rdp
  # this is a go string templated with {{ username }} and {{ token }}
  # the example below uses the ASCII field separator to distinguish
  # between user and token 
  UsernameTemplate: "{{ username }}@bla.com\x1f{{ token }}"
  # If true puts splits "user@domain.com" into the user and domain component so that
  # domain gets set in the rdp file and the domain name is stripped from the username
  SplitUserDomain: false
  # If true, removes "username" (and "domain" if SplitUserDomain is true) from RDP file.
  # NoUsername: true
  # If both SigningCert and SigningKey are set the downloaded RDP file will be signed
  # so the client can authenticate the validity of the RDP file and reduce warnings from
  # the client if the CA that issued the certificate is trusted. Both should be PEM encoded
  # and the key must be an unencrypted RSA private key.
  # SigningCert: /path/to/signing.crt
  # SigningKey: /path/to/signing.key
Security:
  # a random string of 32 characters to secure cookies on the client
  # make sure to share this amongst different pods
  PAATokenSigningKey: thisisasessionkeyreplacethisjetzt
  # PAATokenEncryptionKey: thisisasessionkeyreplacethisjetzt
  # a random string of 32 characters to secure cookies on the client
  UserTokenEncryptionKey: thisisasessionkeyreplacethisjetzt
  # Signing makes the token bigger and we are limited to 511 characters
  # UserTokenSigningKey: thisisasessionkeyreplacethisjetzt
  # if you want to enable token generation for the user
  # if true the username will be set to a jwt with the username embedded into it
  EnableUserToken: true
  # Verifies if the ip used to connect to download the rdp file equals from where the
  # connection is opened.
  VerifyClientIp: true
```

## How to build & install

__NOTE__: a [docker image](https://hub.docker.com/r/bolkedebruin/rdpgw/) is available on docker hub, which removes the need for building and installing go.

Ensure that you have `make` (comes with standard build tools, like `build-essential` on Debian), `go` (version 1.25.9 or another Go 1.25.x patch level compatible with the module's `go.mod`), and development files for PAM (`libpam0g-dev` on Debian) installed.

Then clone the repo and issues the following.

```bash
cd rdpgw
make
make install
```

## Testing locally
A convenience docker-compose allows you to test the split OIDC plus direct-auth gateway locally on ports 8443 and 9443. The checked-in compose file starts the gateway container only; point it at your own OpenID Connect provider and managed RDP hosts before use. You will need to allow your browser
to connect to localhost with and self signed security certificate. For chrome set `chrome://flags/#allow-insecure-localhost`.

__NOTE__: The local testing environment uses a self signed certificate. This works for MAC clients, but not for Windows.
If you want to test it on Windows you will need to provide a valid certificate.

```bash
cd dev/docker
docker compose -f docker-compose.yml up --build
```

The Docker image is hardened to run as the non-root `rdpgw` user (UID/GID 1001) by default. The bundled rdpgw-auth helper runs as the same non-root user and the image does not set the setuid bit on the helper. The local compose file mirrors that runtime model with `user: "1001:1001"`, `no-new-privileges`, dropped Linux capabilities, a read-only root filesystem, and a writable `/tmp` tmpfs where the helper creates the auth socket under `/tmp/rdpgw-auth/`. Keep `./data/dashboard` writable by UID 1001 because dashboard state and generated direct-auth helper configuration are stored there.

For Kubernetes deployments, use an equivalent pod/container security context where it is compatible with your auth mode:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1001
  runAsGroup: 1001
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true
```

Mount writable volumes only where needed, for example a `/tmp` tmpfs that allows the helper to create `/tmp/rdpgw-auth/rdpgw-auth.sock` and `/var/lib/rdpgw/dashboard` for dashboard state. If you deploy behind a TLS-terminating reverse proxy, keep the rdpgw backend service private to the proxy or ingress path and do not expose the backend listener directly to untrusted networks.

### Managed Direct Auth Docker Deployment (TLS-ready)

The managed direct-auth sample runs everything inside a single container with two listeners:

- OIDC listener on port `8443`
- direct-auth listener on port `9443`

The container starts two `rdpgw` processes plus the bundled `rdpgw-auth` helper. The OIDC listener owns the browser routes, `/admin`, and dashboard-generated `.rdp` files. The direct listener owns native `ntlm` and `local` RDP gateway traffic. Both listeners share the same dashboard-managed host inventory and direct-auth user list.

1. Set the OpenID values in [`dev/docker/rdpgw.yaml`](dev/docker/rdpgw.yaml), then make sure the dashboard state directory is writable by the container.
2. From the repository root run:

```bash
docker compose -f dev/docker/docker-compose.yml up --build
```

3. Watch the `rdpgw` logs for: `Starting rdpgw-auth (socket: /tmp/rdpgw-auth/rdpgw-auth.sock)` and `Split gateway mode enabled`, then verify both HTTPS listeners are up:

```bash
curl -k https://localhost:8443/
curl -k https://localhost:9443/
```

4. Sign into `https://localhost:8443/admin`, create at least one enabled host entry, and create the direct-auth user you want to test.

5. Connect with an RDP client that supports NTLM or basic auth. Example using FreeRDP against the managed direct-auth inventory:

```bash
xfreerdp /g:localhost:9443 /gd:"" /u:<direct-auth-user> /p:<direct-auth-password> /v:<enabled-dashboard-host> /cert-ignore
```

Security reminders:

- Direct-auth passwords are stored in plain text inside the managed dashboard state so the helper can serve NTLM and local auth. Treat the dashboard store carefully and rotate credentials frequently.
- If you later disable TLS for a reverse proxy deployment, never expose the plain HTTP listener directly to untrusted networks; terminate HTTPS in the proxy and forward HTTP only on a trusted network.
- Replace the sample session/signing keys in [`dev/docker/rdpgw.yaml`](dev/docker/rdpgw.yaml) before production use.
- The gateway issues an HTTP-only cookie named `rdpgw-ntlm-session` to keep NTLM handshakes consistent; make sure any reverse proxy preserves cookies and allows subsequent requests to reach the same backend instance during authentication.

Bootstrap configuration still lives in `rdpgw.yaml` or environment variables: listener ports, external hostnames, TLS, OpenID provider settings, and admin groups. Operational configuration is dashboard-managed: allowed hosts, uploaded templates, and direct-auth users for `local` and `ntlm`.

## Use
Point your browser to the OIDC hostname, `https://your-oidc-gateway/`, for the dashboard or `https://your-oidc-gateway/admin` for administration. Direct `local` and `ntlm` authentication is intended for native RDP clients and should target the direct-auth hostname, `https://your-direct-gateway/`, while still using the managed users and host inventory created from the web UI.

## Integration
The gateway exposes an endpoint for the verification of user tokens at
https://yourserver/tokeninfo . The query parameter is 'access_token' so
you can just do a GET to https://yourserver/tokeninfo?access_token=<token> .
It will return 200 OK with the decrypted token.

In this way you can integrate, for example, it with [pam-jwt](https://github.com/bolkedebruin/pam-jwt).

## Client Caveats
The several clients that Microsoft provides come with their own caveats. 
The most important one is that the default client on Windows ``mstsc`` does 
not support basic authentication. This means you need to use either OpenID Connect,
Kerberos or ntlm authentication.

In addition to that, ``mstsc``, when configuring a gateway directly in the client requires
you to either:
 * "save the credentials" for the gateway
 * or specify a (random) domain name in the username field (e.g. ``.\username``) when prompted for the gateway credentials,
 
otherwise the client will not connect at all (it won't send any packages to the gateway) and it will keep on asking for new credentials.

Finally, ``mstsc`` requires a valid certificate on the gateway.

Additionally, ``mstsc`` is more restrictive about SSL cipher suites compared to other RDP clients. When using a reverse proxy like nginx for TLS termination, you may need to configure specific cipher suites that ``mstsc`` supports. A working configuration for nginx ``ssl_ciphers`` is:
```
ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256
```

``mstsc`` also requires server names rather than IP addresses for connections, despite Microsoft's documentation suggesting otherwise. When configuring hosts in the rdpgw configuration, ensure you use hostnames.

Furthermore, the ``mstsc`` client sends the hostname including the port number when establishing connections. To ensure proper host verification, configure your hosts in the rdpgw configuration file with the port numbers included (e.g., ``myserver:3389`` even for the default RDP port 3389).

The Microsoft Remote Desktop Client from the Microsoft Store does not have these issues,
but it requires that the username and password used for authentication are the same for
both the gateway and the RDP host.

The Microsoft Remote Desktop Client for Mac does not have these issues and is the most flexible.
It supports basic authentication, OpenID Connect and Kerberos and can use different credentials

The official Microsoft IOS and Android clients seem also more flexible.

Third party clients like [FreeRDP](https://www.freerdp.com) might also provide more
flexibility.

## TODO
* Improve Web Interface

# Acknowledgements
* This product includes software developed by the Thomson Reuters Global Resources. ([go-ntlm](https://github.com/m7913d/go-ntlm) - BSD-4 License)
