package rerank

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProvider is a deterministic, in-process Provider used to exercise the
// core response mapping, ordering, and error categories without any HTTP.
type fakeProvider struct {
	model    Model
	raw      []RawResult
	rawErr   error
	requests []Request
}

func (f *fakeProvider) Rerank(_ context.Context, in Request) (Response, error) {
	f.requests = append(f.requests, in)
	if f.rawErr != nil {
		return Response{}, f.rawErr
	}
	if err := ValidateRequest(in, f.model); err != nil {
		return Response{}, err
	}
	results, err := ValidateAndMapResults(in.Documents, in.TopN, f.raw)
	if err != nil {
		return Response{}, err
	}
	return Response{
		Provider: f.model.Provider,
		Model:    f.model.Name,
		Results:  results,
	}, nil
}

func (f *fakeProvider) Model() Model { return f.model }

func TestValidateRequest_RejectsEmptyQuery(t *testing.T) {
	err := ValidateRequest(Request{
		Query:     "   ",
		Documents: []Document{{ID: "a", Text: "x"}},
		TopN:      1,
	}, Model{})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Contains(t, err.Error(), "query is required")
}

func TestValidateRequest_RejectsZeroDocuments(t *testing.T) {
	err := ValidateRequest(Request{
		Query:     "q",
		Documents: nil,
		TopN:      1,
	}, Model{})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Contains(t, err.Error(), "at least one document")
}

func TestValidateRequest_RejectsEmptyAndDuplicateIDs(t *testing.T) {
	err := ValidateRequest(Request{
		Query:     "q",
		Documents: []Document{{ID: "", Text: "x"}},
		TopN:      1,
	}, Model{})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Contains(t, err.Error(), "non-empty id")

	err = ValidateRequest(Request{
		Query:     "q",
		Documents: []Document{{ID: "a", Text: "x"}, {ID: "a", Text: "y"}},
		TopN:      2,
	}, Model{})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Contains(t, err.Error(), "duplicate id")
}

func TestValidateRequest_RejectsEmptyText(t *testing.T) {
	err := ValidateRequest(Request{
		Query:     "q",
		Documents: []Document{{ID: "CALLER-ID-MUST-NOT-LEAK", Text: "  "}},
		TopN:      1,
	}, Model{})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Contains(t, err.Error(), "non-empty text")
	assert.NotContains(t, err.Error(), "CALLER-ID-MUST-NOT-LEAK")
}

func TestValidateRequest_DuplicateIDDoesNotLeakProtectedValue(t *testing.T) {
	err := ValidateRequest(Request{
		Query: "q",
		Documents: []Document{
			{ID: "CALLER-ID-MUST-NOT-LEAK", Text: "x"},
			{ID: "CALLER-ID-MUST-NOT-LEAK", Text: "y"},
		},
		TopN: 2,
	}, Model{})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.NotContains(t, err.Error(), "CALLER-ID-MUST-NOT-LEAK")
}

func TestValidateRequest_RejectsInvalidTopN(t *testing.T) {
	err := ValidateRequest(Request{
		Query:     "q",
		Documents: []Document{{ID: "a", Text: "x"}},
		TopN:      0,
	}, Model{})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Contains(t, err.Error(), "top_n must be >= 1")

	err = ValidateRequest(Request{
		Query:     "q",
		Documents: []Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}},
		TopN:      3,
	}, Model{})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Contains(t, err.Error(), "must be <= number of documents")
}

func TestValidateRequest_RejectsModelMismatch(t *testing.T) {
	err := ValidateRequest(Request{
		Model:     "wrong-model",
		Query:     "q",
		Documents: []Document{{ID: "a", Text: "x"}},
		TopN:      1,
	}, Model{Name: "configured-model"})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Contains(t, err.Error(), "does not match provider model")
}

func TestValidateRequest_AllowsEmptyRequestModel(t *testing.T) {
	err := ValidateRequest(Request{
		Model:     "",
		Query:     "q",
		Documents: []Document{{ID: "a", Text: "x"}},
		TopN:      1,
	}, Model{Name: "configured-model"})
	require.NoError(t, err)
}

func TestResolveModel_PrefersRequestModel(t *testing.T) {
	assert.Equal(t, "req-model",
		ResolveModel(Request{Model: "req-model"}, Model{Name: "provider-model"}))
	assert.Equal(t, "provider-model",
		ResolveModel(Request{Model: ""}, Model{Name: "provider-model"}))
}

func TestValidateAndMapResults_MapsIndexToDocumentID(t *testing.T) {
	docs := []Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}, {ID: "c", Text: "z"}}
	raw := []RawResult{
		{Index: 2, Score: -11.0, HasIndex: true, HasScore: true},
		{Index: 0, Score: -3.3, HasIndex: true, HasScore: true},
		{Index: 1, Score: -9.8, HasIndex: true, HasScore: true},
	}
	results, err := ValidateAndMapResults(docs, 3, raw)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Descending score: -3.3 (a, idx 0), -9.8 (b, idx 1), -11.0 (c, idx 2).
	assert.Equal(t, "a", results[0].DocumentID)
	assert.Equal(t, -3.3, results[0].Score)
	assert.Equal(t, 1, results[0].Rank)
	assert.Equal(t, "b", results[1].DocumentID)
	assert.Equal(t, 2, results[1].Rank)
	assert.Equal(t, "c", results[2].DocumentID)
	assert.Equal(t, 3, results[2].Rank)
}

func TestValidateAndMapResults_EqualScoreTieBreaksByIndexThenID(t *testing.T) {
	docs := []Document{
		{ID: "zeta", Text: "x"},
		{ID: "alpha", Text: "y"},
		{ID: "mid", Text: "z"},
	}
	raw := []RawResult{
		{Index: 1, Score: 1.0, HasIndex: true, HasScore: true},
		{Index: 0, Score: 1.0, HasIndex: true, HasScore: true},
		{Index: 2, Score: 1.0, HasIndex: true, HasScore: true},
	}
	results, err := ValidateAndMapResults(docs, 3, raw)
	require.NoError(t, err)
	// All equal scores: tie-break by index ascending.
	assert.Equal(t, 0, results[0].Index)
	assert.Equal(t, "zeta", results[0].DocumentID)
	assert.Equal(t, 1, results[1].Index)
	assert.Equal(t, "alpha", results[1].DocumentID)
	assert.Equal(t, 2, results[2].Index)
	assert.Equal(t, "mid", results[2].DocumentID)
}

func TestValidateAndMapResults_RejectsWrongCardinality(t *testing.T) {
	docs := []Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}}
	raw := []RawResult{{Index: 0, Score: 1.0, HasIndex: true, HasScore: true}}
	_, err := ValidateAndMapResults(docs, 2, raw)
	require.ErrorIs(t, err, ErrInvalidResponse)
	assert.Contains(t, err.Error(), "expected 2")
}

func TestValidateAndMapResults_RejectsMissingIndexOrScore(t *testing.T) {
	docs := []Document{{ID: "a", Text: "x"}}
	_, err := ValidateAndMapResults(docs, 1, []RawResult{{Score: 1.0, HasScore: true}})
	require.ErrorIs(t, err, ErrInvalidResponse)
	assert.Contains(t, err.Error(), "missing index")

	_, err = ValidateAndMapResults(docs, 1, []RawResult{{Index: 0, HasIndex: true}})
	require.ErrorIs(t, err, ErrInvalidResponse)
	assert.Contains(t, err.Error(), "missing relevance_score")
}

func TestValidateAndMapResults_RejectsOutOfRangeIndex(t *testing.T) {
	docs := []Document{{ID: "a", Text: "x"}}
	_, err := ValidateAndMapResults(docs, 1, []RawResult{{Index: 5, Score: 1.0, HasIndex: true, HasScore: true}})
	require.ErrorIs(t, err, ErrInvalidResponse)
	assert.Contains(t, err.Error(), "outside the submitted documents")
}

func TestValidateAndMapResults_RejectsDuplicateIndex(t *testing.T) {
	docs := []Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}}
	_, err := ValidateAndMapResults(docs, 2, []RawResult{
		{Index: 0, Score: 1.0, HasIndex: true, HasScore: true},
		{Index: 0, Score: 0.5, HasIndex: true, HasScore: true},
	})
	require.ErrorIs(t, err, ErrInvalidResponse)
	assert.Contains(t, err.Error(), "appears more than once")
}

func TestValidateAndMapResults_AcceptsZeroAndNegativeScores(t *testing.T) {
	docs := []Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}}
	// Zero score is present (HasScore true), negative is valid.
	raw := []RawResult{
		{Index: 0, Score: 0.0, HasIndex: true, HasScore: true},
		{Index: 1, Score: -5.0, HasIndex: true, HasScore: true},
	}
	results, err := ValidateAndMapResults(docs, 2, raw)
	require.NoError(t, err)
	assert.Equal(t, 0.0, results[0].Score)
	assert.Equal(t, "a", results[0].DocumentID)
	assert.Equal(t, -5.0, results[1].Score)
}

func TestFakeProvider_RoundTrip(t *testing.T) {
	provider := &fakeProvider{
		model: Model{Provider: "fake", Name: "fake-model"},
		raw: []RawResult{
			{Index: 0, Score: 0.9, HasIndex: true, HasScore: true},
			{Index: 1, Score: 0.1, HasIndex: true, HasScore: true},
		},
	}
	resp, err := provider.Rerank(context.Background(), Request{
		Query:     "q",
		Documents: []Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}},
		TopN:      2,
	})
	require.NoError(t, err)
	assert.Equal(t, "fake", resp.Provider)
	assert.Equal(t, "fake-model", resp.Model)
	require.Len(t, resp.Results, 2)
	assert.Equal(t, "a", resp.Results[0].DocumentID)
	assert.Equal(t, 1, resp.Results[0].Rank)
}
