# Changelog

## 2026-04-22

- Created a Geppetto-local research ticket for OpenAI Responses multimodal media support
- Reviewed official OpenAI Responses docs for `input_text`, `input_image`, `image_url`, `detail`, and `input_file`
- Audited `pkg/steps/ai/openai_responses/helpers.go`, `pkg/steps/ai/openai/helpers.go`, `pkg/turns/helpers_blocks.go`, and `pkg/steps/ai/openai_responses/token_count.go`
- Wrote a detailed analysis / design / implementation guide and investigation diary
- Implemented OpenAI Responses `input_image` request-part support using the existing `PayloadKeyImages` turn payload shape
- Added regression tests for URL images, inline base64/data-URL images, mixed text+multiple-image content, and the token-count request path
- Updated Geppetto docs to describe the new Responses multimodal behavior and clarified that broader canonical file/media modeling should stay in a follow-up ticket
- Added ticket-local reproduction scripts and a synthetic image fixture under `scripts/` and `sources/`
- Ran a live local Geppetto smoke against the real OpenAI Responses API that proved the model could read the image (`4319`, blue triangle) when the patched engine sent inline image data
- Also ran the installed `pinocchio --images` path against the same fixture and observed that it still replied that it could not see the image, indicating a likely follow-up issue outside the local GP-53 engine patch
