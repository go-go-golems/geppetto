package middleware

import (
    "context"

    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/rs/zerolog/log"
)

// NewSystemPromptMiddleware returns a middleware that ensures a fixed system prompt
// is present as the first system block. If a system block already exists on the Turn,
// the prompt text is appended to that first system block (separated by a blank line).
// If no system block exists, a new one is inserted at the beginning of the Turn.
func NewSystemPromptMiddleware(prompt string) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            if t == nil {
                t = &turns.Turn{}
            }

            if prompt != "" {
                prev := prompt
                if len(prev) > 120 { prev = prev[:120] + "…" }
                log.Debug().
                    Str("run_id", t.RunID).
                    Str("turn_id", t.ID).
                    Int("block_count", len(t.Blocks)).
                    Int("prompt_len", len(prompt)).
                    Str("prompt_preview", prev).
                    Msg("systemprompt: middleware start")
                // Find first system block
                firstSystemIdx := -1
                for i, b := range t.Blocks {
                    if b.Kind == turns.BlockKindSystem {
                        firstSystemIdx = i
                        break
                    }
                }

                if firstSystemIdx >= 0 {
                    // Append to existing first system block
                    if t.Blocks[firstSystemIdx].Payload == nil {
                        t.Blocks[firstSystemIdx].Payload = map[string]any{}
                    }
                    existingText, _ := t.Blocks[firstSystemIdx].Payload[turns.PayloadKeyText].(string)
                    if t.Blocks[firstSystemIdx].Metadata == nil {
                        t.Blocks[firstSystemIdx].Metadata = map[string]any{}
                    }
                    t.Blocks[firstSystemIdx].Metadata["middleware"] = "systemprompt"
                    if existingText == "" {
                        t.Blocks[firstSystemIdx].Payload[turns.PayloadKeyText] = prompt
                        log.Debug().Str("run_id", t.RunID).Str("turn_id", t.ID).Int("system_idx", firstSystemIdx).Msg("systemprompt: set text on existing system block")
                    } else {
                        t.Blocks[firstSystemIdx].Payload[turns.PayloadKeyText] = existingText + "\n\n" + prompt
                        log.Debug().Str("run_id", t.RunID).Str("turn_id", t.ID).Int("system_idx", firstSystemIdx).Msg("systemprompt: appended text to existing system block")
                    }
                } else {
                    // Insert a new system block at the beginning
                    newBlock := turns.WithBlockMetadata(turns.NewSystemTextBlock(prompt), map[string]any{"middleware": "systemprompt"})
                    // Insert at index 0
                    t.Blocks = append([]turns.Block{newBlock}, t.Blocks...)
                    // Log roles snapshot after insertion
                    roles := make([]string, 0, len(t.Blocks))
                    for _, bb := range t.Blocks { roles = append(roles, bb.Role) }
                    log.Debug().
                        Str("run_id", t.RunID).
                        Str("turn_id", t.ID).
                        Strs("roles_after", roles).
                        Msg("systemprompt: inserted new system block at beginning")
                }
            }

            log.Debug().Str("run_id", t.RunID).Str("turn_id", t.ID).Int("block_count", len(t.Blocks)).Msg("systemprompt: middleware end")
            return next(ctx, t)
        }
    }
}


