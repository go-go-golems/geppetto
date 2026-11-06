package systemone

import "encoding/json"

// wireRequest is the POST /v1/systemone body. The gateway rejects unknown
// top-level fields, so nothing beyond these four is ever sent.
type wireRequest struct {
	Model     string                  `json:"model"`
	State     any                     `json:"state"`
	Images    []string                `json:"images,omitempty"`
	Questions map[string]wireQuestion `json:"questions"`
}

// wireQuestion carries the type-specific criteria as raw JSON so a choice's
// option object keeps the caller's order (encoding/json sorts map keys).
type wireQuestion struct {
	Type         string          `json:"type"`
	Instructions any             `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria,omitempty"`
}

// wireResponse is the 200 body. Unknown top-level fields are tolerated: the
// protocol documents fields (latency_ms) that the gateway does not send yet.
type wireResponse struct {
	Model     string                `json:"model"`
	Answers   map[string]wireAnswer `json:"answers"`
	Usage     *wireUsage            `json:"usage"`
	LatencyMs *int64                `json:"latency_ms"`
}

type wireUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// wireAnswer uses pointers so a missing field is distinguishable from zero.
type wireAnswer struct {
	Type          string              `json:"type"`
	Noul          *float64            `json:"noul"`
	Choice        *string             `json:"choice"`
	Score         *float64            `json:"score"`
	Confidence    *float64            `json:"confidence"`
	Probabilities map[string]*float64 `json:"probabilities"` // nil entry = JSON null, rejected
	Legend        map[string]string   `json:"legend"`
}

// wireError is the error envelope: {"error": {code, message, request_id}}.
type wireError struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}
