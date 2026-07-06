# Tasks

## TODO

- [x] Geppetto: add `AllowHTTP`/`AllowLocalNetworks` to provider settings structs (default false); replace hard-coded `AllowHTTP: false` at Claude (`completion.go:91,:152`), OpenAI Responses (`engine.go:104`, `token_count.go:72`), and OpenAI Chat call sites; unit tests asserting opt-in honored + default denies
- [ ] llm-proxy: expose `allow_http`/`allow_local_networks` profile YAML fields; wire into settings reaching the engine; add a dev profile in `examples/` pointing at `http://127.0.0.1`
- [ ] llm-proxy: implement `stream_options.include_usage` — after stream completion emit a final SSE chunk carrying `usage` (mapped via the same helper as the meter) then `[DONE]`; integration test asserting the final frame matches the ledger row
- [ ] Document the in-process fake-engine test seam (`pkg/byok/integration_test.go`) as the canonical CI pattern in a reference doc
- [ ] Operator smoke: stand up a local plain-HTTP fake provider, drive the full HTTP path with the dev profile, confirm the SSRF opt-in works e2e
- [ ] Verify no geppetto engine fails to populate `result.Usage` for streaming (confirm during include_usage implementation)
