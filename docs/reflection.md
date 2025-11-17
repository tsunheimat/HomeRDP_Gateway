# Reflections

- 2025-11-17: Introduced NTLM-specific Docker deployment (initially HTTP-only), documented security trade-offs, and validated compose syntax via `docker compose config`.
- 2025-11-17: Re-enabled TLS for the NTLM sample after mstsc failed with error 0x3000008; updated compose/env/docs to keep TLS on by default while explaining how to disable it safely behind a reverse proxy.
- 2025-11-17: Added cookie-based NTLM session tracking (`rdpgw-ntlm-session`) so reverse proxies or HTTP-only deployments no longer break the multi-step NTLM handshake.
- 2025-11-17: Simplified Docker deployment by auto-starting `rdpgw-auth` from the main container when NTLM/local auth is enabled, eliminating the need for a second service.
