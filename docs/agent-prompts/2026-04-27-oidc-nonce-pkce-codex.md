# Codex execution brief: harden OIDC with nonce and PKCE S256

## Goal

Implement OIDC nonce validation and PKCE S256 support in HomeRDP_Gateway using strict TDD.

This is a security hardening change for the existing OIDC authorization-code flow. The deployment target uses authentik as the OpenID Provider. authentik supports standard OIDC and PKCE; do not add authentik-specific configuration requirements.

## Baseline

- Repository: HomeRDP_Gateway
- Baseline commit when this brief was written: `427914f fix: harden gateway security defaults`
- Main package area: `cmd/rdpgw/web/oidc.go`
- Existing tests: `cmd/rdpgw/web/oidc_test.go`

## Non-goals

- Do not replace the existing confidential-client `client_secret` support.
- Do not change redirect URI configuration.
- Do not add new user-visible authentik setup requirements unless a test proves the existing assumptions wrong.
- Do not implement implicit or hybrid flow.
- Do not log nonce, code verifier, tokens, or secrets.
- Do not commit the implementation; leave the diff for Hermes to verify and commit.

## Required security behavior

### 1. State record must carry OIDC transaction data

The login transaction must store, at minimum:

- `state`
- safe post-login redirect URL
- `nonce`
- PKCE `code_verifier`

This state should remain short-lived using the existing session expiration behavior.

Avoid fragile delimiter parsing for new fields. Prefer a structured JSON record stored in the session. Preserve compatibility with tests/behavior where practical, but it is acceptable to update internal helpers if all tests pass.

### 2. Authorization redirect must include nonce and PKCE S256

When an unauthenticated user is redirected to the OIDC provider, the authorization URL must include:

- `state=<random>`
- `nonce=<random>`
- `code_challenge=<base64url-no-padding sha256(code_verifier)>`
- `code_challenge_method=S256`

Use cryptographically secure randomness. Do not use `math/rand`.

### 3. Callback must exchange code with PKCE verifier

The token exchange must call the OAuth2 token endpoint with the original `code_verifier`, e.g. via the oauth2 package's PKCE verifier option if available.

The token request must continue to authenticate the confidential client normally with client secret if configured.

### 4. Callback must validate ID token nonce

After verifying the ID token signature/issuer/audience through the existing verifier, validate that the ID token contains the expected nonce for this login transaction.

Reject callback if:

- state is missing/unknown/mismatched
- code verifier is missing
- ID token is missing a nonce claim
- ID token nonce does not match the stored nonce

Return generic authentication errors to the client. Detailed provider or token errors may be logged server-side, but do not log secrets/token values.

### 5. One-time cleanup / replay resistance

After a callback uses a state value, remove the OIDC transaction from the session before or during callback handling so the same state cannot be replayed in a second callback with the same session cookie.

If a later step fails after state cleanup, do not leave the transaction reusable.

## Required TDD steps

Write failing tests first, run them and observe RED, then implement. Add focused tests in `cmd/rdpgw/web/oidc_test.go` or adjacent package tests.

Minimum tests:

1. `Authenticated` redirect includes `nonce`, `code_challenge`, and `code_challenge_method=S256`; stored transaction can be retrieved by `state` and contains nonce/verifier/redirect.
2. PKCE challenge is computed as base64url-no-padding SHA256 of the stored verifier.
3. Callback token exchange sends `code_verifier` to the token endpoint.
4. Callback rejects ID token claims when nonce is missing or mismatched.
5. Callback accepts matching nonce and continues existing identity/session behavior.
6. Callback consumes/removes the stored state so a second call with the same state fails.

Use local httptest servers and local test signing keys/JWTs if needed. Avoid real network calls.

## Existing code hints

- Current transaction storage helpers:
  - `storeOIDCState(w, r, state, redirectURL)`
  - `getOIDCState(r, state)`
- Current login redirect:
  - `h.oAuth2Config.AuthCodeURL(state)`
- Current callback exchange:
  - `h.oAuth2Config.Exchange(ctx, r.URL.Query().Get("code"))`
- Current ID token verification:
  - `h.oidcTokenVerifier.Verify(ctx, rawIDToken)`
- Current identity population uses parsed ID token claims map.

Likely implementation approach:

- Introduce an internal OIDC transaction struct.
- Store a JSON record in the existing session key or a versioned key.
- Add helper(s) to generate random base64url/hex values.
- Add helper(s) to compute PKCE S256 challenge.
- Update `Authenticated` to generate state/nonce/verifier and pass `oauth2.SetAuthURLParam("nonce", nonce)`, `oauth2.S256ChallengeOption(verifier)` or equivalent.
- Update `HandleCallback` to retrieve+consume transaction and pass `oauth2.VerifierOption(verifier)` or equivalent to `Exchange`.
- Validate the nonce claim from the verified ID token before populating identity.

Check the installed version of `golang.org/x/oauth2` for exact PKCE helper names. If helper APIs are unavailable, implement equivalent parameters using `oauth2.SetAuthURLParam("code_challenge", ...)`, `oauth2.SetAuthURLParam("code_challenge_method", "S256")`, and token exchange auth code option for `code_verifier`.

## Required verification commands

Run these before stopping:

```bash
gofmt -w cmd/rdpgw/web/oidc.go cmd/rdpgw/web/oidc_test.go
go test ./cmd/rdpgw/web -run 'TestOIDC'
go test ./cmd/rdpgw/web
go test ./...
go vet ./...
govulncheck ./...
git diff --check
```

If `govulncheck` is not installed, run:

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Documentation

Update OIDC documentation only if needed. If docs are updated, state that authentik usually does not need extra settings for nonce/PKCE S256, and confidential client secret remains in use.

## Blocker behavior

If you cannot implement safely, leave the repository in the smallest clear state and write a concise blocker note in your final Codex output. Do not invent passing results.

## Completion contract

Do not commit. Final output should list:

- tests added
- files changed
- verification commands run and pass/fail status
- any remaining risks or blockers
