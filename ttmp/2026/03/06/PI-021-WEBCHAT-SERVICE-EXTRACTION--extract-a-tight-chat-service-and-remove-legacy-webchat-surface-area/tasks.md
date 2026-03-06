# Tasks

## TODO

- [x] Finish the baseline investigation doc and relate the core `pinocchio/pkg/webchat` files to it.
- [x] Collapse `pkg/webchat/chat_service.go` from a forwarding wrapper into a zero-cost compatibility layer over `ConversationService`.
- [x] Validate the wrapper-collapse change with focused `pkg/webchat` and `cmd/web-chat` tests.
- [x] Record the wrapper-collapse step in the diary and changelog, then commit the code change.
- [x] Remove `Server.NewFromRouter` and update any remaining docs or tests that preserve the old construction seam.
- [x] Validate the `NewFromRouter` removal with repo-wide grep and targeted tests.
- [x] Remove or deprecate the alias-only subpackages under `pkg/webchat/{chat,stream,timeline,bootstrap}`.
- [x] Re-run repo-wide import searches and broad `pinocchio` tests after alias cleanup.
- [x] Audit `Router.Mount`, `Router.Handle`, `Router.HandleFunc`, and `Router.Handler` for real consumers and decide whether to delete, deprecate, or move them.
- [x] Update `cmd/web-chat` help text and docs to stop advertising deprecated top-level timeline and turn routes.
- [x] Refactor `web-agent-example` to the modern `glazed` facade packages and handler-first `pinocchio/pkg/webchat` API so it builds again.
- [x] Validate the `web-agent-example` port with `go build ./...` and note any remaining follow-up cleanup such as dead `RegisterMiddleware` usage or SEM package moves.
- [ ] Write the follow-up extraction note mapping the tightened `pinocchio` backend surface into the broader OS chat-service architecture.
