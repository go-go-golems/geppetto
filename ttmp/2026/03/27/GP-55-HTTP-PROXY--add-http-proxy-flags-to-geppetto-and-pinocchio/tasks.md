# Tasks

## TODO

- [x] Create the GP-55-HTTP-PROXY ticket workspace and scaffold the design doc plus diary.
- [x] Audit Geppetto shared section ownership and identify the correct section for proxy configuration.
- [x] Trace Pinocchio's runtime path from Cobra parsing through final `InferenceSettings` resolution.
- [x] Inspect provider engine/client construction seams for OpenAI, Claude, OpenAI Responses, and Gemini.
- [x] Write the intern-facing design and implementation guide.
- [x] Update Geppetto docs to explain base settings, profile overlays, and shared bootstrap ownership.
- [x] Update Pinocchio docs to explain hidden base settings, runtime switching, and the web-chat caveat.
- [x] Add ticket analysis for where `ai-client` flags should be exposed in `pinocchio` and `web-chat`.
- [x] Improve the Pinocchio parsed-base helper so a hidden base can be overlaid with parsed non-profile values.
- [x] Expose the shared `ai-client` section on `cmd/web-chat` and merge its parsed values into the preserved base inference settings.
- [x] Add regression tests for the parsed-base helper and the `web-chat` `ai-client` CLI/base merge path.
- [x] Implement proxy fields in `ClientSettings` and `client.yaml`.
- [x] Add a shared helper to build a proxy-aware `*http.Client` from `ClientSettings`.
- [x] Wire the helper into OpenAI, Claude, OpenAI Responses, and Gemini engine paths.
- [x] Add provider and Pinocchio regression tests for proxy propagation and usage.
- [x] Decide whether `cmd/web-chat` should expose explicit proxy CLI flags or remain config/env-only.
- [x] Run `docmgr doctor` and resolve any validation issues.
- [x] Dry-run and upload the ticket bundle to reMarkable, then verify the remote listing.
