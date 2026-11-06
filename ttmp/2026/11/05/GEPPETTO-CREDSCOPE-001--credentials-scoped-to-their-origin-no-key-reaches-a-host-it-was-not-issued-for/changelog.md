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
