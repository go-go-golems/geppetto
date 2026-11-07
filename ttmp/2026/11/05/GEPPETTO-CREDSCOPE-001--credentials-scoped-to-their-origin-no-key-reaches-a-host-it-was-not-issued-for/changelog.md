# Changelog

## 2026-11-05

- Initial workspace created

## 2026-11-05

Ticket opened; inventory and in-process leak probe (6 paths confirmed); intern design guide written

## 2026-11-05

Implemented phases 1-5: Origin/PinnedTransport/PinClient + credentials.Scoped; Pi endpoints by provider with pinned clients; settings-path engines pinned, Responses slot pairing, embeddings honour openai-base-url; 38-row regression matrix (14 fail before, 38 pass after); help topic geppetto-credential-scope

### Related Files

- /home/manuel/workspaces/2026-11-02/rag-workbench/geppetto/pkg/security/origin.go — pinning primitives
- /home/manuel/workspaces/2026-11-02/rag-workbench/geppetto/pkg/steps/ai/credscope/credscope_test.go — regression matrix
- /home/manuel/workspaces/2026-11-02/rag-workbench/geppetto/pkg/steps/ai/providers/pi/endpoint.go — provider-keyed endpoints

## 2026-11-06

Design 02: define immutable bound connections, request-time authentication and atomic credential layer selection; begin implementation

## 2026-11-06

Implement the immutable bound connection and shared authenticated request executor; connection unit tests and race check pass

## 2026-11-06

Migrate inference, token counting, Pi, System One binding and Whisper to shared connections; original matrix and recording SDK/mutation tests pass

## 2026-11-06

Finish embeddings/rerank migration and atomic credential pairs; centralize provider defaults; 64-path matrix and repository compile/build pass

## 2026-11-06

Bind OAuth token grants and Umans issuer authentication; recording tests refuse credential-bearing redirects and protocol-vendor fallback

## 2026-11-06

Finish connection-aware factories and atomic profile overlays; publish usage guide, compiled examples and validation evidence with environment and custom-vet limits

## 2026-11-06 — PR #446 review fixes

- Preserve complete OAuth token request queries while binding a query-free declaration; retain redirect refusal.
- Restore the direct SystemOne nil-client 60-second timeout and preserve injected timeout policy.
- Recording regressions fail before and pass after the fixes, including race checks; full OAuth socket tests remain environment-blocked. Diary step 14 records commands and evidence.

## 2026-11-06 — CI fixture corrections

- Claude inference tests use the public connection-establishing constructor; payload-only tests keep their lightweight fixture.
- llama.cpp policy refusal assertion follows the actual cause instead of the removed wrapper wording.
- Focused constructor/payload tests pass and both packages compile; TLS inference execution awaits CI. See diary step 15.

## 2026-11-06 — Remaining full-suite CI failures

- CI confirms the Claude and llama.cpp repairs and successful lint.
- Complete the JS embedding-local fixture binding and preserve Pi decision invalid-request classification at connection construction.
- Both affected test groups pass locally; see diary step 16 for CI evidence and the execution-policy rejection.

## 2026-11-06

Require connection arguments for inference/counters, centralize source resolution, remove Responses legacy selection, and document the short migration. Focused/race/build/lint checks pass; full socket suites run in PR CI.
