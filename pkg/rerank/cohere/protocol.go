package cohere

// request is the Cohere v2 /rerank request wire DTO.
//
// Caller document IDs never enter the provider payload. The adapter retains a
// local index-to-ID table and submits only document text in array order.
//
// max_tokens_per_doc is intentionally not exposed: the core rerank.Request is
// transport-neutral and callers pre-truncate document text
// (GEPPETTO-RERANKER-002 decision record DR-4).
type request struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n"`
}

// response is the Cohere v2 /rerank response wire DTO.
//
// Pointer fields distinguish a missing zero from a valid zero. The adapter
// rejects responses where any result is missing index or relevance_score.
type response struct {
	ID      string `json:"id,omitempty"`
	Results []item `json:"results"`
	Meta    *meta  `json:"meta,omitempty"`
}

// item is one scored result. Pointer fields distinguish missing from zero.
type item struct {
	Index          *int     `json:"index"`
	RelevanceScore *float64 `json:"relevance_score"`
}

// meta mirrors the Cohere response metadata object. Only the fields the
// adapter consumes are modeled; strict decoding makes any other evolution of
// this object an explicit, test-visible change.
type meta struct {
	APIVersion  *apiVersion  `json:"api_version,omitempty"`
	BilledUnits *billedUnits `json:"billed_units,omitempty"`
}

// apiVersion mirrors the Cohere API version object.
type apiVersion struct {
	Version        string `json:"version"`
	IsExperimental bool   `json:"is_experimental"`
}

// billedUnits mirrors the Cohere billed units object. Rerank calls bill in
// search units, not tokens.
type billedUnits struct {
	SearchUnits int `json:"search_units"`
}
