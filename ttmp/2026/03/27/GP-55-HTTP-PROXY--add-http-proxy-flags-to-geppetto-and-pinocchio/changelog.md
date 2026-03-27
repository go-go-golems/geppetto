# Changelog

## 2026-03-27

- Initial workspace created
- Created `GP-55-HTTP-PROXY` ticket, primary design doc, and diary for Geppetto/Pinocchio explicit proxy support analysis.
- Completed evidence-backed architecture review of shared section ownership, Pinocchio bootstrap/runtime resolution, and provider HTTP-client construction seams.
- Wrote an intern-facing design and implementation guide recommending `ai-client` as the proxy section and a shared HTTP-client builder as the transport wiring seam.
- Added a second intern-facing design document explaining hidden base settings, profile overlays, and runtime profile switching with sequence, schema, and data-ownership diagrams.
- Updated the actual Geppetto and Pinocchio help docs to explain base settings, profile overlays, runtime switching, and the web-chat hidden-base caveat.
- Added a third design document analyzing where `ai-client` flags should be exposed in standard Pinocchio commands versus `web-chat`, including the requirement for a parsed-values-aware base path in `web-chat`.
- Implemented a shared Pinocchio parsed-base helper that overlays parsed non-profile values onto an optional hidden base.
- Updated `cmd/web-chat` to expose the shared `ai-client` section on its CLI and merge those parsed values into the preserved base inference settings before runtime composition.
- Added regression tests covering the new parsed-base helper, the `web-chat` CLI surface, and the `web-chat` base-settings merge behavior.
- Added `ai-client` proxy fields (`proxy-url`, `proxy-from-environment`) to the shared Geppetto client settings struct and Glazed schema, with focused settings tests.
- Related the key Geppetto and Pinocchio source files to the design doc, diary, and ticket index for traceable review.
- Ran `docmgr doctor --ticket GP-55-HTTP-PROXY --stale-after 30`; all checks passed.
- Ran `go test ./pkg/doc/...` in `geppetto/` and `pinocchio/`; both doc packages passed.
- Dry-ran and uploaded the bundle `GP-55 HTTP Proxy Design Guide.pdf` to `/ai/2026/03/27/GP-55-HTTP-PROXY` on reMarkable, then verified the remote listing.
- Dry-ran and uploaded the refreshed bundle `GP-55 HTTP Proxy Design Guide and Base Lifecycle.pdf` to `/ai/2026/03/27/GP-55-HTTP-PROXY`, then verified that both PDFs are present remotely.
- Dry-ran and uploaded the refreshed bundle `GP-55 HTTP Proxy Design Guide, Base Lifecycle, and CLI Exposure.pdf` to `/ai/2026/03/27/GP-55-HTTP-PROXY`, then verified that all three PDFs are present remotely.
