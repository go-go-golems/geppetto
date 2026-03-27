package factory

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestNewFromSettings_OpenResponsesSupported(t *testing.T) {
	apiType := types.ApiTypeOpenResponses
	ss := &settings.InferenceSettings{
		Chat: &settings.ChatSettings{
			ApiType: &apiType,
		},
	}

	counter, err := NewFromSettings(ss)
	if err != nil {
		t.Fatalf("NewFromSettings returned error: %v", err)
	}
	if counter == nil {
		t.Fatalf("expected counter, got nil")
	}
}

func TestNewFromSettings_OpenAIResponsesSupported(t *testing.T) {
	apiType := types.ApiTypeOpenAIResponses
	ss := &settings.InferenceSettings{
		Chat: &settings.ChatSettings{
			ApiType: &apiType,
		},
	}

	counter, err := NewFromSettings(ss)
	if err != nil {
		t.Fatalf("NewFromSettings returned error: %v", err)
	}
	if counter == nil {
		t.Fatalf("expected counter, got nil")
	}
}

func TestNewFromSettings_OpenAINotSupported(t *testing.T) {
	apiType := types.ApiTypeOpenAI
	ss := &settings.InferenceSettings{
		Chat: &settings.ChatSettings{
			ApiType: &apiType,
		},
	}

	counter, err := NewFromSettings(ss)
	if err == nil {
		t.Fatalf("expected error, got counter=%v", counter)
	}
}
