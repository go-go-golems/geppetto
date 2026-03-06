# Changelog

## 2026-03-06

- Initial workspace created
- Added the investigation and migration guide, seeded the implementation diary, and replaced the placeholder task list with a concrete cleanup sequence.
- Collapsed the `webchat` `ChatService` wrapper into a zero-cost compatibility alias over `ConversationService` and validated the change with focused tests plus the repository pre-commit suite (`pinocchio` commit `10caa7e`).
- Removed `webchat.NewFromRouter`, confirmed there were no remaining in-repo Go call sites, and revalidated the transport packages (`pinocchio` commit `8221fec`).
- Removed the alias-only `webchat/{chat,stream,timeline,bootstrap}` subpackages after a workspace-wide importer sweep found no live consumers, then revalidated the webchat packages and the full repository commit hook (`pinocchio` commit `51053f0`).
