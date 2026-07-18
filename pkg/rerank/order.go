package rerank

import (
	"fmt"
	"math"
	"sort"
)

// ValidateAndMapResults validates provider-returned raw index/score pairs,
// maps them to caller document IDs, and returns deterministically ordered,
// ranked results.
//
// documents is the caller's submitted document list (providing ID-by-index).
// topN is the requested response cardinality. raw is the provider's
// index/score pairs in provider response order.
//
// Validation rejects:
//   - response count not equal to topN;
//   - missing index or score (both must be present even when zero);
//   - index outside the submitted document array;
//   - duplicate response indices;
//   - NaN or infinite score.
//
// After validation, results are sorted:
//  1. score descending;
//  2. input index ascending for equal scores;
//  3. document ID ascending as a final deterministic tie break.
//
// Ranks are assigned from 1 after sorting. Response order is never treated as
// durable identity; only the mapped DocumentID is.
func ValidateAndMapResults(documents []Document, topN int, raw []RawResult) ([]Result, error) {
	if len(raw) != topN {
		return nil, fmt.Errorf("rerank response returned %d results, expected %d: %w",
			len(raw), topN, ErrInvalidResponse)
	}
	if topN == 0 {
		return nil, fmt.Errorf("rerank response top_n must be >= 1: %w", ErrInvalidResponse)
	}

	seen := make(map[int]struct{}, len(raw))
	results := make([]Result, 0, len(raw))
	for i, item := range raw {
		if !item.HasIndex {
			return nil, fmt.Errorf("rerank response result %d is missing index: %w", i, ErrInvalidResponse)
		}
		if !item.HasScore {
			return nil, fmt.Errorf("rerank response result %d (index %d) is missing relevance_score: %w",
				i, item.Index, ErrInvalidResponse)
		}
		if item.Index < 0 || item.Index >= len(documents) {
			return nil, fmt.Errorf("rerank response index %d is outside the submitted documents: %w",
				item.Index, ErrInvalidResponse)
		}
		if _, exists := seen[item.Index]; exists {
			return nil, fmt.Errorf("rerank response index %d appears more than once: %w",
				item.Index, ErrInvalidResponse)
		}
		if math.IsNaN(item.Score) || math.IsInf(item.Score, 0) {
			return nil, fmt.Errorf("rerank response index %d has non-finite relevance score: %w",
				item.Index, ErrInvalidResponse)
		}
		seen[item.Index] = struct{}{}
		results = append(results, Result{
			DocumentID: documents[item.Index].ID,
			Index:      item.Index,
			Score:      item.Score,
		})
	}

	SortResults(results)
	AssignRanks(results)
	return results, nil
}

// RawResult is a provider-returned index/score pair before mapping and
// validation. HasIndex and HasScore distinguish a missing field from a valid
// zero value (pointer-like semantics without pointers).
type RawResult struct {
	Index    int
	Score    float64
	HasIndex bool
	HasScore bool
}

// SortResults sorts results in place deterministically:
//  1. score descending;
//  2. input index ascending for equal scores;
//  3. document ID ascending as a final tie break.
//
// It is stable with respect to the declared tie-breakers.
func SortResults(results []Result) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Index != results[j].Index {
			return results[i].Index < results[j].Index
		}
		return results[i].DocumentID < results[j].DocumentID
	})
}

// AssignRanks assigns Rank = position+1 to each result in slice order. It
// assumes the slice is already sorted by SortResults.
func AssignRanks(results []Result) {
	for i := range results {
		results[i].Rank = i + 1
	}
}
