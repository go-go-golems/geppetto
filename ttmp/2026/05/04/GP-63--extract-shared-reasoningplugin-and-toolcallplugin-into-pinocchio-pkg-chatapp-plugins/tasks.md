# Tasks

## TODO

- [x] Add tasks here

- [x] Add ToolCallUpdate, ToolResultUpdate, ToolCallEntity, ToolResultEntity protos to chat.proto and regenerate
- [x] Create pkg/chatapp/plugins/reasoning.go — extract from cmd/web-chat reasoning_chat_feature.go
- [x] Create pkg/chatapp/plugins/toolcall.go — new ToolCallPlugin
- [x] Wire shared plugins into pinocchio cmd/web-chat, delete local reasoning_chat_feature.go
- [x] Wire shared plugins into coinvault, delete RuntimeDebugFeature and custom protos
- [x] Update coinvault frontend JS to handle new event names
- [ ] Verify builds and tests across pinocchio, coinvault, geppetto
