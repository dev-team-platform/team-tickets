# Task 001: Keycloak Realm and Runtime Configuration

## Purpose

Prepare reproducible Keycloak and application configuration for the BFF client, the Admin API client, Redis-backed sessions, and secure deployment. This task establishes the configuration contract used by every later task.

## Plan Alignment

Implements the Target Design and Configuration, Data, and Integration Contract sections of the generic plan. It is the prerequisite for Tasks 002 through 006.

## Specific Checklist

- [ ] Define two distinct confidential Keycloak clients: one for browser BFF login and one service-account client for the Admin API.
- [ ] Configure the BFF client for Standard Authorization Code Flow and PKCE using the S256 method; disable implicit flow and Direct Access Grants.
- [ ] Configure the Admin API service account with only the user-management, user-query, credential-reset, action-email, and user-session-invalidation permissions that its endpoints need. Do not grant realm-admin.
- [ ] Disable Keycloak self-registration and configure the BFF registration route as the only supported registration path.
- [ ] Configure password policy, required email verification, SMTP, and the standard identity claims needed by Team Tickets: subject, email, email-verified state, given name, family name, and display-name attribute if used.
- [ ] Configure exact BFF login and post-logout callback URIs, exact frontend origin, and strict web-origin settings. Do not use wildcard redirect URIs or wildcard web origins.
- [ ] Create a version-controlled realm-import definition with no committed credentials or generated secrets. Add a runbook that explains secure secret injection and import deployment.
- [ ] Extend AppConfig with Redis and BFF-session configuration while retaining the existing Keycloak, CSRF, and CORS configuration groups.
- [ ] Add separate Keycloak administrator-client identifier and secret settings without reusing the BFF client secret.
- [ ] Validate all required settings at startup: URL formats, realm, client identifiers, callback URIs, frontend origin, CORS origins, Redis connectivity settings, cookie names, session timings, encryption-key format, and production HTTPS requirements.
- [ ] Define public versus backend Keycloak URL semantics and verify that the expected token issuer is derived only from the configured public URL and realm.
- [ ] Make local environment files optional development helpers, give injected runtime environment values precedence over all file values, and remove the production dependency on an ignored env file.
- [ ] Update the example environment file with every required non-secret variable and safe explanatory placeholders.
- [ ] Add Redis connection lifecycle ownership to dependency construction and pass the full AppConfig from main into the application dependency graph.

## Done When

- Keycloak configuration can be recreated from tracked artifacts plus injected secrets.
- Startup fails with an actionable, secret-safe error when a BFF, Keycloak, Redis, CSRF, or origin setting is invalid.
- Later tasks can consume one reviewed configuration contract without reading raw environment variables directly.
