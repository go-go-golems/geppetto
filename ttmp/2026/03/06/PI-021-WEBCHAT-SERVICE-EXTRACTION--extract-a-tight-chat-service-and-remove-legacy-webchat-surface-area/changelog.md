# Changelog

## 2026-03-06

- Initial workspace created
- Added the investigation and migration guide, seeded the implementation diary, and replaced the placeholder task list with a concrete cleanup sequence.
- Collapsed the `webchat` `ChatService` wrapper into a zero-cost compatibility alias over `ConversationService` and validated the change with focused tests plus the repository pre-commit suite (`pinocchio` commit `10caa7e`).
