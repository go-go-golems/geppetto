# Scripts

## `glazed_migration_analyzer.go`

Static migration analyzer that combines:

- `go/ast` scanning for:
  - legacy Glazed imports
  - selector usage from those imports
  - legacy struct tags (`glazed.parameter`, `glazed.layer`, `glazed.default`, `glazed.help`)
  - function signatures still referencing legacy package symbols
- `gopls references` enrichment for signature hotspots

### Run

```bash
go run ./geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/scripts/glazed_migration_analyzer.go \
  -repo-root /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump \
  -modules geppetto,pinocchio \
  -out-json /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/20-ast-gopls-report.json \
  -out-md /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/20-ast-gopls-report.md \
  -include-gopls \
  -max-gopls-calls 60 \
  -gopls-timeout 12s
```

### Notes

- The analyzer runs `gopls` with `GOWORK=off` per module so references can resolve even when the workspace `go.work` version and local gopls workspace loading disagree.
- The script intentionally skips `ttmp/`, `vendor/`, and `node_modules/` directories.
