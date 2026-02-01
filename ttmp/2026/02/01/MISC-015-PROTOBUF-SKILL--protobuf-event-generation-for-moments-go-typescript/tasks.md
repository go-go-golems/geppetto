# Tasks

## TODO

- [x] Add tasks here

- [x] Inspect `2026-02-01--test-protobuf-ts-go-skill` and decide the canonical layout (`proto/`, `go/`, `web/`, `scripts/`), including where generated Go + TS outputs will live.
- [x] Define a minimal proto schema that mirrors go-go-mento strict exchange conventions:
  - include `schema_version` (uint32) and a stable package name
  - include a nested message for `data` payload
  - include `int64` (to validate bigint/string handling)
  - include `google.protobuf.Struct` (to validate open object mapping)
  - include a `map<string,string>` or `repeated` field (to validate collections)
  - document the expected protojson JSON mapping (camelCase)
- [x] Add `buf.yaml` + `buf.gen.yaml` for Go + TS generation (bufbuild/es + protocolbuffers/go), plus a simple script/Makefile target to run `buf generate`.
- [x] Generate Go + TS artifacts and commit them into the test repo locations that mirror the reference doc conventions.
- [x] Implement a Go emitter that:
  - constructs the protobuf message with `schema_version`
  - renders it to JSON using `protojson` (camelCase)
  - optionally converts to `map[string]any` (protoToMap-style)
  - wraps it in a SEM frame `{sem:true, event:{type,id,data}}`
  - writes the JSON to a file for the TS consumer
- [x] Implement a TS consumer that:
  - loads the JSON output
  - calls `fromJson(<Schema>, event.data)`
  - verifies `schemaVersion`, nested fields, `int64` value types (bigint/string), and `Struct` contents
  - prints a structured success/failure report
- [x] (Optional, if time) Add a minimal Go schema-dump tool that reflects protobuf descriptors to JSON schema to validate camelCase naming and scalar mapping rules.
- [x] Run the end-to-end flow (buf generate -> go emitter -> ts consumer), capture outputs, and record validation results in the validation diary.
