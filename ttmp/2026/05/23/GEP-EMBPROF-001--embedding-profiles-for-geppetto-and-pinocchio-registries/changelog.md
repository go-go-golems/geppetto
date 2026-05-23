# Changelog

## 2026-05-23

- Initial workspace created


## 2026-05-23

Created embedding profiles analysis/design ticket, redacted profile source snapshot, intern-facing implementation guide, and diary.

### Related Files

- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp/2026/05/23/GEP-EMBPROF-001--embedding-profiles-for-geppetto-and-pinocchio-registries/design-doc/01-embedding-profiles-analysis-design-and-implementation-guide.md — Primary design guide
- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp/2026/05/23/GEP-EMBPROF-001--embedding-profiles-for-geppetto-and-pinocchio-registries/reference/01-diary.md — Diary entry for initial analysis


## 2026-05-23

Uploaded the embedding profiles guide bundle to reMarkable at /ai/2026/05/23/GEP-EMBPROF-001.

### Related Files

- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp/2026/05/23/GEP-EMBPROF-001--embedding-profiles-for-geppetto-and-pinocchio-registries/design-doc/01-embedding-profiles-analysis-design-and-implementation-guide.md — Uploaded primary guide
- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/ttmp/2026/05/23/GEP-EMBPROF-001--embedding-profiles-for-geppetto-and-pinocchio-registries/reference/01-diary.md — Records reMarkable upload


## 2026-05-23

Added embedding profile stack tests, profile-aware embedding settings validation, and profile-backed embedding documentation (code commit bf38f712).

### Related Files

- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/pkg/doc/topics/06-embeddings.md — User-facing documentation
- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/pkg/embeddings/settings_validation.go — Validation helper


## 2026-05-23

Added example embedding profile registry fixture for OpenAI and Ollama profile-backed embeddings.

### Related Files

- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/examples/js/geppetto/profiles/40-embeddings.yaml — Embedding profile examples


## 2026-05-23

Ran profile-backed Ollama embedding smoke successfully; OpenAI live smoke remains pending because process OpenAI key env var is not set.

### Related Files

- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/examples/js/geppetto/profiles/40-embeddings.yaml — Smoke-tested ollama-nomic-embedding profile


## 2026-05-23

Corrected OpenAI smoke approach: resolved the OpenAI key through the Pinocchio profile stack and successfully generated a text-embedding-3-small vector; task 10 complete.

### Related Files

- /home/manuel/.config/pinocchio/profiles.yaml — Runtime profile registry used for OpenAI key resolution (secret not copied)


## 2026-05-23

Removed OpenAI environment-key examples from active code/docs and aligned runner examples/E2E smoke with ~/.config/pinocchio/profiles.yaml profile resolution.

### Related Files

- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/cmd/examples/internal/runnerexample/inference_settings.go — Runner examples now resolve credentials through Pinocchio profiles
- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/pkg/doc/topics/06-embeddings.md — Embedding docs now use profile-backed provider construction
- /home/manuel/workspaces/2026-05-23/add-embeddings-profiles/geppetto/pkg/steps/ai/openai_responses/helpers_e2e_test.go — E2E test resolves OpenAI credentials through profiles

