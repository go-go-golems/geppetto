# Tasks

## Done

- [x] Create Geppetto-local docmgr ticket workspace for OpenAI Responses multimodal media support
- [x] Review official OpenAI Responses image/file input documentation captured in HAIR-020
- [x] Audit Geppetto request-building code, token-count reuse, and existing tests
- [x] Produce a detailed analysis / design / implementation guide for a new intern
- [x] Record the investigation in the ticket diary
- [x] Implement image support in `pkg/steps/ai/openai_responses/helpers.go`
- [x] Add regression tests for image URL, base64 data URL, optional `detail`, and mixed text+image content
- [x] Add a token-count test proving `/responses/input_tokens` sees image-bearing requests too
- [x] Decide that canonical `input_file` / broader media support belongs in a follow-up turn-schema/design ticket; keep GP-53 focused on image parity plus optional `file_id` passthrough on `input_image`
- [x] Update Geppetto documentation/examples after the implementation lands
- [x] Run implementation validation and record it in this ticket

## Follow-up work

- [ ] Create a follow-up ticket for canonical provider-neutral file/media turn modeling if a concrete caller needs `input_file` or audio support
- [ ] Evaluate whether the OpenAI chat and OpenAI Responses media-normalization logic should be deduplicated after the behavior settles
- [ ] Consider whether assistant-side multimodal replay should become a supported part of turn history in the future
