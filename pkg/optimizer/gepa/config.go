package gepa

import "time"

// Config controls the GEPA-style optimization loop.
type Config struct {
	// Maximum number of evaluator calls. In prompt optimization, a “rollout”
	// usually corresponds to one (candidate, example) evaluation.
	MaxEvalCalls int

	// BatchSize is the number of examples sampled per iteration for mutation testing.
	BatchSize int

	// FrontierSize caps the number of candidates kept on the Pareto frontier
	// in the single-objective fallback path.
	FrontierSize int

	// RandomSeed controls stochastic selection and batching.
	// If 0, the optimizer will use time-based entropy.
	RandomSeed int64

	// ReflectionSystemPrompt is the system prompt used for the reflection LLM.
	ReflectionSystemPrompt string

	// ReflectionPromptTemplate must include "<curr_param>" and "<side_info>" placeholders.
	ReflectionPromptTemplate string

	// Objective is an optional natural-language description of what we are optimizing for.
	Objective string

	// MaxSideInfoChars caps the amount of formatted side-info passed to the reflector.
	// 0 means “no explicit cap”.
	MaxSideInfoChars int

	// Epsilon is the minimum improvement required to accept a child over its parent
	// in the single-objective setting. 0 is fine for most use.
	Epsilon float64

	// Now is injectable for tests. If nil, time.Now is used.
	Now func() time.Time
}

func (c Config) withDefaults() Config {
	out := c
	if out.MaxEvalCalls <= 0 {
		out.MaxEvalCalls = 200
	}
	if out.BatchSize <= 0 {
		out.BatchSize = 8
	}
	if out.FrontierSize <= 0 {
		out.FrontierSize = 10
	}
	if out.RandomSeed == 0 {
		out.RandomSeed = time.Now().UnixNano()
	}
	if out.ReflectionSystemPrompt == "" {
		out.ReflectionSystemPrompt = "You are an expert prompt engineer."
	}
	if out.ReflectionPromptTemplate == "" {
		out.ReflectionPromptTemplate = DefaultReflectionPromptTemplate
	}
	if out.Now == nil {
		out.Now = time.Now
	}
	return out
}
