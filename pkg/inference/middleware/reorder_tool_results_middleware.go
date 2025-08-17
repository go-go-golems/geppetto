package middleware

import (
    "context"

    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/rs/zerolog/log"
)

// NewToolResultReorderMiddleware ensures that for any contiguous group of tool_call blocks,
// the corresponding tool_use blocks appear immediately after, preserving call order.
// It does not alter the positions of non-tool blocks, except that it moves tool_use blocks
// forward to satisfy provider adjacency constraints (e.g., OpenAI tool messages).
func NewToolResultReorderMiddleware() Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            if t == nil || len(t.Blocks) == 0 {
                return next(ctx, t)
            }

            originalCount := len(t.Blocks)
            // Track indices of tool_use blocks that we relocate so we can skip them later
            movedIndex := make(map[int]bool)
            // Build a new sequence while scanning the original
            newBlocks := make([]turns.Block, 0, originalCount)

            for i := 0; i < len(t.Blocks); i++ {
                if movedIndex[i] {
                    continue
                }
                b := t.Blocks[i]
                if b.Kind != turns.BlockKindToolCall {
                    // Only append tool_use if not moved
                    if b.Kind == turns.BlockKindToolUse {
                        if movedIndex[i] {
                            continue
                        }
                    }
                    newBlocks = append(newBlocks, b)
                    continue
                }

                // We encountered a run of tool_call blocks. Collect them and their IDs.
                callStart := i
                callIDs := make([]string, 0, 4)
                for i < len(t.Blocks) && t.Blocks[i].Kind == turns.BlockKindToolCall {
                    newBlocks = append(newBlocks, t.Blocks[i]) // keep tool_call in place
                    if id, _ := t.Blocks[i].Payload[turns.PayloadKeyID].(string); id != "" {
                        callIDs = append(callIDs, id)
                    }
                    i++
                }
                // i currently points to the first non-tool_call after the run; adjust for for-loop increment
                i--

                // For these callIDs, find matching tool_use blocks later in the Turn and move them here
                // Preserve order according to callIDs, not discovery order.
                // Build index lists per ID to support multiple tool_use for the same id (rare but possible)
                idToIndices := make(map[string][]int)
                for j := callStart + 1; j < len(t.Blocks); j++ {
                    if movedIndex[j] {
                        continue
                    }
                    bj := t.Blocks[j]
                    if bj.Kind != turns.BlockKindToolUse {
                        continue
                    }
                    id, _ := bj.Payload[turns.PayloadKeyID].(string)
                    if id == "" {
                        continue
                    }
                    // Only collect tool_use that match one of the current call ids
                    idToIndices[id] = append(idToIndices[id], j)
                }

                movedForRun := 0
                for _, id := range callIDs {
                    idxs := idToIndices[id]
                    if len(idxs) == 0 {
                        continue
                    }
                    for _, idx := range idxs {
                        if movedIndex[idx] {
                            continue
                        }
                        newBlocks = append(newBlocks, t.Blocks[idx])
                        movedIndex[idx] = true
                        movedForRun++
                    }
                }

                if movedForRun > 0 {
                    log.Debug().Int("moved_tool_use", movedForRun).Int("start", callStart).Msg("tool-reorder: grouped tool_use after tool_call run")
                }
            }

            // Append any remaining blocks that were not moved and not yet appended.
            // This primarily handles trailing non-tool blocks when the loop ended on a tool_call run with no further blocks.
            if len(newBlocks) < originalCount {
                for i := 0; i < len(t.Blocks); i++ {
                    if movedIndex[i] {
                        continue
                    }
                    // Skip duplicates: if this block was already appended (by value), we cannot easily detect without ids.
                    // Rely on movedIndex to avoid only moved tool_use. For others, appending is safe.
                    if t.Blocks[i].Kind == turns.BlockKindToolUse {
                        // If not moved and still present here, keep original placement.
                        newBlocks = append(newBlocks, t.Blocks[i])
                    } else if t.Blocks[i].Kind != turns.BlockKindToolCall {
                        newBlocks = append(newBlocks, t.Blocks[i])
                    }
                }
            }

            // If we changed sequence length or order, replace and log
            if len(newBlocks) == originalCount {
                // Check if order changed by shallow comparison of pointers/IDs
                changed := false
                for i := range newBlocks {
                    if newBlocks[i].ID != t.Blocks[i].ID {
                        changed = true
                        break
                    }
                }
                if changed {
                    log.Debug().Msg("tool-reorder: applied block reordering to satisfy tool adjacency")
                    t.Blocks = newBlocks
                }
            } else {
                // Defensive: if counts diverge, keep original to avoid data loss
                if len(newBlocks) != 0 {
                    log.Warn().Int("old", originalCount).Int("new", len(newBlocks)).Msg("tool-reorder: block count changed after reorder; keeping original order")
                }
            }

            return next(ctx, t)
        }
    }
}


