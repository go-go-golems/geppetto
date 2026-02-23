package gepa

import (
	"math"
	"sort"
)

// Dominates returns true if a dominates b (higher is better).
// A dominates B if, for all objectives, A >= B, and for at least one, A > B.
// Missing objectives are treated as -Inf (so they generally lose).
func Dominates(a, b ObjectiveScores) bool {
	if len(a) == 0 && len(b) == 0 {
		return false
	}
	allKeys := map[string]struct{}{}
	for k := range a {
		allKeys[k] = struct{}{}
	}
	for k := range b {
		allKeys[k] = struct{}{}
	}

	strict := false
	for k := range allKeys {
		av, aok := a[k]
		bv, bok := b[k]
		if !aok {
			av = math.Inf(-1)
		}
		if !bok {
			bv = math.Inf(-1)
		}
		if av < bv {
			return false
		}
		if av > bv {
			strict = true
		}
	}
	return strict
}

// ParetoFront returns the indices of the non-dominated points.
// Complexity is O(n^2) which is fine for small candidate pools.
func ParetoFront(points []ObjectiveScores) []int {
	n := len(points)
	if n == 0 {
		return nil
	}
	nd := make([]bool, n)
	for i := range nd {
		nd[i] = true
	}
	for i := 0; i < n; i++ {
		if !nd[i] {
			continue
		}
		for j := 0; j < n; j++ {
			if i == j || !nd[i] {
				continue
			}
			if Dominates(points[j], points[i]) {
				nd[i] = false
				break
			}
		}
	}
	out := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if nd[i] {
			out = append(out, i)
		}
	}
	return out
}

// TopKByScore returns the indices of the top-k items by score (descending).
func TopKByScore(scores []float64, k int) []int {
	if k <= 0 {
		return nil
	}
	type pair struct {
		I int
		S float64
	}
	ps := make([]pair, 0, len(scores))
	for i, s := range scores {
		ps = append(ps, pair{I: i, S: s})
	}
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].S > ps[j].S
	})
	if k > len(ps) {
		k = len(ps)
	}
	out := make([]int, 0, k)
	for i := 0; i < k; i++ {
		out = append(out, ps[i].I)
	}
	return out
}
