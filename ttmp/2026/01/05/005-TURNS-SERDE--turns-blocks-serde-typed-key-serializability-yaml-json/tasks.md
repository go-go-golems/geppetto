# Tasks

## TODO

- [x] Reproduce YAML decode type-mismatch for `geppetto.tool_config@v1` and `geppetto.agent_mode_allowed_tools@v1`
- [ ] Decide desired YAML/JSON schema for typed Turn/Block data+metadata values (snake_case, durations, enums)
- [x] Choose implementation approach (lenient `Get`, serde post-process, or registry-based codecs)
- [x] Implement chosen approach for `engine.ToolConfig` (and add tests via YAML fixtures)
- [ ] Expand coverage to the rest of canonical typed keys (turn meta keys, block meta keys, lists/maps)
- [x] Update docs: `pkg/doc/topics/10-turn-blocks-serialization.md` with key-value schema guidance
