# Tasks

## TODO

- [x] Create GEP-EMBPROF-001 ticket under the requested Geppetto `ttmp` root.
- [x] Capture a redacted snapshot of relevant current Pinocchio profile snippets without storing provider keys.
- [x] Write an intern-facing analysis/design/implementation guide for embedding-capable Geppetto profiles.
- [x] Record the initial documentation work in the ticket diary.
- [x] Upload the guide bundle to reMarkable.

## Future implementation tasks

- [x] Add example embedding profiles to an appropriate Geppetto/Pinocchio profile registry.
- [x] Add profile-resolution tests proving embedding profiles inherit API keys/base URLs from stacked base profiles.
- [x] Add a profile-aware embedding settings validation helper.
- [x] Update Geppetto embedding documentation with profile-backed examples.
- [x] Run Ollama and OpenAI smoke tests using profile-backed embedding providers.
- [x] Add focused engine profile stack tests for OpenAI and Ollama embedding profiles.
- [x] Add embeddings.ValidateInferenceSettingsForEmbeddings with profile-oriented errors.
- [x] Cover validation helper with table-driven unit tests for chat-only, OpenAI, Ollama, and unsupported providers.
- [x] Document profile-backed embedding usage and chat-profile failure mode in pkg/doc/topics/06-embeddings.md.
- [x] Remove OpenAI environment-key examples and make docs/examples resolve credentials through profiles.
