# Task 005: Angular, Proxy, CORS, and CSRF Integration

## Purpose

Make the browser use the BFF safely while preserving the rule that Angular never owns Keycloak tokens.

## Plan Alignment

Implements the browser, same-origin production, Nginx, CORS, CSRF, and token-isolation rules in the generic plan. Depends on Tasks 003 and 004.

## Specific Checklist

- [ ] Add an Angular authentication facade that starts login by browser navigation to the BFF route and submits registration as POST /api/v1/auth/register to the BFF. Do not import a Keycloak browser adapter.
- [ ] Use GET /api/v1/auth/me to restore application session state after application load. Treat a 401 response as signed out without attempting browser-side token refresh.
- [ ] Configure Angular HTTP requests to include credentials and send the configured CSRF header only for unsafe same-origin or explicitly allowed credentialed-development BFF requests.
- [ ] Route logout and password-change interactions through the BFF endpoints. Do not place Keycloak endpoints, client secrets, access tokens, refresh tokens, or ID tokens in Angular configuration or storage.
- [ ] Handle BFF login, registration, reset-password, password-change, callback failure, session expiry, and logout outcomes with clear user-facing states that do not reveal identity-provider internals.
- [ ] Update Nginx to proxy both /api and /auth to the backend and align its upstream port with the configured API deployment port.
- [ ] Add an Angular development proxy for /api and /auth where possible so development follows the same-origin BFF model.
- [ ] Configure Gin CORS only for explicit development or deployment origins that cannot use the proxy. Enable credentials only with an exact allow-list and never with a wildcard origin.
- [ ] Confirm cookie domain, Secure, SameSite, and path settings work for the selected HTTPS same-origin production topology.
- [ ] Record the existing Nginx-to-backend port mismatch as a backend deployment defect if it is fixed during implementation, following the repository bug-report convention.

## Done When

- Angular can register, start login, discover session state, change or reset a password, and log out without receiving a Keycloak token.
- Production routes both BFF browser endpoints and API endpoints through one HTTPS origin.
- Cross-origin development behaviour is explicit, credentialed, CSRF-protected, and limited to configured origins.
