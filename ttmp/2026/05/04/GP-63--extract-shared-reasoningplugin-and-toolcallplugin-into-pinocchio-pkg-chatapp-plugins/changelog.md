# Changelog

## 2026-05-04

- Initial workspace created


## 2026-05-04

Created ticket and design doc. Identified 3 duplicate implementations (pinocchio reasoningPlugin, pinocchio agent forwarder, coinvault RuntimeDebugFeature). Designed shared ReasoningPlugin and ToolCallPlugin with new proto messages.


## 2026-05-04

Implementation complete. Created ReasoningPlugin and ToolCallPlugin in pkg/chatapp/plugins/. Wired into pinocchio cmd/web-chat and coinvault. Deleted RuntimeDebugFeature. Updated frontend parsing.ts. All builds and tests pass.


## 2026-05-04

Follow-up hardening complete. Added ToolCallPlugin unit tests in pinocchio, improved publish error handling and timeline state preservation, and made coinvault frontend parsing tolerate raw non-JSON tool payload strings.


## 2026-05-04

End-to-end validation complete with wafer-qwen3.5-397b. HTTP snapshot verified accumulated thinking and assistant completion. Browser smoke test initially found missing StructSchema registration for shared reasoning events; fixed in coinvault commit 48c59d4. Re-test confirmed websocket connected, Thoughts panel rendered accumulated reasoning, and final answer completed.

