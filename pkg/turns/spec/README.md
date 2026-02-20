# geppetto_codegen.yaml

Source-of-truth manifest for generating turns and JS helper code:

- BlockKind enum string/YAML mappings.
- Geppetto-owned Turn/Data/Block key constants and typed key vars.
- Engine-owned typed turn keys (`pkg/inference/engine/turnkeys_gen.go`).
- Geppetto JS `consts` groups and TypeScript const declarations.

Notes:

- Keep exported symbols stable (for example `BlockKindUser`, `KeyTurnMetaSessionID`).
- `ToolConfigValueKey` is generated as a value constant only; its typed key is generated in `pkg/inference/engine/turnkeys_gen.go` to avoid import cycles.
- Do not hand-edit generated constants in `pkg/turns/keys_gen.go`, `pkg/turns/block_kind_gen.go`, `pkg/inference/engine/turnkeys_gen.go`, `pkg/js/modules/geppetto/consts_gen.go`, `pkg/doc/types/turns.d.ts`, or `pkg/doc/types/geppetto.d.ts`.
- Add/update keys and enums in `pkg/spec/geppetto_codegen.yaml`, then run:
  - `go generate ./pkg/turns`
  - `go generate ./pkg/inference/engine`
  - `go generate ./pkg/js/modules/geppetto`
