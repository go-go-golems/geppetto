package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDocuments_UsesFirstPipeAndDoesNotLeakInput(t *testing.T) {
	docs, err := parseDocuments([]string{"a|text with | another pipe"})
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "a", docs[0].ID)
	assert.Equal(t, "text with | another pipe", docs[0].Text)

	_, err = parseDocuments([]string{"CALLER-ID-SECRET text without separator"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "CALLER-ID-SECRET")
}

func TestResolveSettings_InlineModeNeedsNoRegistry(t *testing.T) {
	resolved, closeFn, label, err := resolveSettings(context.Background(), nil, "", &rerankSettings{
		RerankType:    "llamacpp",
		RerankEngine:  "test-model",
		RerankBaseURL: "http://127.0.0.1:18012",
	})
	require.NoError(t, err)
	assert.Nil(t, closeFn)
	assert.Equal(t, "inline rerank(llamacpp/test-model)", label)
	require.NotNil(t, resolved)
	require.NotNil(t, resolved.Rerank)
	assert.Equal(t, "llamacpp", resolved.Rerank.Type)
	assert.Equal(t, "test-model", resolved.Rerank.Engine)
	require.NotNil(t, resolved.API)
	assert.Equal(t, "http://127.0.0.1:18012", resolved.API.BaseUrls["rerank-base-url"])
}
