package decision

import "context"

// Type is a question type.
type Type string

const (
	// Noul asks for the probability that a proposition is true.
	Noul Type = "noul"
	// Choice asks for one of 1–255 named options.
	Choice Type = "choice"
	// Score asks for a level on an ordered rubric of 1–10 levels.
	Score Type = "score"
)

// Limits shared by every provider of this question model.
const (
	MaxOptions    = 255
	MaxLevels     = 10
	MaxQuestionID = 64
)

// Option is one choice option. Order is significant: adapters send options in
// slice order, because decision models are sensitive to option order.
type Option struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Question is one typed question about the request's state.
type Question struct {
	Type Type `json:"type" yaml:"type"`
	// Instructions says what to decide: a non-empty string, or JSON-encodable
	// structured data (map or slice) that adapters send as-is.
	Instructions any `json:"instructions" yaml:"instructions"`
	// True and False optionally describe the two cases of a Noul question.
	True  string `json:"true,omitempty" yaml:"true,omitempty"`
	False string `json:"false,omitempty" yaml:"false,omitempty"`
	// Options are a Choice question's options, in presentation order.
	Options []Option `json:"options,omitempty" yaml:"options,omitempty"`
	// Levels are a Score question's level descriptions, level 0 first.
	Levels []string `json:"levels,omitempty" yaml:"levels,omitempty"`
}

// Request asks a set of questions about one state. All questions share the
// state, which providers count once; batching questions is cheaper than
// separate requests.
//
// Model is either empty (the provider's configured model is used) or equal
// to the configured model; a mismatch is rejected rather than silently
// served by another model.
type Request struct {
	Model string `json:"model,omitempty" yaml:"model,omitempty"`
	// State is what the questions are evaluated against: a non-empty string,
	// or JSON-encodable structured data (map or slice).
	State any `json:"state" yaml:"state"`
	// Images are optional state images as data URLs
	// (data:image/{jpeg,png,webp};base64,...), for vision-capable models.
	Images []string `json:"images,omitempty" yaml:"images,omitempty"`
	// Questions are keyed by caller-chosen ids, echoed in Response.Answers.
	Questions map[string]Question `json:"questions" yaml:"questions"`
}

// Answer is one question's answer. Which fields are set depends on Type.
type Answer struct {
	Type Type `json:"type" yaml:"type"`
	// Probability (Noul) that the proposition is true.
	Probability float64 `json:"probability,omitempty" yaml:"probability,omitempty"`
	// Choice is the most likely option (Choice).
	Choice string `json:"choice,omitempty" yaml:"choice,omitempty"`
	// Probabilities over option names (Choice).
	Probabilities map[string]float64 `json:"probabilities,omitempty" yaml:"probabilities,omitempty"`
	// Score is the fractional level in [0, len(Levels)-1] (Score).
	Score float64 `json:"score,omitempty" yaml:"score,omitempty"`
	// LevelProbabilities[i] is the probability of level i (Score).
	LevelProbabilities []float64 `json:"level_probabilities,omitempty" yaml:"level_probabilities,omitempty"`
	// Confidence (Choice, Score) measures how peaked the distribution is, in
	// [0, 1]. It is derived from the distribution, not predicted separately.
	Confidence float64 `json:"confidence,omitempty" yaml:"confidence,omitempty"`
}

// Usage is provider-reported token consumption. A nil *Usage means the
// provider reported none.
type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty" yaml:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty" yaml:"output_tokens,omitempty"`
}

// Response carries the answers plus provider observations. Cost is nil when
// pricing is unknown and points at 0 when the call is known to be free.
type Response struct {
	Provider   string            `json:"provider" yaml:"provider"`
	Model      string            `json:"model" yaml:"model"`
	Answers    map[string]Answer `json:"answers" yaml:"answers"`
	Usage      *Usage            `json:"usage,omitempty" yaml:"usage,omitempty"`
	Cost       *float64          `json:"cost,omitempty" yaml:"cost,omitempty"`
	RequestID  string            `json:"request_id,omitempty" yaml:"request_id,omitempty"`
	DurationMs *int64            `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty"`
}

// Model identifies a provider instance's configured model.
type Model struct {
	Provider string `json:"provider" yaml:"provider"`
	Name     string `json:"name" yaml:"name"`
}

// Provider is the transport-neutral decision provider interface.
//
// Implementations must validate the request (ValidateRequest), send choice
// options in slice order, validate the answers (ValidateResponse), and
// preserve usage, model, request id and duration where available, leaving
// unknown values nil rather than fabricating zero.
type Provider interface {
	Decide(ctx context.Context, in Request) (Response, error)
	Model() Model
}

// ResolveModel returns the model a request should be served by: the
// request's model when set, otherwise the provider's.
func ResolveModel(in Request, configured Model) string {
	if in.Model != "" {
		return in.Model
	}
	return configured.Name
}
