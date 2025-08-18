package middleware

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// NewTurnLoggingMiddleware logs run/turn and block details before and after inference.
// It is safe to use with in-memory Turns that may not carry IDs; missing IDs are logged as empty.
func NewTurnLoggingMiddleware(logger zerolog.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
			lg := logger
			// fall back to global if uninitialized
			if lg.GetLevel() == zerolog.NoLevel {
				lg = log.Logger
			}

			lg = lg.With().
				Str("run_id", t.RunID).
				Str("turn_id", t.ID).
				Int("block_count", len(t.Blocks)).
				Logger()

			lg.Info().Msg("turn: starting inference")

			result, err := next(ctx, t)
			if err != nil {
				lg.Error().Err(err).Msg("turn: inference failed")
				return result, err
			}

			// Count kinds for summary
			var numUser, numLLM, numToolCall, numToolUse, numSystem, numOther int
			for _, b := range result.Blocks {
				switch b.Kind {
				case turns.BlockKindUser:
					numUser++
				case turns.BlockKindLLMText:
					numLLM++
				case turns.BlockKindToolCall:
					numToolCall++
				case turns.BlockKindToolUse:
					numToolUse++
				case turns.BlockKindSystem:
					numSystem++
				case turns.BlockKindOther:
					numOther++
				}
			}

			lg = lg.With().
				Int("result_block_count", len(result.Blocks)).
				Int("user_blocks", numUser).
				Int("llm_text_blocks", numLLM).
				Int("tool_call_blocks", numToolCall).
				Int("tool_use_blocks", numToolUse).
				Int("system_blocks", numSystem).
				Int("other_blocks", numOther).
				Logger()

			lg.Info().Msg("turn: inference completed")
			return result, nil
		}
	}
}
