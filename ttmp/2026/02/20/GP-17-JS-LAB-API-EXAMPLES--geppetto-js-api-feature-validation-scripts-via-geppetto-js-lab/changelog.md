# Changelog

## 2026-02-20

- Initial workspace created
- Added detailed validation guide document mapping new JS API features to concrete geppetto-js-lab checks.
- Added ticket-local validation scripts:
  - `scripts/01_handles_consts_and_turns.js`
  - `scripts/02_context_hooks_and_run_options.js`
  - `scripts/03_async_surface_smoke.js`
  - `scripts/run_all.sh`
- Executed all scripts successfully with `go run ./cmd/examples/geppetto-js-lab --script ...` and `scripts/run_all.sh`.
- Uploaded guide to reMarkable:
  - Primary upload attempt via `remarquee upload md` failed with TLS protocol error.
  - Fallback upload via `python3 /home/manuel/.local/bin/remarkable_upload.py` succeeded.
