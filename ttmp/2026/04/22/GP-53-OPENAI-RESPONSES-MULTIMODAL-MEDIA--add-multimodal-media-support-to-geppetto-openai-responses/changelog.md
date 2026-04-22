# Changelog

## 2026-04-22

- Created a Geppetto-local research ticket for OpenAI Responses multimodal media support
- Reviewed official OpenAI Responses docs for `input_text`, `input_image`, `image_url`, `detail`, and `input_file`
- Audited `pkg/steps/ai/openai_responses/helpers.go`, `pkg/steps/ai/openai/helpers.go`, `pkg/turns/helpers_blocks.go`, and `pkg/steps/ai/openai_responses/token_count.go`
- Wrote a detailed analysis / design / implementation guide and investigation diary
- Implemented OpenAI Responses `input_image` request-part support using the existing `PayloadKeyImages` turn payload shape
- Added regression tests for URL images, inline base64/data-URL images, mixed text+multiple-image content, and the token-count request path
- Updated Geppetto docs to describe the new Responses multimodal behavior and clarified that broader canonical file/media modeling should stay in a follow-up ticket
