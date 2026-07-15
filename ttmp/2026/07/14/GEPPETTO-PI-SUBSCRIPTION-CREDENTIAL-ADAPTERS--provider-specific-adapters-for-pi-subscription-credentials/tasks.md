# Tasks

## TODO

- [x] Capture evidence for Pi, OpenAI Codex, Anthropic, Umans, and relevant Geppetto boundaries <!-- t:sv58 -->
- [x] Write the provider-specific adapter architecture, API contracts, pseudocode, and phased implementation guide <!-- t:6hs1 -->
- [x] Validate ticket metadata, relate evidence files, and commit documentation <!-- t:mbn9 -->
- [x] Bundle the guide and diary for reMarkable delivery <!-- t:7zsw -->
- [x] Revise the design: Geppetto owns reusable provider lifecycle mechanics; hosts own storage binding and interaction policy <!-- t:up03 -->
- [x] Validate and re-upload the revised design bundle to reMarkable <!-- t:xkc2 -->
- [x] Replace the isolated Codex engine proposal with restricted shared-core route and request/response middleware <!-- t:2c83 -->
- [ ] Implement the Go-only restricted transport contracts: RouteResolver, HeaderWriter, request/response middleware, response decisions, ordering, and security tests <!-- t:79md -->
- [ ] Wire the OpenAI Responses core through the transport pipeline while preserving URL-before-credential, cancellation, bounded replay, and existing bearer regression coverage <!-- t:idw2 -->
- [ ] Implement and fake-server test the OpenAI Codex route resolver and typed credential middleware, including account headers and exactly-once eligible 401 refresh <!-- t:4rvt -->
- [ ] Define and implement reusable Geppetto credential lifecycle primitives: Store helpers, PKCE/state flow, refresh rotation, redacted status, and idempotent local logout <!-- t:ml0f -->
- [ ] Audit and add contract-tested Anthropic OAuth and Umans API-key middleware across Messages streaming, non-streaming, and token-count paths <!-- t:acua -->
- [ ] Bind Pinocchio direct-YAML lifecycle storage and CLI UX to Geppetto primitives; separately design explicit consented Pi migration <!-- t:m1dp -->
- [ ] Run focused race/security/quality validation and, only with explicit approval, execute a redacted provider-specific live smoke <!-- t:kd0z -->
