# Tasks

## TODO

- [ ] Add tasks here

- [ ] Define protobuf + SEM event types for disco dialogue (generate Go code)
- [ ] Implement DiscoDialogue middleware (Go) with config + event emission
- [ ] Add timeline projection handler for disco dialogue events
- [ ] Build DiscoDialogueCard widget + SEM registration in frontend
- [ ] Wire middleware + widget into web-agent-example; add demo + docs
- [x] Define disco dialogue protobuf schema in pinocchio/proto/sem/middleware + regenerate pb.go
- [x] Implement disco dialogue structuredsink extractors (dialogue_line/dialogue_check/dialogue_state) + event payloads
- [x] Add disco dialogue middleware prompt injection + config parsing (web-agent-example/pkg/discodialogue)
- [x] Wire FilteringSink extractors into webchat sink pipeline (pinocchio option + web-agent-example integration)
- [ ] Add SEM registry mapping for disco dialogue events + timeline projector handler
- [ ] Implement DiscoDialogueCard widget + SEM frontend registration + styles
- [ ] Wire disco middleware + widget into web-agent-example profile; add demo docs
