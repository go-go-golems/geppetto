package rerank

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRequest_JSONRoundTrip(t *testing.T) {
	in := Request{
		Model: "m",
		Query: "q",
		Documents: []Document{
			{ID: "a", Text: "x"},
			{ID: "b", Text: "y"},
		},
		TopN: 2,
	}
	b, err := json.Marshal(in)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"m","query":"q","documents":[{"id":"a","text":"x"},{"id":"b","text":"y"}],"top_n":2}`, string(b))

	var out Request
	require.NoError(t, json.Unmarshal(b, &out))
	assert.Equal(t, in, out)
}

func TestRequest_YAMLRoundTrip(t *testing.T) {
	in := Request{
		Model: "m",
		Query: "q",
		Documents: []Document{
			{ID: "a", Text: "x"},
		},
		TopN: 1,
	}
	b, err := yaml.Marshal(in)
	require.NoError(t, err)
	var out Request
	require.NoError(t, yaml.Unmarshal(b, &out))
	assert.Equal(t, in, out)
}

func TestResponse_JSONRoundTrip_OmitsEmptyOptionals(t *testing.T) {
	resp := Response{
		Provider: "llama.cpp",
		Model:    "m",
		Results: []Result{
			{DocumentID: "a", Index: 0, Score: -3.3, Rank: 1},
		},
		// Usage, Cost, RequestID, DurationMs intentionally nil.
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.JSONEq(t, `{"provider":"llama.cpp","model":"m","results":[{"document_id":"a","index":0,"score":-3.3,"rank":1}]}`, string(b))
}

func TestResponse_JSONRoundTrip_WithOptionals(t *testing.T) {
	cost := 0.0
	dur := int64(42)
	resp := Response{
		Provider:   "llama.cpp",
		Model:      "m",
		Results:    []Result{{DocumentID: "a", Index: 0, Score: 1.0, Rank: 1}},
		Usage:      &Usage{InputTokens: 96, TotalTokens: 96},
		Cost:       &cost,
		RequestID:  "req-1",
		DurationMs: &dur,
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	var out Response
	require.NoError(t, json.Unmarshal(b, &out))
	assert.Equal(t, resp, out)
	// Distinguish nil from zero cost: round-tripped Cost is a non-nil pointer to 0.
	require.NotNil(t, out.Cost)
	assert.Equal(t, 0.0, *out.Cost)
}

func TestUsage_OmitsZeroTokens(t *testing.T) {
	b, err := json.Marshal(Usage{})
	require.NoError(t, err)
	// omitempty drops both zero fields.
	assert.Equal(t, `{}`, string(b))
}

func TestModel_JSONRoundTrip(t *testing.T) {
	m := Model{Provider: "llama.cpp", Name: "bge"}
	b, err := json.Marshal(m)
	require.NoError(t, err)
	assert.JSONEq(t, `{"provider":"llama.cpp","name":"bge"}`, string(b))
}
