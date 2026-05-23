# Tasks

## Phase 0 — Ticket, source capture, and design package

- [x] Create GEP-EMBPROF-001 under `/home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp`.
- [x] Capture a redacted snapshot of relevant current Pinocchio profile snippets without storing provider keys.
- [x] Write the intern-facing analysis/design/implementation guide.
- [x] Relate the guide to the relevant Geppetto profile, inference-settings, and embeddings files.
- [x] Record initial work in the ticket diary and changelog.
- [x] Upload the initial guide bundle to reMarkable at `/ai/2026/05/23/GEP-EMBPROF-001`.

## Phase 1 — Core profile semantics and validation helper

- [x] Add focused engine profile stack tests proving an OpenAI embedding profile can inherit `openai-api-key` from a base profile.
- [x] Add focused engine profile stack tests proving an Ollama embedding profile can inherit `ollama-base-url` from a base or self-contained profile.
- [x] Add `embeddings.ValidateInferenceSettingsForEmbeddings` for final, already-merged `InferenceSettings`.
- [x] Cover validation helper failure modes with table-driven unit tests:
  - nil settings;
  - chat-only profile with no `inference_settings.embeddings`;
  - missing embedding provider type;
  - missing embedding engine;
  - OpenAI profile without a resolved `openai-api-key`;
  - Ollama profile without explicit dimensions;
  - unsupported provider type;
  - valid OpenAI and Ollama profiles.
- [x] Run focused tests for `pkg/embeddings` and `pkg/engineprofiles`.
- [x] Commit the helper and tests.

## Phase 2 — Checked-in example embedding profiles

- [x] Add a checked-in example embedding profile registry fixture.
- [x] Include OpenAI profiles that stack a provider/base registry instead of duplicating secrets:
  - `openai-embedding-small`;
  - `openai-embedding-large`.
- [x] Include local Ollama profiles with explicit base URL, model, dimensions, and cache settings:
  - `ollama-nomic-embedding`;
  - `ollama-all-minilm-embedding`.
- [x] Update the example README to explain the embedding profile fixture.
- [x] Decode and resolve the fixture to prove the example shape is valid.
- [x] Commit the example profile registry and docs.

## Phase 3 — Documentation and consumer guidance

- [x] Document profile-backed embedding usage in `pkg/doc/topics/06-embeddings.md`.
- [x] Explain the chat-profile vs embedding-profile failure mode.
- [x] Show consumer pseudocode:
  - resolve profile registry;
  - validate final `InferenceSettings`;
  - construct provider through `NewSettingsFactoryFromInferenceSettings`.
- [x] Remove active examples that teach direct OpenAI provider-key environment-variable handling.
- [x] Update runner examples to default to `~/.config/pinocchio/profiles.yaml` and expose profile selection.
- [x] Commit documentation and example cleanup.

## Phase 4 — Smoke testing profile-backed providers

- [x] Run a local Ollama smoke using the profile-backed embedding path.
- [x] Confirm `nomic-embed-text` resolves to 768 dimensions.
- [x] Correct the OpenAI smoke workflow to resolve credentials through profiles rather than process environment variables.
- [x] Run or document OpenAI profile-backed smoke without printing or copying provider keys.
- [x] Record smoke outcomes in the diary and changelog.

## Phase 5 — Real Pinocchio user registry installation

- [x] Back up `~/.config/pinocchio/profiles.yaml` before editing.
- [x] Add `openai-embedding-small` to `~/.config/pinocchio/profiles.yaml`, stacked on `openai-responses-base`.
- [x] Add `openai-embedding-large` to `~/.config/pinocchio/profiles.yaml`, stacked on `openai-responses-base`.
- [x] Add `ollama-nomic-embedding` to `~/.config/pinocchio/profiles.yaml` with `ollama-base-url: http://localhost:11434`.
- [x] Add `ollama-all-minilm-embedding` to `~/.config/pinocchio/profiles.yaml` with `ollama-base-url: http://localhost:11434`.
- [x] Validate the edited YAML parses.
- [x] Resolve and validate all four newly installed profiles with Geppetto's current profile and embedding validation APIs.
- [x] Refresh the redacted ticket source snapshot to show the installed embedding profile shapes without secrets.
- [x] Record the user-registry installation in the diary and changelog.

## Phase 6 — Downstream adoption follow-ups

- [ ] Update downstream consumers, such as Readwise Viewer, to use a Geppetto version containing `ValidateInferenceSettingsForEmbeddings` and profile-resolved API decoding.
- [ ] Rebuild downstream binaries and rerun the original vector-search command with `--profile openai-embedding-small` or `--profile ollama-nomic-embedding`.
- [ ] Add downstream error handling so selecting `gpt-5-low` for vector search says it is a chat profile, not an embedding-capable profile.
- [ ] Optionally add a small CLI/profile inspection command that reports embedding provider metadata with secrets redacted.
- [ ] Add a small CLI smoke tool that loads Pinocchio profiles and computes one embedding.
