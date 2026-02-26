# Changelog

## 2026-02-12

- Initial workspace created
- Added JS help entries:
  - `pkg/doc/topics/13-js-api-reference.md`
  - `pkg/doc/topics/14-js-api-user-guide.md`
  - `pkg/doc/tutorials/05-js-api-getting-started.md`
- Updated docs index with JS links in `pkg/doc/topics/00-docs-index.md`
- Added doc help loading test in `pkg/doc/doc_test.go`
- Added validation runner:
  - `geppetto/ttmp/2026/02/12/GP-011-JS-HELP-DOCS--js-api-help-documentation-suite/scripts/test_doc_examples.sh`
- Added script execution host:
  - `cmd/examples/geppetto-js-lab/main.go`
- Added script-first JS example suite:
  - `examples/js/geppetto/01_turns_and_blocks.js`
  - `examples/js/geppetto/02_session_echo.js`
  - `examples/js/geppetto/03_middleware_composition.js`
  - `examples/js/geppetto/04_tools_and_toolloop.js`
  - `examples/js/geppetto/05_go_tools_from_js.js`
  - `examples/js/geppetto/06_live_profile_inference.js`
- Rewrote JS docs to be fully focused on writing and running JS scripts (not Go unit tests)
- Executed all script-first documentation examples via `geppetto-js-lab`
- Expanded `pkg/doc/tutorials/05-js-api-getting-started.md` into a long-form fundamentals guide with:
  - detailed concept explanations per step
  - API deep-dive sections
  - pseudocode blocks
  - ASCII flow diagrams
  - script-first validation checklists

## 2026-02-25

Ticket closed

