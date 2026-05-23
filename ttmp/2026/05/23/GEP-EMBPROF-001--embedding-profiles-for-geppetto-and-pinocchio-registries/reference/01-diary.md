---
Title: Diary
Ticket: GEP-EMBPROF-001
Status: active
Topics:
    - geppetto
    - embeddings
    - profiles
    - vector-search
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: examples/js/geppetto/README.md
      Note: Documents the new embedding profile fixture
    - Path: examples/js/geppetto/profiles/40-embeddings.yaml
      Note: Example embedding profile registry with OpenAI stacked profiles and Ollama local profiles
    - Path: pkg/doc/topics/06-embeddings.md
      Note: Documents profile-backed embedding profile shapes and consumer validation
    - Path: pkg/embeddings/settings_validation.go
      Note: Adds profile-aware final InferenceSettings validation for embedding providers (commit bf38f712)
    - Path: pkg/embeddings/settings_validation_test.go
      Note: Covers validation helper failure modes and supported providers (commit bf38f712)
    - Path: pkg/engineprofiles/stack_merge_embeddings_test.go
      Note: Proves embedding profiles inherit stacked API keys and base URLs (commit bf38f712)
    - Path: ttmp/2026/05/23/GEP-EMBPROF-001--embedding-profiles-for-geppetto-and-pinocchio-registries/design-doc/01-embedding-profiles-analysis-design-and-implementation-guide.md
      Note: Initial intern-facing design guide
    - Path: ttmp/2026/05/23/GEP-EMBPROF-001--embedding-profiles-for-geppetto-and-pinocchio-registries/sources/pinocchio-profiles-redacted-relevant-snippets.yaml
      Note: Initial source snapshot
    - Path: ttmp/2026/05/23/GEP-EMBPROF-001--embedding-profiles-for-geppetto-and-pinocchio-registries/tasks.md
      Note: Initial task checklist
ExternalSources: []
Summary: Chronological diary for the embedding profiles analysis/design ticket.
LastUpdated: 2026-05-23T12:49:15.130617936-04:00
WhatFor: Use to resume or review the documentation/design work for embedding-capable Geppetto profiles.
WhenToUse: When continuing GEP-EMBPROF-001 or checking what was investigated and documented.
---




# Diary

## Goal

This diary records the creation of the GEP-EMBPROF-001 ticket and the first analysis/design guide for adding embedding-specific profiles to the existing Geppetto/Pinocchio profile registry workflow.

## Step 1: Create embedding profiles analysis ticket and guide

I created a dedicated docmgr ticket in the Geppetto workspace under the requested `ttmp` root and wrote a detailed intern-facing analysis/design/implementation guide. The guide explains why a chat profile such as `gpt-5-low` is not automatically an embedding profile, how Geppetto engine profiles resolve into final `InferenceSettings`, how the embeddings package consumes those settings, and how to add explicit OpenAI/Ollama embedding profiles that reuse existing provider key/base profiles.

I also captured a redacted snapshot of the relevant current Pinocchio profile snippets so the guide can refer to the real shape of `openai-responses-base`, `ollama-openai-base`, and `gpt-5-low` without storing secrets.

### Prompt Context

**User prompt (verbatim):** "Ok, let's create a ticket under /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp (docmgr --root ...) 

and then Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a docmgr ticket in the specified Geppetto workspace root, write a detailed technical guide for implementing embedding-capable profiles using existing Geppetto/Pinocchio profile mechanisms, and upload the resulting documentation bundle to reMarkable.

**Inferred user intent:** The user wants durable handoff documentation for an intern so the embedding-profile problem can be implemented correctly without adding ad hoc provider-key handling to consumer applications.

**Commit (code):** N/A — documentation-only step at this point.

### What I did

- Created ticket `GEP-EMBPROF-001` under:
  - `/home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp`
- Added design document:
  - `design-doc/01-embedding-profiles-analysis-design-and-implementation-guide.md`
- Added this diary:
  - `reference/01-diary.md`
- Added a redacted source snapshot:
  - `sources/pinocchio-profiles-redacted-relevant-snippets.yaml`
- Read and used the following Geppetto files:
  - `pkg/cli/bootstrap/profile_selection.go`
  - `pkg/cli/bootstrap/profile_runtime.go`
  - `pkg/cli/bootstrap/engine_settings.go`
  - `pkg/engineprofiles/types.go`
  - `pkg/engineprofiles/inference_settings_merge.go`
  - `pkg/steps/ai/settings/settings-inference.go`
  - `pkg/embeddings/config/settings.go`
  - `pkg/embeddings/config/flags/embeddings.yaml`
  - `pkg/embeddings/settings_factory.go`
  - `pkg/embeddings/openai.go`
  - `pkg/embeddings/ollama.go`
  - `pkg/doc/topics/06-embeddings.md`
- Analyzed the current `~/.config/pinocchio/profiles.yaml` shape with secrets redacted.

### Why

- The failure `no API key provided for OpenAI` was not primarily a request to pass keys differently; it exposed a missing profile convention for embeddings.
- The existing profile stack machinery is the right place for provider keys and base URLs.
- Intern-facing documentation needs to explain the system boundaries before proposing code changes, otherwise the implementation could accidentally add a second credential system.

### What worked

- `docmgr --root /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp` successfully created the ticket and documents.
- The current Geppetto workspace already contains the newer `InferenceSettings` shape with `yaml:"api,omitempty"`, which matches the current Pinocchio profile registry structure.
- A redacted profile snapshot was created without copying API keys into the ticket.

### What didn't work

- The original command used a chat profile (`gpt-5-low`) as an embedding profile. That cannot work reliably unless the profile also contains `inference_settings.embeddings`.
- No code implementation or live embedding smoke was performed in this step; this was the requested design/guide pass.

### What I learned

- Geppetto's current workspace has the correct conceptual bridge already: `embeddings.NewSettingsFactoryFromInferenceSettings` copies `InferenceSettings.API` and `InferenceSettings.Embeddings` into the embedding provider config.
- The missing piece is mostly profile authoring, validation, tests, and documentation rather than a new provider-key mechanism.
- The strongest implementation rule is: validate final merged settings after profile stack resolution, not raw YAML.

### What was tricky to build

- The tricky part was separating three concepts that look similar from the command line: provider key profiles, chat profiles, and embedding profiles. They all live in one registry and all use `InferenceSettings`, but they configure different fields.
- The guide needed to preserve the user's key-management rule while still explaining why `gpt-5-low` failed. The correct explanation is not “the key is missing from the registry”; it is “the selected profile is not embedding-capable, and the application should report that more clearly.”

### What warrants a second pair of eyes

- Confirm whether standard embedding profiles should be added to Geppetto's `misc/profiles.yaml`, Pinocchio's user registry, or both.
- Confirm cache-directory semantics: relative to working directory, registry location, or app config directory.
- Review the proposed validation API name and package location before implementation.

### What should be done in the future

- Add embedding profile examples and profile-resolution tests.
- Add a profile-aware embedding settings validation helper.
- Update Geppetto embedding docs with profile-backed usage.
- Run local Ollama smoke first, then OpenAI smoke only after confirming key values are loaded from the profile registry and never printed.

### Code review instructions

- Start with the design guide:
  - `design-doc/01-embedding-profiles-analysis-design-and-implementation-guide.md`
- Then inspect the source snapshot:
  - `sources/pinocchio-profiles-redacted-relevant-snippets.yaml`
- Review the referenced Geppetto files listed in the guide's `References` section.
- Validate the ticket with:
  - `docmgr --root /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp doctor --ticket GEP-EMBPROF-001`

### Technical details

The intended profile shape is:

```yaml
profiles:
  openai-embedding-small:
    stack:
      - profile_slug: openai-responses-base
    inference_settings:
      embeddings:
        type: openai
        engine: text-embedding-3-small
        dimensions: 1536
        cache_type: file
        cache_directory: ./.geppetto/embeddings-cache/openai-text-embedding-3-small
```

The intended consumer code shape is:

```go
resolved, err := profilebootstrap.ResolveCLIEngineSettings(ctx, parsed)
factory := embeddings.NewSettingsFactoryFromInferenceSettings(resolved.FinalInferenceSettings)
provider, err := factory.NewProvider()
```

## Step 2: Upload guide bundle to reMarkable

I uploaded the GEP-EMBPROF-001 guide bundle to reMarkable so the embedding profiles design can be reviewed away from the terminal. The uploaded bundle includes the main design guide, diary, and task checklist.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete the requested durable documentation workflow by publishing the guide to reMarkable.

**Inferred user intent:** The user wants the intern-facing guide available on reMarkable for reading and review.

**Commit (code):** N/A — documentation/upload step.

### What I did

- Ran `remarquee upload bundle` for:
  - `design-doc/01-embedding-profiles-analysis-design-and-implementation-guide.md`
  - `reference/01-diary.md`
  - `tasks.md`
- Uploaded as:
  - `GEP EMBPROF 001 Embedding Profiles Guide.pdf`
- Remote path:
  - `/ai/2026/05/23/GEP-EMBPROF-001`
- Marked the reMarkable task complete.

### Why

- The user explicitly asked to upload the ticket guide to reMarkable.
- Bundling the guide with the diary and checklist gives the reviewer both technical content and work context.

### What worked

- Upload succeeded with:
  - `OK: uploaded GEP EMBPROF 001 Embedding Profiles Guide.pdf -> /ai/2026/05/23/GEP-EMBPROF-001`

### What didn't work

- N/A. No upload retry or reauthentication was required.

### What I learned

- The guide's markdown rendered sufficiently for the bundle upload path; no pandoc syntax errors were encountered.

### What was tricky to build

- The main constraint was keeping the upload bundle focused. I included the main guide, diary, and tasks, but not the redacted YAML source as a separate source-code PDF because the core profile snippets are already embedded in the design guide.

### What warrants a second pair of eyes

- Review the generated PDF visually on reMarkable if formatting fidelity matters for diagrams and long tables.

### What should be done in the future

- Re-upload with `--force` if the guide is materially revised after implementation details are added.

### Code review instructions

- Review the reMarkable document at `/ai/2026/05/23/GEP-EMBPROF-001`.
- Cross-check against the source markdown in the ticket workspace.

### Technical details

Upload command shape:

```bash
remarquee upload bundle <guide.md> <diary.md> <tasks.md> \
  --name "GEP EMBPROF 001 Embedding Profiles Guide" \
  --remote-dir "/ai/2026/05/23/GEP-EMBPROF-001" \
  --toc-depth 2 \
  --non-interactive
```

## Step 3: Add embedding profile validation and stack tests

I moved the ticket from design-only into implementation by adding focused tests for embedding profile stack resolution and a validation helper for final merged inference settings. The tests prove that an OpenAI embedding profile can inherit `openai-api-key` from a stacked base profile and that an Ollama embedding profile can inherit `ollama-base-url`, while the helper gives profile-oriented errors before provider construction.

I also added a profile-backed usage section to the embeddings documentation so consumer applications have a concrete integration path: resolve profiles, validate `InferenceSettings`, then create a provider with `NewSettingsFactoryFromInferenceSettings`.

### Prompt Context

**User prompt (verbatim):** "Work on @geppetto/ttmp/2026/05/23/GEP-EMBPROF-001--embedding-profiles-for-geppetto-and-pinocchio-registries/ docmgr ticket, add tasks to the ticket after reading the design doc, then work on them incrementally, committing at a ppropriate intervals, and keeping a detailed diary."

**Assistant interpretation:** Continue the existing GEP-EMBPROF-001 docmgr ticket, derive concrete implementation tasks from the design guide, implement them in small increments, commit when useful, and keep the ticket diary/changelog current.

**Inferred user intent:** The user wants the design converted into working Geppetto code and docs without losing traceability between ticket tasks, commits, tests, and review notes.

**Commit (code):** bf38f712aa30394708e339de7cfec1cc339fa74e — "Add embedding profile validation tests"

### What I did

- Read the design guide, existing diary, and task checklist.
- Added implementation tasks for stack tests, validation, validation tests, and documentation.
- Added `pkg/engineprofiles/stack_merge_embeddings_test.go` with OpenAI and Ollama embedding profile stack-resolution tests.
- Added `pkg/embeddings/settings_validation.go` with `ValidateInferenceSettingsForEmbeddings`.
- Added `pkg/embeddings/settings_validation_test.go` with table-driven coverage for nil settings, chat-only profiles, missing type/engine, missing OpenAI keys, unsupported providers, Ollama dimensions, and valid OpenAI/Ollama settings.
- Updated `pkg/doc/topics/06-embeddings.md` with profile-backed embedding examples and consumer integration code.
- Marked tasks 7, 8, 9, 11, 12, 13, and 14 complete.
- Related the modified implementation and documentation files to this diary.
- Updated the ticket changelog.

### Why

- The design doc's core implementation rule is that applications should validate final merged profile settings instead of managing provider credentials themselves.
- Tests make the intended profile composition explicit: base profiles carry API keys/base URLs, embedding profiles carry model/dimension/cache settings.
- The validation helper gives consumer commands a stable place to surface helpful profile errors before low-level providers return less actionable messages such as `no API key provided for OpenAI`.

### What worked

- Focused package tests passed:
  - `go test ./pkg/embeddings ./pkg/engineprofiles -count=1`
  - `go test ./pkg/doc ./pkg/embeddings ./pkg/engineprofiles -count=1`
- The commit pre-commit hook also ran the full test and lint suite successfully:
  - `go test ./...`
  - `golangci-lint run -v --max-same-issues=100 --timeout=5m ./cmd/... ./pkg/...`
  - `go vet -vettool=/tmp/geppetto-lint ...`
- `docmgr task check` updated the ticket checklist cleanly.
- `docmgr doc relate` added focused code/documentation file relationships to this diary.

### What didn't work

- N/A for this implementation step. The first code/test pass compiled, passed focused tests, and passed the repository pre-commit hook.

### What I learned

- The existing engine profile merge path already preserves nested `api` maps and overlays `embeddings` blocks correctly; the new tests primarily lock down that behavior for embedding-specific profiles.
- The most useful validation boundary is still after stack resolution: raw profile YAML cannot tell whether a missing key will be provided by a base profile.
- OpenAI can technically default dimensions later in `SettingsFactory.NewProvider`, but profile-backed documentation and tests should still encourage explicit dimensions for predictable vector-index metadata.

### What was tricky to build

- The main sharp edge was deciding how profile-aware the embeddings package should become. I kept the helper focused on final `InferenceSettings` rather than taking a profile slug, registry, or raw YAML, which preserves separation from the engineprofiles package while still giving applications better errors.
- Another subtle point was OpenAI dimensions: the low-level provider factory can default dimensions to 1536, but the validation helper currently only requires type, engine, and key for OpenAI. This avoids rejecting older direct configurations while the docs recommend explicit dimensions for profiles.
- The documentation had to distinguish three ways to configure embeddings: direct providers, `EmbeddingsConfig`, and engine profiles. I inserted the profile-backed section near settings configuration so it appears before lower-level parsed-values examples.

### What warrants a second pair of eyes

- Review whether `ValidateInferenceSettingsForEmbeddings` should require OpenAI dimensions as strictly as Ollama or only warn/document the preference.
- Review the exported helper name and package placement before downstream applications adopt it.
- Confirm whether supported provider strings should be constants exported from the embeddings package rather than private strings in the validation file.

### What should be done in the future

- Add concrete example embedding profiles to a real registry file if the team decides Geppetto should ship them rather than only documenting them.
- Run local Ollama and OpenAI smoke tests with real profile registries when credentials/services are available.
- Consider a CLI inspection/validation command for embedding-capable profiles.

### Code review instructions

- Start with `pkg/embeddings/settings_validation.go` and confirm the error messages match the intended profile-backed UX.
- Then review `pkg/embeddings/settings_validation_test.go` to check supported and rejected shapes.
- Review `pkg/engineprofiles/stack_merge_embeddings_test.go` to confirm the stack tests reflect the desired OpenAI/Ollama profile patterns.
- Review `pkg/doc/topics/06-embeddings.md` for the new “Using Embeddings from Engine Profiles” section.
- Validate with:
  - `go test ./pkg/doc ./pkg/embeddings ./pkg/engineprofiles -count=1`
  - optionally `go test ./...`

### Technical details

The new validation API is:

```go
func ValidateInferenceSettingsForEmbeddings(s *settings.InferenceSettings) error
```

It validates:

- non-nil final inference settings;
- presence of `inference_settings.embeddings`;
- `embeddings.type` and `embeddings.engine`;
- supported provider type (`openai` or `ollama`);
- `openai-api-key` for OpenAI profiles, after stack resolution;
- non-zero dimensions for Ollama profiles.

## Step 4: Add example embedding profile registry fixture

I added a concrete example registry fixture under the existing JS profile examples so there is a checked-in YAML shape for profile-backed embeddings. The fixture includes OpenAI embedding profiles that stack the existing provider registry for the API key and local Ollama profiles that carry explicit base URLs, dimensions, and cache settings.

This turns the design guide's recommended profile shapes into copyable repository examples without putting secrets into the repo. I also updated the examples README so future readers know the embeddings fixture exists and what relationship it has to the provider fixture.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Continue implementing ticket tasks incrementally after the initial validation/test/doc work.

**Inferred user intent:** The user wants the ticket's implementation checklist driven toward usable examples, not only helper APIs and prose documentation.

**Commit (code):** 8d045bb8067423732624929fb30931fdc95cefdd — "Add example embedding profile registry"

### What I did

- Added `examples/js/geppetto/profiles/40-embeddings.yaml`.
- Included four example profiles:
  - `openai-embedding-small`
  - `openai-embedding-large`
  - `ollama-nomic-embedding`
  - `ollama-all-minilm-embedding`
- Made the OpenAI profiles stack `provider-openai/default` so they inherit `openai-api-key` from the existing provider fixture.
- Updated `examples/js/geppetto/README.md` to list and explain the new fixture.
- Marked task 6 complete.
- Related the fixture and README to the ticket diary.
- Updated the ticket changelog.

### Why

- The design doc asked for explicit embedding profiles rather than overloading chat profiles.
- A repository fixture gives downstream applications and reviewers a known-good registry shape to copy and adapt.
- Keeping OpenAI credentials in the existing provider fixture demonstrates the intended separation between provider/base profiles and embedding-specific profiles.

### What worked

- The new registry decoded successfully with `DecodeEngineProfileYAMLSingleRegistry`:
  - `go run /tmp/check_embedding_profiles.go examples/js/geppetto/profiles/10-provider-openai.yaml examples/js/geppetto/profiles/40-embeddings.yaml`
- Cross-registry resolution and validation succeeded for both OpenAI and Ollama examples:
  - `go run /tmp/check_embedding_profiles_resolve.go examples/js/geppetto/profiles/10-provider-openai.yaml examples/js/geppetto/profiles/40-embeddings.yaml`
  - output included `openai-embedding-small ok` and `ollama-nomic-embedding ok`.
- The commit pre-commit hook skipped Go test/lint because only YAML/Markdown/ticket files were staged.

### What didn't work

- N/A. The YAML decoded and the selected profiles resolved/validated with the new helper.

### What I learned

- The examples registry directory is a better place for checked-in embedding profile examples than `misc/profiles.yaml`, which currently uses an older/simple configuration shape rather than the engine profile registry schema.
- Cross-registry stacking works for OpenAI embedding profiles as expected when both the provider and embeddings registries are loaded together.

### What was tricky to build

- The main decision was where to place the example profiles. `misc/profiles.yaml` was named in the design guide, but the actual file currently does not use the newer `slug`/`profiles` engine registry shape, while `examples/js/geppetto/profiles` already contains runtime YAML registry fixtures.
- I avoided copying real keys or inventing a second provider base. Instead, the OpenAI embedding examples stack the existing `provider-openai/default` fixture, which demonstrates credential reuse and keeps the embedding profiles focused on model/dimension/cache settings.

### What warrants a second pair of eyes

- Confirm whether `examples/js/geppetto/profiles/40-embeddings.yaml` is the right long-term home, or whether the project should add a dedicated non-JS profile fixture directory.
- Confirm whether cache directories in shared profile examples should stay relative to the caller working directory or use an app/config-root convention.

### What should be done in the future

- Wire the new embeddings fixture into an automated example or docs test if profile-backed embedding resolution becomes part of CI.
- Add a profile inspection command or smoke command when the CLI surface is ready.

### Code review instructions

- Review `examples/js/geppetto/profiles/40-embeddings.yaml` first.
- Check that OpenAI profiles stack `provider-openai/default` and do not duplicate secrets.
- Check that Ollama profiles set explicit dimensions matching the documented model expectations.
- Validate manually with the same temporary decode/resolve commands recorded above, or convert them into a permanent test if desired.

### Technical details

The key OpenAI example shape is:

```yaml
profiles:
  openai-embedding-small:
    stack:
      - registry_slug: provider-openai
        profile_slug: default
    inference_settings:
      embeddings:
        type: openai
        engine: text-embedding-3-small
        dimensions: 1536
```

## Step 5: Smoke-test the Ollama embedding profile path

I ran a live local smoke test for the new `ollama-nomic-embedding` profile fixture. The test loaded the embedding registry, resolved the profile, validated the final settings with `ValidateInferenceSettingsForEmbeddings`, constructed a provider through `NewSettingsFactoryFromInferenceSettings`, and generated a real embedding through the local Ollama service.

I did not mark the full smoke-test task complete because the OpenAI live smoke still needs a real `OPENAI_API_KEY` and would send text to an external paid provider. The local profile-backed path is verified; the remote OpenAI path remains an explicit follow-up.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Continue incremental ticket work by validating the profile-backed embedding path where safe and available.

**Inferred user intent:** The user wants implementation evidence, not just compile-time tests, while preserving provider-key and privacy discipline.

**Commit (code):** N/A — smoke-test documentation only.

### What I did

- Checked local Ollama availability with `ollama list`.
- Confirmed `nomic-embed-text` and `all-minilm` were already installed locally.
- Ran a temporary Go smoke program at `/tmp/smoke_ollama_embedding_profile.go` that:
  - decoded `examples/js/geppetto/profiles/40-embeddings.yaml`;
  - resolved `ollama-nomic-embedding` from the `embeddings` registry;
  - validated the final settings;
  - constructed the provider via `NewSettingsFactoryFromInferenceSettings`;
  - generated an embedding for `hello profile-backed embeddings`;
  - asserted actual vector length matched configured dimensions.
- Checked whether `OPENAI_API_KEY` was available for a live OpenAI smoke; it was not set.
- Updated the ticket changelog with the partial smoke result.

### Why

- The design calls for validating profile-backed providers end-to-end.
- Ollama is the safest first smoke because it is local and does not require external credentials or paid API calls.
- OpenAI should only be smoke-tested when a real key is intentionally available and the user is comfortable sending a small test string to OpenAI.

### What worked

- `ollama list` succeeded and showed `nomic-embed-text:latest` installed.
- The live smoke command succeeded:
  - `go run /tmp/smoke_ollama_embedding_profile.go examples/js/geppetto/profiles/40-embeddings.yaml`
- Output:
  - `model=nomic-embed-text dimensions=768 actual=768`

### What didn't work

- The OpenAI live smoke was not run because:
  - `OPENAI_API_KEY_not_set`
- I intentionally left task 10 open because it explicitly includes both Ollama and OpenAI smoke tests.

### What I learned

- The local profile-backed path works all the way from YAML registry to resolved settings to provider construction to actual vector generation.
- The fixture can be smoke-tested without loading the OpenAI provider registry when testing Ollama-only profiles because the Ollama profile is self-contained.

### What was tricky to build

- The key boundary was avoiding a false “complete” status for smoke testing. Ollama passed, but OpenAI requires a real secret and network call. I recorded the successful local evidence while keeping the broader task open.
- The temporary smoke code uses the exported registry decode and store APIs rather than application-specific bootstrap code, so it validates the core Geppetto profile/provider path without depending on a downstream app.

### What warrants a second pair of eyes

- Decide whether to add a permanent non-network smoke or example command for profile-backed embeddings.
- Decide the policy for OpenAI smoke tests: manual only, environment-gated, or skipped in CI.

### What should be done in the future

- Run the OpenAI smoke when `OPENAI_API_KEY` is intentionally set or when a real Pinocchio profile registry with a valid key is available.
- Consider adding a small reusable CLI/demo command that resolves a profile and embeds one string.

### Code review instructions

- No source code changed in this step.
- To reproduce the local smoke, use the temporary program shape from this diary or create a permanent test/demo from it.
- Ensure Ollama is running and `nomic-embed-text` is installed before reproducing.

### Technical details

Successful smoke output:

```text
model=nomic-embed-text dimensions=768 actual=768
```

## Step 6: Correct OpenAI smoke to resolve credentials through profiles

I corrected my earlier mistake: checking `OPENAI_API_KEY` was the wrong operational test for this ticket. The whole point of GEP-EMBPROF-001 is that provider credentials should be resolved through Geppetto/Pinocchio profiles, not through application-specific or process-environment key handling.

I reran the OpenAI smoke by loading the real Pinocchio profile registry, adding an in-memory temporary embedding profile that stacks `openai-responses-base`, resolving the final merged `InferenceSettings`, validating it, and generating an embedding. This proved the OpenAI API key is available through the profile stack without printing or copying the secret.

### Prompt Context

**User prompt (verbatim):** "ok wait, we don't need OPENAI_API_KEY, we should resolve it through the profiles. What is this about."

**Assistant interpretation:** The user is correcting my mistaken environment-variable framing and asking why OpenAI smoke was not attempted through the profile registry as designed.

**Inferred user intent:** The user wants the implementation and validation workflow to respect the architecture: Geppetto/Pinocchio profiles own provider keys, while embedding profiles stack those bases.

**Commit (code):** N/A — smoke-test correction and ticket documentation only.

### What I did

- Acknowledged that checking `OPENAI_API_KEY` was the wrong criterion.
- Inspected the runtime Pinocchio registry only enough to confirm it exists and did not print secrets.
- Created a temporary in-memory profile named `tmp-openai-embedding-small` that stacks `openai-responses-base` and adds:
  - `embeddings.type: openai`
  - `embeddings.engine: text-embedding-3-small`
  - `embeddings.dimensions: 1536`
- Resolved that profile through the engine profile registry APIs.
- Ran `ValidateInferenceSettingsForEmbeddings` on the resolved final settings.
- Constructed an embedding provider with `NewSettingsFactoryFromInferenceSettings`.
- Generated a live OpenAI embedding and verified dimensions.
- Marked task 10 complete.
- Updated the ticket changelog.

### Why

- The ticket explicitly rejects bespoke credential paths for consumer applications.
- The correct smoke is not “does the shell have `OPENAI_API_KEY`?”; it is “does the selected profile stack resolve an OpenAI key into final `InferenceSettings.API.APIKeys`?”
- The temporary in-memory profile avoided modifying the real Pinocchio registry while still testing the intended stack shape.

### What worked

- OpenAI profile-backed smoke succeeded with output:
  - `profile=tmp-openai-embedding-small profile=tmp-openai-embedding-small model=text-embedding-3-small dimensions=1536 actual=1536 key_configured=true`
- The key was resolved through `~/.config/pinocchio/profiles.yaml` via `openai-responses-base`.
- No API key value was printed or copied into the ticket.

### What didn't work

- My earlier Step 5 conclusion was wrong/incomplete because it treated the absence of `OPENAI_API_KEY` as a blocker. That contradicted the ticket architecture.
- The first version of the temporary smoke program tried to print `provider.GetModel().Provider`, but `EmbeddingModel` has only `Name` and `Dimensions`; the command failed with:
  - `/tmp/smoke_openai_profile_overlay.go:30:165: provider.GetModel().Provider undefined (type embeddings.EmbeddingModel has no field or method Provider)`
- I removed that field from the output and reran successfully.

### What I learned

- The existing Pinocchio base profile already provides the OpenAI key correctly for stacked embedding profiles.
- The validation helper correctly accepts the final merged settings once the temporary embedding profile stacks `openai-responses-base`.
- Future smoke documentation should explicitly say to resolve a profile registry, not check provider environment variables directly.

### What was tricky to build

- The important nuance is that the real registry currently has base and chat profiles but no dedicated embedding profile. To avoid editing the user's real registry during a smoke, I added the embedding profile in memory and stacked the existing base profile.
- This is still a valid architecture smoke because it exercises the same merge path a permanent `openai-embedding-small` profile would use.

### What warrants a second pair of eyes

- Decide whether to add `openai-embedding-small` directly to the user's Pinocchio registry or keep only repository examples until the user explicitly asks.
- Review whether a permanent smoke/demo command should support an overlay profile file so users do not need temporary Go programs.

### What should be done in the future

- Replace temporary smoke code with a maintained CLI/demo if profile-backed embedding testing becomes a common workflow.
- Consider documenting an overlay-registry smoke pattern for users who have base profiles but have not yet committed embedding profiles to their real registry.

### Code review instructions

- No production source changed in this correction.
- Review the earlier validation helper and stack tests with this corrected smoke in mind: the helper should be used after profile resolution and must not assume environment-variable credential plumbing.

### Technical details

Temporary smoke command:

```bash
go run /tmp/smoke_openai_profile_overlay.go
```

Successful output:

```text
profile=tmp-openai-embedding-small profile=tmp-openai-embedding-small model=text-embedding-3-small dimensions=1536 actual=1536 key_configured=true
```
