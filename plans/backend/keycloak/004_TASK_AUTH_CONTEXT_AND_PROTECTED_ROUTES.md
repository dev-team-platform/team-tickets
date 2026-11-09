# Task 004: Auth Context and Protected Routes

## Purpose

Connect the BFF session to the existing user feature and protect application routes with an authenticated local-user context.

## Plan Alignment

Implements the users.identity_subject linkage, local-user login checks, middleware, protected-route rules, and legacy local-password transition in the generic plan. Depends on Tasks 002 and 003.

## Specific Checklist

- [ ] Extend user repository and service interfaces with the subject-based lookup and safe login/logout timestamp updates needed by BFF authentication.
- [ ] Make identity subject immutable after registration and enforce the existing unique subject and email constraints in application behaviour.
- [ ] Inspect deployed database versions and add a versioned migration or explicit schema-verification gate so every deployed users table has the required unique identity subject, email-verification, activity, and login/logout columns before BFF rollout.
- [ ] Define a small request-context model containing only the application user identifier, Keycloak subject, and safe identity attributes needed by downstream handlers.
- [ ] Implement session authentication middleware that reads the opaque BFF session, confirms the current local-user state, and stores the context model for downstream handlers.
- [ ] Reject missing, expired, revoked, malformed, inactive, deleted, or unlinked sessions using the application's standard error response shape.
- [ ] Update last-login time only after a successful BFF session is established and update last-logout time for successful local logout processing.
- [ ] Group application API routes behind the session middleware while keeping health checks, registration, reset-password, login, and OAuth callback routes public.
- [ ] Apply CSRF middleware to every unsafe cookie-authenticated endpoint, including password change and local logout, while allowing only the defined safe methods and public callbacks to bypass it.
- [ ] Provide GET /api/v1/auth/me as a token-free way for Angular to obtain the current application identity and session state.
- [ ] Remove, replace, or restrict the current password-based local user-creation behaviour so it cannot act as a competing authentication or registration path.
- [ ] Reconcile this work with the current uncommitted users-feature changes without discarding user-owned edits.
- [ ] Do not add Keycloak role-to-permission mapping in this task; retain that as a separate future authorization feature.

## Done When

- Protected handlers receive a trusted local-user context instead of parsing browser tokens.
- Public and protected route boundaries are explicit, tested, and consistent with CSRF policy.
- A Keycloak identity without a valid linked application user cannot access Team Tickets.
