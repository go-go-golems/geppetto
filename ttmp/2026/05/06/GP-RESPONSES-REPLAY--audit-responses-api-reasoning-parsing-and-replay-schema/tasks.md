# Tasks

## TODO

- [x] Add tasks here

- [x] Move OpenAI Responses documentation snapshots into the Geppetto ticket sources directory
- [x] Write intern-oriented audit/design/implementation guide for Responses reasoning parsing and replay
- [x] Correct misleading code comment about official reasoning_text schema support
- [x] Upload Responses reasoning parsing/replay audit guide to reMarkable
- [x] Incorporate review decisions: keep item_id, no migration, use openai_responses metadata, no capabilities
- [x] Add typed Responses reasoning content support in request/response structs
- [x] Add redacted Responses input preview helper that shows item type, ids, summary counts, and encrypted-content presence
- [x] Store OpenAI Responses-specific item metadata under block metadata openai_responses.*
- [x] Replace streaming latestEncryptedContent with per-reasoning-item accumulator state
- [x] Parse and merge reasoning content[].reasoning_text from streaming output_item.done
- [x] Replay plaintext reasoning as official content[{type: reasoning_text}] instead of dropping it
- [x] Add regression tests for reasoning_text replay, empty reasoning omission, and item_id-only provider ID usage
- [x] Add regression tests for streaming reasoning_text terminal content and encrypted content isolation across reasoning items
- [x] Update Geppetto docs for Responses reasoning parsing/replay semantics
- [x] Re-upload the updated GP-RESPONSES-REPLAY guide to reMarkable after implementation
