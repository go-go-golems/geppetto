# turns_codegen.yaml

Source-of-truth manifest for generating turns helper code:

- BlockKind enum string/YAML mappings.
- Geppetto-owned Turn/Data/Block key constants and typed key vars.

Notes:

- Keep exported symbols stable (for example `BlockKindUser`, `KeyTurnMetaSessionID`).
- `ToolConfigValueKey` is generated as a value constant only; its typed key is owned by `pkg/inference/engine` to avoid import cycles.
