# turns_codegen.yaml

Source-of-truth manifest for generating turns helper code:

- BlockKind enum string/YAML mappings.
- Geppetto-owned Turn/Data/Block key constants and typed key vars.

Notes:

- Keep exported symbols stable (for example `BlockKindUser`, `KeyTurnMetaSessionID`).
- `ToolConfigValueKey` is generated as a value constant only; its typed key is owned by `pkg/inference/engine` to avoid import cycles.
- Do not hand-edit geppetto namespace `*ValueKey` constants in `pkg/turns/keys.go` or `pkg/turns/keys_gen.go`.
- Add/update keys in `pkg/turns/spec/turns_codegen.yaml`, then run `go generate ./pkg/turns`.
- JS constants in `gp.consts` import turns-domain groups via `cmd/gen-js-api --turns-schema`; regenerate with `go generate ./pkg/js/modules/geppetto` after turns-key changes.
