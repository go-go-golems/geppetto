# Tasks

## TODO

- [x] Create the ticket workspace and seed the primary docs.
- [x] Verify the official OpenAI and Anthropic token-count APIs with web research.
- [x] Audit the current `pinocchio` token-count command and `geppetto` provider request builders.
- [x] Write an intern-friendly analysis, design, and implementation guide with file references, diagrams, and pseudocode.
- [x] Extract reusable request-projection helpers in `geppetto` so inference and count paths do not duplicate Turn-to-provider conversion logic.
- [x] Implement OpenAI provider-native token counting against `POST /v1/responses/input_tokens`.
- [x] Implement Claude provider-native token counting against `POST /v1/messages/count_tokens`.
- [x] Extend `pinocchio tokens count` with `estimate|api|auto` behavior and geppetto-aware provider/profile flags.
- [x] Add unit tests for request building, HTTP success/error handling, and CLI behavior.
- [x] Update CLI/help docs after implementation lands.
- [x] Validate the ticket docs and deliver them to reMarkable.
