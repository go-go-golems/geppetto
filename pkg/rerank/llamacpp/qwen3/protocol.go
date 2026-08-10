package qwen3

// completionRequest is the exact llama.cpp /completion request used by the
// Qwen3-Reranker scoring contract.
type completionRequest struct {
	Prompt      string  `json:"prompt"`
	NPredict    int     `json:"n_predict"`
	NProbs      int     `json:"n_probs"`
	Temperature float64 `json:"temperature"`
}

type completionResponse struct {
	CompletionProbabilities []completionProbability `json:"completion_probabilities"`
	TokensEvaluated         *int                    `json:"tokens_evaluated,omitempty"`
	TokensPredicted         *int                    `json:"tokens_predicted,omitempty"`
}

type completionProbability struct {
	TopLogprobs []tokenLogprob `json:"top_logprobs"`
}

type tokenLogprob struct {
	Token   string   `json:"token"`
	Logprob *float64 `json:"logprob"`
}
