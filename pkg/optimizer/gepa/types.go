package gepa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// Candidate is a mapping of optimizable text parameters.
// For simple prompt optimization, this often has a single key (e.g., "prompt").
type Candidate map[string]string

// CandidateID is an internal identifier for a candidate.
type CandidateID int

// ObjectiveScores is a multi-objective score vector (higher is better on every dimension).
type ObjectiveScores map[string]float64

// EvalResult is the per-example evaluation output returned by the evaluator.
type EvalResult struct {
	Score          float64         `json:"score"`
	Objectives     ObjectiveScores `json:"objectives,omitempty"`
	Output         any             `json:"output,omitempty"`
	Feedback       any             `json:"feedback,omitempty"`
	Trace          any             `json:"trace,omitempty"`
	Raw            any             `json:"raw,omitempty"` // full raw return value for debugging
	EvaluatorNotes string          `json:"evaluator_notes,omitempty"`
}

// ExampleEval ties an evaluation to a specific example index.
type ExampleEval struct {
	ExampleIndex int        `json:"example_index"`
	Result       EvalResult `json:"result"`
}

// CandidateStats tracks aggregate stats computed over a subset of examples.
type CandidateStats struct {
	MeanScore      float64         `json:"mean_score"`
	MeanObjectives ObjectiveScores `json:"mean_objectives,omitempty"`
	N              int             `json:"n"`
}

// candidateHash is a deterministic hash of a candidate mapping.
func candidateHash(c Candidate) string {
	if c == nil {
		return ""
	}
	// Sort keys for stable JSON.
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	type kv struct {
		K string `json:"k"`
		V string `json:"v"`
	}
	pairs := make([]kv, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, kv{K: k, V: c[k]})
	}
	b, _ := json.Marshal(pairs)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
