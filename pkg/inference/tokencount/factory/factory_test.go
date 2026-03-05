package factory

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestNewFromStepSettings_OpenAIResponsesSupported(t *testing.T) {
	apiType := types.ApiTypeOpenAIResponses
	ss := &settings.StepSettings{
		Chat: &settings.ChatSettings{
			ApiType: &apiType,
		},
	}

	counter, err := NewFromStepSettings(ss)
	if err != nil {
		t.Fatalf("NewFromStepSettings returned error: %v", err)
	}
	if counter == nil {
		t.Fatalf("expected counter, got nil")
	}
}

func TestNewFromStepSettings_OpenAINotSupported(t *testing.T) {
	apiType := types.ApiTypeOpenAI
	ss := &settings.StepSettings{
		Chat: &settings.ChatSettings{
			ApiType: &apiType,
		},
	}

	counter, err := NewFromStepSettings(ss)
	if err == nil {
		t.Fatalf("expected error, got counter=%v", counter)
	}
}
