package middleware

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// NewSystemPromptMiddleware returns a middleware that ensures a fixed system prompt
// is present as the first system block. If a system block already exists on the Turn,
// the prompt text is replaced. If no system block exists, a new one is inserted at
// the beginning of the Turn.
func NewSystemPromptMiddleware(prompt string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
			if t == nil {
				t = &turns.Turn{}
			}

			runID := ""
			sessionID := ""
			if sid, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
				sessionID = sid
			}
			runID = sessionID // backwards compatibility

			inferenceID := ""
			if iid, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok {
				inferenceID = iid
			}

			if prompt != "" {
				prev := prompt
				if len(prev) > 120 {
					prev = prev[:120] + "…"
				}
				log.Debug().
					Str("run_id", runID).
					Str("session_id", sessionID).
					Str("inference_id", inferenceID).
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
					// Replace the existing first system block
					if t.Blocks[firstSystemIdx].Payload == nil {
						t.Blocks[firstSystemIdx].Payload = map[string]any{}
					}
					existingText, _ := t.Blocks[firstSystemIdx].Payload[turns.PayloadKeyText].(string)
					if existingText == prompt {
						log.Debug().Str("run_id", runID).Str("session_id", sessionID).Str("inference_id", inferenceID).Str("turn_id", t.ID).Int("system_idx", firstSystemIdx).Msg("systemprompt: prompt already set on existing system block")
					} else {
						t.Blocks[firstSystemIdx].Payload[turns.PayloadKeyText] = prompt
						if err := turns.KeyBlockMetaMiddleware.Set(&t.Blocks[firstSystemIdx].Metadata, "systemprompt"); err != nil {
							return nil, errors.Wrap(err, "set block middleware metadata (existing system block)")
						}
						log.Debug().Str("run_id", runID).Str("session_id", sessionID).Str("inference_id", inferenceID).Str("turn_id", t.ID).Int("system_idx", firstSystemIdx).Msg("systemprompt: replaced text on existing system block")
					}
				} else {
					// Insert a new system block at the beginning
					newBlock := turns.NewSystemTextBlock(prompt)
					if err := turns.KeyBlockMetaMiddleware.Set(&newBlock.Metadata, "systemprompt"); err != nil {
						return nil, errors.Wrap(err, "set block middleware metadata (new system block)")
					}
					// Insert at index 0
					t.Blocks = append([]turns.Block{newBlock}, t.Blocks...)
					// Log roles snapshot after insertion
					roles := make([]string, 0, len(t.Blocks))
					for _, bb := range t.Blocks {
						roles = append(roles, bb.Role)
					}
					log.Debug().
						Str("run_id", runID).
						Str("session_id", sessionID).
						Str("inference_id", inferenceID).
						Str("turn_id", t.ID).
						Strs("roles_after", roles).
						Msg("systemprompt: inserted new system block at beginning")
				}
			}

			log.Debug().Str("run_id", runID).Str("session_id", sessionID).Str("inference_id", inferenceID).Str("turn_id", t.ID).Int("block_count", len(t.Blocks)).Msg("systemprompt: middleware end")
			return next(ctx, t)
		}
	}
}
