# Changelog

## 2026-02-12

- Initial workspace created


## 2026-02-12

Task 1 complete: scaffolded turns generator, schema manifest, and go:generate wiring; validated via smoke generation.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/cmd/gen-turns/main.go — Generator CLI and templates
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/generate.go — go:generate entrypoint
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/spec/turns_codegen.yaml — Schema source of truth


## 2026-02-12

Task 2 complete: adopted generated BlockKind mapper and removed handwritten mapper code from types.go.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/block_kind_gen.go — Generated BlockKind enum/string/YAML mapper
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/generate.go — Updated go:generate directives for partial migration
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/types.go — Removed handwritten block-kind mapper logic


## 2026-02-12

Task 3 complete: adopted generated keys constants and typed key vars in keys_gen.go; removed duplicate handwritten sections.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/generate.go — Generate keys in package root
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/keys.go — Kept only handwritten payload/run constants
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/keys_gen.go — Generated key constants and typed keys


## 2026-02-12

Task 4 complete: added generator unit tests, generation hygiene (.generated ignore), and final validation runs.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/.gitignore — Ignore temporary generated scratch outputs
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/cmd/gen-turns/main_test.go — Generator validation regression tests

