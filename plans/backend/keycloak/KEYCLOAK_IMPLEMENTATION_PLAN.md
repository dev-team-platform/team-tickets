# Keycloak BFF Implementation Plan

## Goal

Implement Keycloak as the identity provider for Team Tickets with a backend-for-frontend (BFF) design. The Angular application must never receive or store Keycloak access, refresh, or ID tokens. The Go backend owns the OAuth and OpenID Connect protocol work, stores tokens in a server-side session, and exposes only application-specific authentication endpoints.

Registration, password reset, and password change use the Keycloak Admin API. For registration, the backend creates the Keycloak user first and creates the application user only after Keycloak returns the identity subject. Login uses the normal authorization-code BFF flow.

## Target Design

- Use two confidential Keycloak clients:
  - The BFF client supports Authorization Code Flow with PKCE and handles browser login and logout.
  - The administration client has a service account with only the realm-management permissions needed to create users, set credentials, send password actions, and invalidate user sessions.
- Cache an administration-client service token only on the backend for a short bounded lifetime, and redact credentials, passwords, and all token values from logs.
- Keep Keycloak self-registration disabled. Registration is performed only through the Team Tickets BFF so a Keycloak identity and an application user are created as one managed business operation.
- Keep tokens only in Redis-backed, encrypted server sessions. Maintain a TTL-bound local-user-to-session index so password changes and administrative revocation can invalidate every affected BFF session. The browser receives an opaque session identifier in an HttpOnly cookie and a separate readable CSRF cookie.
- Treat KEYCLOAK_PUBLIC_URL as the browser-facing issuer origin and KEYCLOAK_BASE_URL as the backend-reachable Keycloak origin. Validate that issued tokens use the public issuer and never accept an issuer supplied by a request.
- Use the existing users.identity_subject column as the immutable link to Keycloak sub. Keycloak owns credentials; Team Tickets must not retain local passwords.
- Keep authorization limited to authentication and a valid local user in this phase. Mapping Keycloak roles to application permissions is explicitly out of scope.

Keycloak should use a confidential client with Standard Flow and PKCE, while browser registration is disabled because the BFF uses the Admin API instead. Keycloak documents the authorization-code browser flow, client authentication, registration controls, and strict redirect-URI requirements in its [Server Administration Guide](https://www.keycloak.org/docs/latest/server_admin/).

## Public Behaviour and Security Rules

- Public browser flow routes are GET /auth/login, GET /auth/login/callback, and GET /auth/logout/callback. These routes are outside /api/v1 because Keycloak is already configured to return to top-level /auth callbacks.
- Public lifecycle routes are POST /api/v1/auth/register and POST /api/v1/auth/password-reset. Registration is a BFF form submission, not a browser redirect: it accepts the required profile values and initial password over HTTPS. Registration and reset are rate-limited public endpoints, and password reset always returns the same success response to prevent email enumeration.
- Authenticated lifecycle routes are GET /api/v1/auth/me, POST /api/v1/auth/password-change, and POST /api/v1/auth/logout. Unsafe routes require the existing CSRF cookie/header pair.
- Registration flow: validate request, local uniqueness, and the configured password policy before calling Keycloak; create the Keycloak user through the administration client, set the Keycloak credential and required email-verification action, then create the application user with the returned Keycloak subject. If local creation fails, perform and record a compensating Keycloak deletion or reconciliation action. Do not create a local user before Keycloak succeeds.
- Password-reset flow: request Keycloak to send an UPDATE_PASSWORD action email. Return a generic result whether or not the email exists.
- Password-change flow: require a valid BFF session, CSRF validation, and recent Keycloak reauthentication. The backend sets the new credential through the Admin API, invalidates Keycloak and BFF sessions, clears cookies, and requires a normal login. Do not enable Direct Access Grants merely to verify a password.
- When a password change needs reauthentication, use a server-created continuation through the normal GET /auth/login authorization-code flow with Keycloak reauthentication parameters. Bind the purpose and recent-auth result to the BFF session server-side; do not accept a client assertion that reauthentication occurred.
- Login flow: generate and store one-time state, nonce, and PKCE verifier; redirect to Keycloak; exchange the callback code on the backend; verify signature, issuer, audience, expiry, and nonce; find an active local user by identity subject; then create the BFF session. A Keycloak user without an application user is denied and sent to a support-safe error path rather than silently provisioned during login.
- Logout flow: use a CSRF-protected local logout request, delete the local session, and start a one-time opaque continuation to Keycloak's end-session endpoint. Do not expose an ID token or a Keycloak token in Angular data, URLs controlled by Angular, or browser storage.
- Session middleware loads the Redis session, refreshes server-held tokens when needed with a short refresh lock, rejects invalid or expired sessions, and places only the application user identity in the request context.
- Cookies are host-only unless a configured domain is explicitly required, Secure in production, HttpOnly for session and OAuth correlation cookies, SameSite=Lax by default, and scoped to the narrowest practical path. CORS allows only configured origins and credentials; wildcard origins are prohibited.

## Configuration, Data, and Integration Contract

- Extend AppConfig rather than creating a parallel configuration path. Add validated Redis and BFF-session settings, retain Keycloak and CSRF settings, and pass the complete AppConfig to dependency construction.
- Retain the existing Keycloak variables for the BFF client and add separate administrator-client credentials. Document and validate the public URL, backend URL, realm, exact login callback URI, exact post-logout callback URI, frontend URL, CORS origins, Redis connection settings, session cookie name, OAuth correlation-cookie name, session lifetime, state lifetime, refresh skew, reauthentication maximum age, and session-encryption key.
- Change environment loading so local development files are optional helpers and injected runtime environment values take precedence over every file value. Do not require an ignored env file inside production containers. Update the example environment file with every non-secret variable.
- Add a version-controlled Keycloak realm-import definition and a deployment runbook. It must configure the two clients, exact redirect and post-logout URIs, SMTP and required email verification, password policy, required claims, PKCE, disabled implicit/direct-grant flows, service-account least privilege, and no committed secrets. Keycloak supports boot-time realm imports from a correctly named realm JSON file; use that approach instead of untracked Admin Console state. See [Keycloak realm import guidance](https://www.keycloak.org/server/importExport).
- Add only the database migration needed to make the existing users identity linkage safe in deployed databases. The current DDL already contains identity_subject, email-verification, activity, and login/logout fields; do not introduce a local password column. Preserve the user's current uncommitted users-feature changes and reconcile them rather than overwriting them.
- Protect application API groups after the BFF middleware is available, leaving health checks and the defined public auth routes open. Existing local password-based user creation is not an authentication route and must be removed, replaced, or restricted consistently with the Keycloak-owned credential model.
- Update Angular, Nginx, and development proxy behaviour so /auth and /api use the same production origin. Align the Nginx backend upstream with the API port and support explicit credentialed development origins only where a proxy is not used.
- Let application dependency construction own Redis connection lifecycle. Add secret-safe structured lifecycle logs with correlation identifiers, Keycloak and Redis health/readiness checks, and a rollback procedure that cleans up BFF sessions safely. Record the current Nginx-to-backend port mismatch in bugs/backend if implementation fixes it, following the repository convention.

## Task Checklist

- [ ] [001_TASK_KEYCLOAK_REALM_AND_RUNTIME_CONFIGURATION.md](001_TASK_KEYCLOAK_REALM_AND_RUNTIME_CONFIGURATION.md) — Keycloak clients, realm import, AppConfig, environment, Redis, and deployment configuration.
- [ ] [002_TASK_ADMIN_API_IDENTITY_LIFECYCLE.md](002_TASK_ADMIN_API_IDENTITY_LIFECYCLE.md) — Admin API registration, reset-password, change-password, compensation, and local-user creation.
- [ ] [003_TASK_BFF_LOGIN_AND_SERVER_SESSION.md](003_TASK_BFF_LOGIN_AND_SERVER_SESSION.md) — Authorization-code login, PKCE, callback validation, Redis sessions, refresh, and logout.
- [ ] [004_TASK_AUTH_CONTEXT_AND_PROTECTED_ROUTES.md](004_TASK_AUTH_CONTEXT_AND_PROTECTED_ROUTES.md) — Local-user lookup, request context, middleware, route protection, and user-feature transition.
- [ ] [005_TASK_ANGULAR_PROXY_CORS_AND_CSRF_INTEGRATION.md](005_TASK_ANGULAR_PROXY_CORS_AND_CSRF_INTEGRATION.md) — Browser behaviour, Angular integration, Nginx, CORS, and CSRF use.
- [ ] [006_TASK_TESTING_SECURITY_AND_RELEASE_VALIDATION.md](006_TASK_TESTING_SECURITY_AND_RELEASE_VALIDATION.md) — Automated tests, integration scenarios, observability, rollout, and acceptance checks.

## Completion Criteria

- A user can register through the BFF, receive a Keycloak identity first, receive an application user second, verify email, and later log in through the normal BFF redirect flow.
- Reset and change password operate through the least-privileged Keycloak Admin API client; neither Angular nor the database stores credentials or Keycloak tokens.
- A valid session is opaque to the browser, protected against CSRF, stored server-side, refreshed safely, and removed on local or Keycloak logout failure.
- Requests without an active local user, with a disabled user, with an invalid callback, or with a failed session refresh are denied without leaking identity-provider details.
- Production uses HTTPS, strict redirect/origin allow-lists, secure cookies, injected secrets, and a Keycloak realm configuration that can be reviewed and reproduced.

## Plan and Task Consistency Checks

These checks validate the planning documents only; they do not indicate that implementation tasks are complete.

- [x] First pass, plan to task: every generic responsibility has exactly one primary task. Configuration is in 001; Admin API lifecycle is in 002; BFF protocol and session work is in 003; local-user context and protection is in 004; browser and proxy controls are in 005; verification and release controls are in 006.
- [x] Second pass, task to plan: every task checklist is limited to a responsibility, route, security rule, configuration contract, or completion criterion defined above. No task introduces Keycloak role-to-permission mapping, browser token storage, Keycloak self-registration, or local password storage.
