# Task 002: Keycloak Admin API Identity Lifecycle

## Purpose

Implement the backend-only identity lifecycle that uses the Keycloak Admin API for registration, password reset, and password change. Keycloak is created first during registration, then the application user is created with the Keycloak subject.

## Plan Alignment

Implements the registration, password-reset, password-change, data-linkage, and no-local-password rules in the generic plan. Depends on Task 001.

## Specific Checklist

- [ ] Define an Admin API client interface and a Keycloak-backed implementation that obtains service-account tokens server-side and never exposes them to Angular.
- [ ] Limit service-token caching to a short server-side lifetime and handle Keycloak token failure without logging credentials, access tokens, refresh tokens, or passwords.
- [ ] Define registration input with the required application profile fields, email, and initial password; apply request validation, password-policy prevalidation, rate limiting, and generic error responses. Apply equivalent rate limiting to public password-reset requests.
- [ ] Expose POST /api/v1/auth/register and POST /api/v1/auth/password-reset as public lifecycle contracts, and expose POST /api/v1/auth/password-change only to a valid BFF session with CSRF protection.
- [ ] Check local email and subject availability before calling the Keycloak Admin API, while treating Keycloak as the final identity-uniqueness authority.
- [ ] On registration, create the Keycloak user first with identity data and a non-temporary credential, then configure and trigger Keycloak's required email-verification action.
- [ ] Capture the Keycloak user identifier as the immutable identity subject before attempting application-user creation.
- [ ] Create the application user only after Keycloak succeeds. Persist identity subject, profile data, email-verification state, activity state, and audit timestamps without storing a password.
- [ ] Make duplicate-email and duplicate-subject handling deterministic across Keycloak and PostgreSQL. Return a safe conflict response without revealing unnecessary Keycloak details.
- [ ] Handle the non-transactional Keycloak/PostgreSQL boundary with a durable provisioning-operation or reconciliation record. If application-user creation fails, request compensating Keycloak deletion and retain enough safe state for retry or operator repair.
- [ ] Ensure an incomplete or failed provisioning operation cannot create a usable BFF session.
- [ ] Implement password-reset request by locating the Keycloak identity through the Admin API and issuing Keycloak's UPDATE_PASSWORD action email. Return the same external response when no matching identity exists.
- [ ] When password change requires reauthentication, start a server-bound normal BFF/OIDC login continuation that forces Keycloak reauthentication and records its completion in the session. Do not trust a client claim that reauthentication occurred.
- [ ] Implement password change only for an authenticated, CSRF-validated, recently Keycloak-reauthenticated local user. Send the new password only to the BFF over HTTPS and set the Keycloak credential through the Admin API.
- [ ] Do not enable Direct Access Grants or perform password verification through a resource-owner password flow.
- [ ] After a successful password change, invalidate the user's Keycloak sessions and all corresponding BFF sessions, clear local cookies, and require a new normal BFF login.
- [ ] Map Admin API failures to stable application error codes and structured operational logs with correlation identifiers but no secrets.

## Done When

- Registration always attempts Keycloak identity creation before application-user creation and has an auditable recovery path for partial failure.
- Password reset and password change are Keycloak-owned actions initiated only by the backend Admin API client.
- The user table has no local credential data and no browser response contains an Admin API or Keycloak token.
