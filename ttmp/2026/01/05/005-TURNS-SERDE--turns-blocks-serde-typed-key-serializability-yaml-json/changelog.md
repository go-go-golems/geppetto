# Changelog

## 2026-01-05

- Initial workspace created


## 2026-01-05

Initial analysis: YAML turns decode data/metadata values as any, breaking typed-key reads for structs/slices (e.g. tool_config)

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/steps/ai/openai/engine_openai.go — KeyToolConfig.Get aborts on mismatch
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/key_families.go — Typed key Get fails with type mismatch
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/types.go — Data/Metadata UnmarshalYAML decodes into map[string]any

