# Changelog

## 2026-02-14

- Initial workspace created

## 2026-02-14 - Added extraction design and execution tasks

Documented a detailed architecture and implementation plan to move `BuildConfig`/`BuildFromConfig` out of `pkg/webchat` and into app-owned runtime composer code.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-023--webchat-runtime-builder-extraction-move-buildconfig-buildfromconfig-out-of-pkg-webchat-core/design-doc/01-buildconfig-buildfromconfig-extraction-plan.md — Primary design and migration plan
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-023--webchat-runtime-builder-extraction-move-buildconfig-buildfromconfig-out-of-pkg-webchat-core/tasks.md — Detailed phased task list

## 2026-02-14 - Implementation completed (core + apps + docs)

Completed the GP-023 cutover by removing `BuildConfig`/`BuildFromConfig` from core webchat and migrating runtime policy into app-owned composers in both `cmd/web-chat` and `web-agent-example`.

Core now uses `RuntimeComposer` (`RuntimeComposeRequest` + `RuntimeArtifacts`) and rebuilds conversations on `RuntimeFingerprint` changes. Legacy `EngineConfig` and `EngineBuilder` files were deleted, and stale docs were updated to the new architecture.

### Commits

- `04dc5e6` (`pinocchio`) — webchat: replace BuildConfig with app-owned runtime composer
- `c3bad6c` (`pinocchio`) — webchat docs: document runtime composer architecture
- `855b358` (`pinocchio`) — webchat docs: remove stale engine_config reference
- `8221473` (`web-agent-example`) — web-agent-example: adopt webchat runtime composer contract

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/runtime_composer.go — New core runtime composer contract
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/conversation.go — ConvManager composer integration + runtime fingerprint rebuild logic
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router.go — Router wiring for composer, default sink path, and inference tool selection
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/runtime_composer.go — App-owned runtime policy in pinocchio web-chat
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/runtime_composer.go — App-owned runtime policy in web-agent-example
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/doc/topics/webchat-backend-reference.md — Updated backend docs for resolver/composer architecture
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/doc/tutorials/03-thirdparty-webchat-playbook.md — Updated tutorial snippets to new sink wrapper/composer contract
