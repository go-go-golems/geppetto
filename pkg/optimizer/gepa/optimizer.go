package gepa

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// EvaluateFunc evaluates a candidate on a single example.
// The exampleIndex is provided for caching and traceability.
type EvaluateFunc func(ctx context.Context, candidate Candidate, exampleIndex int, example any) (EvalResult, error)

// Optimizer runs a GEPA-style reflective evolutionary loop.
//
// This is intentionally “GEPA-inspired” rather than a 1:1 port of the Python reference.
// The goal is a small, composable core that can sit on top of Geppetto and JS evaluators.
type Optimizer struct {
	cfg       Config
	eval      EvaluateFunc
	reflector *Reflector

	rng       *rand.Rand
	cache     map[string]map[int]EvalResult // candidateHash -> exampleIndex -> result
	callsUsed int

	// pool is append-only.
	pool []*candidateNode
}

type candidateNode struct {
	ID        CandidateID
	ParentID  CandidateID
	Candidate Candidate
	CreatedAt time.Time

	// Evaluations is the subset of cached evaluations we’ve pulled into the node.
	// The optimizer cache is the source of truth; this field is a convenience view.
	Evaluations map[int]EvalResult

	// ReflectionRaw is the full raw LLM response from the last mutation step (debug).
	ReflectionRaw string
}

type Result struct {
	BestCandidate Candidate        `json:"best_candidate"`
	BestStats     CandidateStats   `json:"best_stats"`
	CallsUsed     int              `json:"calls_used"`
	Candidates    []CandidateEntry `json:"candidates"`
}

type CandidateEntry struct {
	ID            int            `json:"id"`
	ParentID      int            `json:"parent_id"`
	Hash          string         `json:"hash"`
	CreatedAt     time.Time      `json:"created_at"`
	Candidate     Candidate      `json:"candidate"`
	GlobalStats   CandidateStats `json:"global_stats"`
	EvalsCached   int            `json:"evals_cached"`
	ReflectionRaw string         `json:"reflection_raw,omitempty"`
}

// NewOptimizer constructs an optimizer.
func NewOptimizer(cfg Config, eval EvaluateFunc, reflector *Reflector) *Optimizer {
	c := cfg.withDefaults()
	r := rand.New(rand.NewSource(c.RandomSeed))
	return &Optimizer{
		cfg:       c,
		eval:      eval,
		reflector: reflector,
		rng:       r,
		cache:     map[string]map[int]EvalResult{},
	}
}

// CallsUsed returns the number of evaluator calls consumed so far.
func (o *Optimizer) CallsUsed() int {
	if o == nil {
		return 0
	}
	return o.callsUsed
}

func (o *Optimizer) Optimize(ctx context.Context, seed Candidate, examples []any) (*Result, error) {
	if o == nil {
		return nil, fmt.Errorf("optimizer is nil")
	}
	if o.eval == nil {
		return nil, fmt.Errorf("optimizer: evaluator is nil")
	}
	if o.reflector == nil || o.reflector.Engine == nil {
		return nil, fmt.Errorf("optimizer: reflector is nil")
	}
	if len(seed) == 0 {
		return nil, fmt.Errorf("optimizer: seed candidate is empty")
	}
	if len(examples) == 0 {
		return nil, fmt.Errorf("optimizer: dataset is empty")
	}

	// Initialize pool with seed.
	seedNode := &candidateNode{
		ID:          0,
		ParentID:    -1,
		Candidate:   cloneCandidate(seed),
		CreatedAt:   o.cfg.Now(),
		Evaluations: map[int]EvalResult{},
	}
	o.pool = append(o.pool, seedNode)

	// Evaluate seed on an initial batch (to get some ASI).
	initIdx := o.sampleBatchIndices(len(examples), o.cfg.BatchSize, o.remainingBudget())
	if len(initIdx) == 0 {
		return nil, fmt.Errorf("optimizer: insufficient budget to evaluate seed")
	}
	if _, err := o.ensureEvaluated(ctx, seedNode, examples, initIdx); err != nil {
		return nil, err
	}

	bestNode := seedNode

	for o.callsUsed < o.cfg.MaxEvalCalls {
		callsAtIterStart := o.callsUsed
		remaining := o.remainingBudget()
		if remaining <= 0 {
			break
		}

		parent := o.selectParent()
		if parent == nil {
			break
		}

		// Plan evaluations for parent/child within remaining budget.
		// Worst case: we need to evaluate both parent and child on the same minibatch.
		// Parent may already be cached on some indices, so we conservatively size based on remaining.
		batchSize := o.cfg.BatchSize
		if batchSize*2 > remaining {
			batchSize = remaining / 2
		}
		if batchSize <= 0 {
			break
		}

		batchIdx := o.sampleBatchIndices(len(examples), batchSize, remaining)
		if len(batchIdx) == 0 {
			break
		}

		parentEvals, err := o.ensureEvaluated(ctx, parent, examples, batchIdx)
		if err != nil {
			return nil, err
		}

		sideInfo := FormatSideInfo(examples, parentEvals, o.cfg.MaxSideInfoChars)

		// Mutate candidate (currently: optimize the "prompt" key if present; else first key).
		paramKey := primaryParamKey(parent.Candidate)
		current := parent.Candidate[paramKey]

		childText, rawReflection, err := o.reflector.Propose(ctx, current, sideInfo)
		if err != nil {
			return nil, err
		}
		childCand := cloneCandidate(parent.Candidate)
		childCand[paramKey] = childText

		childNode := &candidateNode{
			ID:            CandidateID(len(o.pool)),
			ParentID:      parent.ID,
			Candidate:     childCand,
			CreatedAt:     o.cfg.Now(),
			Evaluations:   map[int]EvalResult{},
			ReflectionRaw: rawReflection,
		}

		childEvals, err := o.ensureEvaluated(ctx, childNode, examples, batchIdx)
		if err != nil {
			return nil, err
		}

		parentStats := AggregateStats(parentEvals)
		childStats := AggregateStats(childEvals)

		accepted := o.acceptChild(parentStats, childStats)
		if accepted {
			o.pool = append(o.pool, childNode)
			// Update best based on global stats available so far.
			if childGlobal := o.globalStats(childNode); childGlobal.MeanScore > o.globalStats(bestNode).MeanScore {
				bestNode = childNode
			}
		}

		// Guard against stagnation: when all parent/child evals come from cache
		// and no candidate is accepted, the loop would otherwise spin forever.
		if o.callsUsed == callsAtIterStart && !accepted {
			break
		}
	}

	bestStats := o.globalStats(bestNode)

	entries := make([]CandidateEntry, 0, len(o.pool))
	for _, n := range o.pool {
		entries = append(entries, CandidateEntry{
			ID:            int(n.ID),
			ParentID:      int(n.ParentID),
			Hash:          candidateHash(n.Candidate),
			CreatedAt:     n.CreatedAt,
			Candidate:     cloneCandidate(n.Candidate),
			GlobalStats:   o.globalStats(n),
			EvalsCached:   len(o.cache[candidateHash(n.Candidate)]),
			ReflectionRaw: n.ReflectionRaw,
		})
	}

	return &Result{
		BestCandidate: cloneCandidate(bestNode.Candidate),
		BestStats:     bestStats,
		CallsUsed:     o.callsUsed,
		Candidates:    entries,
	}, nil
}

func (o *Optimizer) remainingBudget() int {
	return o.cfg.MaxEvalCalls - o.callsUsed
}

func (o *Optimizer) acceptChild(parent, child CandidateStats) bool {
	// Multi-objective: accept if child dominates parent.
	if len(parent.MeanObjectives) > 1 || len(child.MeanObjectives) > 1 {
		if Dominates(child.MeanObjectives, parent.MeanObjectives) {
			return true
		}
		// Fallback: accept if scalar score improved.
	}

	return child.MeanScore > parent.MeanScore+o.cfg.Epsilon
}

func (o *Optimizer) selectParent() *candidateNode {
	if len(o.pool) == 0 {
		return nil
	}

	// Build objective vectors for selection.
	obj := make([]ObjectiveScores, 0, len(o.pool))
	scalars := make([]float64, 0, len(o.pool))
	for _, n := range o.pool {
		stats := o.globalStats(n)
		vec := stats.MeanObjectives
		if len(vec) == 0 {
			vec = ObjectiveScores{"score": stats.MeanScore}
		}
		obj = append(obj, vec)
		scalars = append(scalars, stats.MeanScore)
	}

	// If we have multiple objectives overall, use Pareto front.
	keys := unionObjectiveKeys(obj)
	var candIdx []int
	if len(keys) > 1 {
		candIdx = ParetoFront(obj)
	} else {
		candIdx = TopKByScore(scalars, o.cfg.FrontierSize)
	}

	if len(candIdx) == 0 {
		// Fallback to uniform.
		return o.pool[o.rng.Intn(len(o.pool))]
	}

	// Weighted random selection by scalar score (shifted to positive).
	minS := math.Inf(1)
	for _, i := range candIdx {
		if scalars[i] < minS {
			minS = scalars[i]
		}
	}
	weights := make([]float64, 0, len(candIdx))
	sum := 0.0
	for _, i := range candIdx {
		w := scalars[i] - minS + 1e-9
		if w < 0 {
			w = 0
		}
		weights = append(weights, w)
		sum += w
	}
	var chosen int
	if sum <= 0 {
		chosen = candIdx[o.rng.Intn(len(candIdx))]
	} else {
		r := o.rng.Float64() * sum
		acc := 0.0
		for j, i := range candIdx {
			acc += weights[j]
			if r <= acc {
				chosen = i
				break
			}
		}
	}
	return o.pool[chosen]
}

func unionObjectiveKeys(vecs []ObjectiveScores) map[string]struct{} {
	out := map[string]struct{}{}
	for _, v := range vecs {
		for k := range v {
			out[k] = struct{}{}
		}
	}
	return out
}

func (o *Optimizer) sampleBatchIndices(n, batchSize, budget int) []int {
	if n <= 0 || batchSize <= 0 || budget <= 0 {
		return nil
	}
	if batchSize > n {
		batchSize = n
	}
	if batchSize > budget {
		batchSize = budget
	}
	if batchSize <= 0 {
		return nil
	}
	if batchSize == n {
		out := make([]int, n)
		for i := 0; i < n; i++ {
			out[i] = i
		}
		return out
	}

	// Sample without replacement.
	perm := o.rng.Perm(n)
	return append([]int(nil), perm[:batchSize]...)
}

func (o *Optimizer) ensureEvaluated(ctx context.Context, n *candidateNode, examples []any, indices []int) ([]ExampleEval, error) {
	if n == nil {
		return nil, fmt.Errorf("ensureEvaluated: node is nil")
	}
	h := candidateHash(n.Candidate)
	if h == "" {
		return nil, fmt.Errorf("ensureEvaluated: candidate hash is empty")
	}
	if _, ok := o.cache[h]; !ok {
		o.cache[h] = map[int]EvalResult{}
	}

	out := make([]ExampleEval, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= len(examples) {
			continue
		}
		if cached, ok := o.cache[h][idx]; ok {
			n.Evaluations[idx] = cached
			out = append(out, ExampleEval{ExampleIndex: idx, Result: cached})
			continue
		}
		if o.callsUsed >= o.cfg.MaxEvalCalls {
			break
		}
		res, err := o.eval(ctx, n.Candidate, idx, examples[idx])
		if err != nil {
			return nil, fmt.Errorf("evaluator failed for example %d: %w", idx, err)
		}
		// Ensure objectives has a "score" dimension if none were provided.
		if len(res.Objectives) == 0 {
			res.Objectives = ObjectiveScores{"score": res.Score}
		}
		o.cache[h][idx] = res
		n.Evaluations[idx] = res
		o.callsUsed++
		out = append(out, ExampleEval{ExampleIndex: idx, Result: res})
	}
	return out, nil
}

func (o *Optimizer) globalStats(n *candidateNode) CandidateStats {
	if n == nil {
		return CandidateStats{}
	}
	evals := make([]ExampleEval, 0, len(n.Evaluations))
	for idx, res := range n.Evaluations {
		evals = append(evals, ExampleEval{ExampleIndex: idx, Result: res})
	}
	return AggregateStats(evals)
}

// AggregateStats computes mean score + mean objectives for a slice of evaluations.
func AggregateStats(evals []ExampleEval) CandidateStats {
	if len(evals) == 0 {
		return CandidateStats{}
	}
	sumScore := 0.0
	count := 0

	sumObj := map[string]float64{}
	cntObj := map[string]int{}

	for _, e := range evals {
		sumScore += e.Result.Score
		count++

		vec := e.Result.Objectives
		if len(vec) == 0 {
			vec = ObjectiveScores{"score": e.Result.Score}
		}
		for k, v := range vec {
			sumObj[k] += v
			cntObj[k]++
		}
	}

	meanObj := ObjectiveScores{}
	for k, s := range sumObj {
		if cntObj[k] > 0 {
			meanObj[k] = s / float64(cntObj[k])
		}
	}

	return CandidateStats{
		MeanScore:      sumScore / float64(count),
		MeanObjectives: meanObj,
		N:              count,
	}
}

func cloneCandidate(c Candidate) Candidate {
	if c == nil {
		return nil
	}
	out := make(Candidate, len(c))
	for k, v := range c {
		out[k] = v
	}
	return out
}

func primaryParamKey(c Candidate) string {
	if c == nil {
		return "prompt"
	}
	if _, ok := c["prompt"]; ok {
		return "prompt"
	}
	// deterministic fallback: smallest key.
	var best string
	for k := range c {
		if best == "" || k < best {
			best = k
		}
	}
	if best == "" {
		return "prompt"
	}
	return best
}
