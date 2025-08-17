package middleware

import (
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// SnapshotBlockIDs captures the set of block IDs currently present on a Turn.
func SnapshotBlockIDs(t *turns.Turn) map[string]struct{} {
	ids := make(map[string]struct{}, 16)
	if t == nil {
		return ids
	}
	for _, b := range t.Blocks {
		if b.ID == "" {
			continue
		}
		ids[b.ID] = struct{}{}
	}
	return ids
}

// NewBlocksNotIn returns blocks on the provided Turn whose IDs are not present
// in the baseline set. This is resilient to reordering, removals, and insertions
// by other middlewares.
func NewBlocksNotIn(t *turns.Turn, baselineIDs map[string]struct{}) []turns.Block {
	if t == nil {
		return nil
	}
	if baselineIDs == nil {
		// If no baseline, consider all blocks as new
		return append([]turns.Block(nil), t.Blocks...)
	}
	added := make([]turns.Block, 0)
	for _, b := range t.Blocks {
		if b.ID == "" {
			// Without ID, we cannot reliably diff; skip
			continue
		}
		if _, ok := baselineIDs[b.ID]; !ok {
			added = append(added, b)
		}
	}
	return added
}
