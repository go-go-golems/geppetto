# Changelog

## 2026-01-05

- Initial workspace created


## 2026-01-05

Initial analysis: YAML turns decode data/metadata values as any, breaking typed-key reads for structs/slices (e.g. tool_config)

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/steps/ai/openai/engine_openai.go — KeyToolConfig.Get aborts on mismatch
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/key_families.go — Typed key Get fails with type mismatch
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/types.go — Data/Metadata UnmarshalYAML decodes into map[string]any


## 2026-01-05

Prototype: typed key Decode via JSON re-marshal; YAML fixtures can set tool_config and []string values without hard failure

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/key_families.go — Decode() and Get() behavior
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/serde/key_decode_regression_test.go — Fixture-style YAML regression tests


## 2026-01-05

ToolConfig: custom UnmarshalJSON parses duration strings (execution_timeout/backoff_base) so YAML fixtures decode cleanly

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/inference/engine/types.go — ToolConfig/RetryConfig UnmarshalJSON


## 2026-01-05

Step 5: unblock Moments lint/push + fix Gemini metadata (geppetto 6bc67dd, moments c479ee0)

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/steps/ai/gemini/engine_gemini.go — Switch provider/model Turn.Metadata writes to typed keys
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/moments/backend/pkg/inference/middleware/compression/block_compressor.go — Use helper key instead of calling turns.BlockMetaK directly
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/moments/backend/pkg/inference/middleware/compression/compression_keys.go — Define runtime-configurable BlockMetaKey helper in *_keys.go

