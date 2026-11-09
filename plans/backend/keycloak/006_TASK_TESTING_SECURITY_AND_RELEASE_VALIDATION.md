# Task 006: Testing, Security, and Release Validation

## Purpose

Verify the complete Keycloak BFF lifecycle, failure behaviour, deployment configuration, and security boundaries before release.

## Plan Alignment

Validates every completion criterion in the generic plan. Depends on Tasks 001 through 005.

## Specific Checklist

- [ ] Add unit tests for configuration validation, environment precedence, issuer construction, cookie policy, CSRF checks, session encryption, session expiry, refresh locking, and safe error mapping.
- [ ] Add unit tests for Keycloak Admin API request handling, registration ordering, duplicate identities, password-reset enumeration resistance, password-change reauthentication, session invalidation, and compensation or reconciliation behaviour.
- [ ] Add unit tests for OIDC callback validation: state, nonce, PKCE, signature, issuer, audience, expiry, required claims, and replay failures.
- [ ] Add repository and service tests for subject lookup, active/deleted-user denial, identity-link immutability, and login/logout timestamps.
- [ ] Add route and middleware tests for public-route exceptions, protected API access, missing session, expired session, invalid refresh, CSRF failures, and GET /api/v1/auth/me.
- [ ] Add integration coverage using isolated Keycloak, Redis, and PostgreSQL services for registration, email-verification gate, normal BFF login, session refresh, logout, reset-password action, and password-change forced re-login.
- [ ] Verify that browser-visible responses, cookies, local storage, session storage, request logs, and application logs do not contain Keycloak access, refresh, ID, Admin API, or user-password secrets.
- [ ] Verify Nginx forwarding for /auth and /api, CORS credentials with allowed origins, rejected wildcard origins, HTTPS cookie flags, and exact redirect URI handling.
- [ ] Run the backend test suite, Angular tests and build, and a production-like smoke test with injected environment variables rather than local env files.
- [ ] Add structured authentication lifecycle logs and health/readiness checks for Redis and Keycloak dependencies without logging secrets.
- [ ] Prepare a rollback procedure that disables the BFF routes or restores the previous route exposure only after confirming session cleanup and user-data safety.
- [ ] Review the task checklist against the generic plan once more before release and record any intentional deferred authorization work separately.

## Done When

- Automated and integration tests cover successful flows, security failures, and cross-system partial failures.
- Release validation proves that the browser sees only opaque BFF state and that Keycloak and Redis failures fail safely.
- Operators have reproducible configuration, health checks, logs, and a rollback procedure.
