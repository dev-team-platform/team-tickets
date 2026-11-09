# Task 003: BFF Login and Server Session

## Purpose

Implement the normal Keycloak authorization-code BFF login flow and the opaque Redis-backed session that keeps all Keycloak tokens on the backend.

## Plan Alignment

Implements the normal-login, callback, session, refresh, logout, cookie, and token-isolation rules in the generic plan. Depends on Task 001 and uses the local-user identity contract from Task 002.

## Specific Checklist

- [ ] Define an OIDC client abstraction using the existing OAuth2 and OIDC libraries, with configured Keycloak endpoints rather than request-supplied endpoints.
- [ ] Implement GET /auth/login to create single-use state, nonce, and PKCE verifier values in a short-lived server-side record and redirect the browser to Keycloak.
- [ ] Set only an opaque, short-lived OAuth correlation cookie before the browser redirect, with HttpOnly, production Secure, SameSite=Lax default, host-only default, and narrow /auth path settings. Do not put PKCE verifier, nonce, state, or tokens in Angular storage.
- [ ] Implement GET /auth/login/callback to validate callback state, exchange the authorization code from the backend, and verify the ID token signature, expected issuer, audience, expiry, and nonce.
- [ ] Reject callbacks with replayed or missing state, invalid PKCE exchange, invalid issuer or audience, invalid nonce, expired tokens, or missing required claims.
- [ ] Look up the local user by Keycloak subject after token validation. Allow login only for an existing, active, non-deleted application user with the required verified-email policy.
- [ ] Create an opaque, cryptographically random BFF session identifier and store encrypted token material, token expirations, local user identifier, Keycloak subject, and session metadata only in Redis.
- [ ] Maintain a TTL-bound Redis index from local user and Keycloak subject to active BFF session identifiers so password changes and account revocation can invalidate all affected sessions safely.
- [ ] Set the BFF session cookie with HttpOnly, production Secure, SameSite=Lax default, host-only default, narrow practical path, and configured lifetime. Set or rotate a separate readable CSRF cookie that contains only a random anti-forgery value; session and OAuth-correlation cookies remain HttpOnly.
- [ ] Refresh server-held tokens before expiry using the refresh token, a bounded refresh skew, and a Redis lock that prevents concurrent refresh races for one session.
- [ ] Delete the server session and clear cookies when refresh fails, the local user is no longer active, or token validation cannot be recovered safely.
- [ ] Implement CSRF-protected POST /api/v1/auth/logout. Create a one-time opaque logout continuation, remove the BFF session, and send the browser through Keycloak end-session handling without exposing an ID token to Angular.
- [ ] Implement GET /auth/logout/callback to complete local cleanup and redirect only to the configured frontend location.
- [ ] If Keycloak end-session handling times out, fails, or returns an error callback, keep the local BFF session deleted, keep cookies cleared, emit a secret-safe operational event, and send the browser to a safe configured frontend result.
- [ ] Ensure every redirect target is configured or server-validated; do not accept an arbitrary return URL.

## Done When

- A normal browser login results in an opaque application session and no Keycloak token is visible in browser storage, response bodies, or Angular state.
- Invalid callback and session conditions fail closed with a safe application error or configured frontend error path.
- Logout removes the local session and completes Keycloak logout without allowing an open redirect.
