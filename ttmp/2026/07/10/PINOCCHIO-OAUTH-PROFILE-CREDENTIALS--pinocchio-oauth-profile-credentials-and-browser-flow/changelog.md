# Changelog

## 2026-07-10

- Initial workspace created


## 2026-07-10

Created follow-on design ticket for Pinocchio profile YAML access/refresh/expiry persistence, token refresh endpoint support, and PKCE browser login.

## 2026-07-10

Implemented Geppetto OAuth protocol primitives in 9e0d93b4: PKCE, authorization URL, code exchange, forced refresh grant, expiry normalization, rotation policy, and redacted errors; Pinocchio remains profile/browser host.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/credentials/oauth/oauth.go — Reusable OAuth client
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/credentials/oauth/oauth_test.go — Fake OAuth endpoint tests
