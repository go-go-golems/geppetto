# Changelog

## 2026-02-04

- Initial workspace created


## 2026-02-04

Added design + implementation plan for disco dialogue middleware/widget.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/04/PI-011-DISCO-DIALOGUE-MW--disco-style-internal-dialogue-middleware-widget/analysis/01-disco-internal-dialogue-middleware-widget-design-implementation-plan.md — Initial design and plan.


## 2026-02-04

Updated design to include passive/active checks, anti-passives, and thought cabinet behavior from imported reference.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/04/PI-011-DISCO-DIALOGUE-MW--disco-style-internal-dialogue-middleware-widget/analysis/01-disco-internal-dialogue-middleware-widget-design-implementation-plan.md — Expanded event model and middleware behavior.


## 2026-02-04

Added LLM prompt pack for internal dialogue + dice simulation.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/04/PI-011-DISCO-DIALOGUE-MW--disco-style-internal-dialogue-middleware-widget/analysis/01-disco-internal-dialogue-middleware-widget-design-implementation-plan.md — Prompt templates and schema.


## 2026-02-04

Documented structured sink prompting pipeline and updated plan (commit 0b9ad10).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/04/PI-011-DISCO-DIALOGUE-MW--disco-style-internal-dialogue-middleware-widget/analysis/01-disco-internal-dialogue-middleware-widget-design-implementation-plan.md — Updated plan with structured sink phases and tagged YAML prompts
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/04/PI-011-DISCO-DIALOGUE-MW--disco-style-internal-dialogue-middleware-widget/analysis/02-prompting-structured-sink-pipeline-for-disco-dialogue.md — New analysis document with pipeline details


## 2026-02-04

Step 2: added disco dialogue protobuf schema + generated outputs (commit 254094a).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/sem/pb/proto/sem/middleware/disco_dialogue.pb.go — Generated Go types
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/proto/sem/middleware/disco_dialogue.proto — New schema for disco dialogue events


## 2026-02-04

Step 3: added disco dialogue payload events + structuredsink extractors (commit 68c93ab).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/pkg/discodialogue/events.go — Event types + payload structs
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/pkg/discodialogue/extractor.go — FilteringSink extractors for line/check/state


## 2026-02-04

Step 4: added disco dialogue prompt injection middleware (commit 21b2967).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/pkg/discodialogue/middleware.go — Prompt injection + config parsing


## 2026-02-04

Step 5: added event sink wrapper and wired disco extractors (commits 5f1de53, c87512b).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine_builder.go — Apply event sink wrapper during BuildFromConfig
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router_options.go — New WithEventSinkWrapper option
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/cmd/web-agent-example/sink_wrapper.go — Attach structuredsink extractors


## 2026-02-04

Step 6: added disco SEM mapping and timeline snapshots (commits 2440f7a, ad15ec2).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/proto/sem/timeline/middleware.proto — Disco dialogue timeline snapshots
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/proto/sem/timeline/transport.proto — Timeline oneof additions
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/pkg/discodialogue/sem.go — SEM registry mappings
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/pkg/discodialogue/timeline.go — Timeline projector handlers

