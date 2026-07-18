package llamacpp

// request is the llama.cpp /v1/rerank request wire DTO.
//
// Caller document IDs never enter the provider payload. The adapter retains a
// local index-to-ID table and submits only document text in array order.
type request struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n"`
}

// response is the llama.cpp /v1/rerank response wire DTO.
//
// Pointer fields distinguish a missing zero from a valid zero. The adapter
// rejects responses where any result is missing index or relevance_score.
type response struct {
	Model   string `json:"model,omitempty"`
	Object  string `json:"object,omitempty"`
	Usage   *usage `json:"usage,omitempty"`
	Results []item `json:"results"`
}

// usage mirrors the llama.cpp prompt/total token usage object.
type usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// item is one scored result. Pointer fields distinguish missing from zero.
type item struct {
	Index          *int     `json:"index"`
	RelevanceScore *float64 `json:"relevance_score"`
}
