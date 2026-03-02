package engine

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
)

func TestBuildInferenceResultFromEventMetadata_MaxTokensIsSnapshot(t *testing.T) {
	maxTokens := 256
	metadata := events.EventMetadata{
		LLMInferenceData: events.LLMInferenceData{
			MaxTokens: &maxTokens,
		},
	}

	result := BuildInferenceResultFromEventMetadata(metadata, "openai", false)
	if result.MaxTokens == nil {
		t.Fatalf("expected max_tokens to be set")
	}
	if *result.MaxTokens != 256 {
		t.Fatalf("expected max_tokens=256, got %d", *result.MaxTokens)
	}
	if result.MaxTokens == metadata.MaxTokens {
		t.Fatalf("expected max_tokens pointer to be copied")
	}

	maxTokens = 2048
	if *result.MaxTokens != 256 {
		t.Fatalf("expected max_tokens snapshot to stay 256, got %d", *result.MaxTokens)
	}
}
