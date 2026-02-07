package middleware

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// NewTurnLoggingMiddleware logs session/turn and block details before and after inference.
// It is safe to use with in-memory Turns that may not carry IDs; missing IDs are logged as empty.
func NewTurnLoggingMiddleware(logger zerolog.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
			if t == nil {
				t = &turns.Turn{}
			}

			lg := logger
			// fall back to global if uninitialized
			if lg.GetLevel() == zerolog.NoLevel {
				lg = log.Logger
			}

			sessionID := ""
			if sid, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
				sessionID = sid
			}

			inferenceID := ""
			if iid, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok {
				inferenceID = iid
			}

			lg = lg.With().
				Str("session_id", sessionID).
				Str("inference_id", inferenceID).
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
			var numUser, numLLM, numToolCall, numToolUse, numSystem, numReasoning, numOther int
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
				case turns.BlockKindReasoning:
					numReasoning++
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
				Int("reasoning_blocks", numReasoning).
				Int("other_blocks", numOther).
				Logger()

			lg.Info().Msg("turn: inference completed")
			return result, nil
		}
	}
}
