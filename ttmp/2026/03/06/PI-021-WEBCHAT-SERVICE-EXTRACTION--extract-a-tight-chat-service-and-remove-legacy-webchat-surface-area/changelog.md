# Changelog

## 2026-03-06

- Initial workspace created
- Added the investigation and migration guide, seeded the implementation diary, and replaced the placeholder task list with a concrete cleanup sequence.
- Collapsed the `webchat` `ChatService` wrapper into a zero-cost compatibility alias over `ConversationService` and validated the change with focused tests plus the repository pre-commit suite (`pinocchio` commit `10caa7e`).
- Removed `webchat.NewFromRouter`, confirmed there were no remaining in-repo Go call sites, and revalidated the transport packages (`pinocchio` commit `8221fec`).
- Removed the alias-only `webchat/{chat,stream,timeline,bootstrap}` subpackages after a workspace-wide importer sweep found no live consumers, then revalidated the webchat packages and the full repository commit hook (`pinocchio` commit `51053f0`).
- Removed the unused router utility mux API (`Mount`, `Handle`, `HandleFunc`, `Handler`) after confirming it had no live production consumers, then updated package docs to match the stricter handler-first model (`pinocchio` commit `7ab4beb`).
- Updated `cmd/web-chat` flag help text to advertise `/api/timeline` and `/api/debug/turns` instead of the older top-level route names (`pinocchio` commit `a091f2d`).
- Ported `web-agent-example` to the modern `glazed` facade packages plus the handler-first `pinocchio/pkg/webchat` API, replacing `EngineFromReqBuilder`/`AddProfile` with a small request resolver and runtime composer, and updated its custom SEM/timeline code to the current protobuf and `TimelineEntityV2` surfaces (`web-agent-example` commit `bcec871`).
- Removed the dead `webchat` middleware registration surface (`MiddlewareBuilder`, `mwFactories`, `Router.RegisterMiddleware`, `Server.RegisterMiddleware`) after confirming there were no live call sites or read paths left in the workspace (`pinocchio` commit `e1ae805`).
