# Tasks

## TODO

- [x] Finish the baseline investigation doc and relate the core `pinocchio/pkg/webchat` files to it.
- [ ] Collapse `pkg/webchat/chat_service.go` from a forwarding wrapper into a zero-cost compatibility layer over `ConversationService`.
- [ ] Validate the wrapper-collapse change with focused `pkg/webchat` and `cmd/web-chat` tests.
- [ ] Record the wrapper-collapse step in the diary and changelog, then commit the code change.
- [ ] Remove `Server.NewFromRouter` and update any remaining docs or tests that preserve the old construction seam.
- [ ] Validate the `NewFromRouter` removal with repo-wide grep and targeted tests.
- [ ] Remove or deprecate the alias-only subpackages under `pkg/webchat/{chat,stream,timeline,bootstrap}`.
- [ ] Re-run repo-wide import searches and broad `pinocchio` tests after alias cleanup.
- [ ] Audit `Router.Mount`, `Router.Handle`, `Router.HandleFunc`, and `Router.Handler` for real consumers and decide whether to delete, deprecate, or move them.
- [ ] Update `cmd/web-chat` help text and docs to stop advertising deprecated top-level timeline and turn routes.
- [ ] Write the follow-up extraction note mapping the tightened `pinocchio` backend surface into the broader OS chat-service architecture.
