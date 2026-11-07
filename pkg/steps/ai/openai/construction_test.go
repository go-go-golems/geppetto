package openai

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connectionfactory"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/stretchr/testify/require"
)

// Test fixture: resolve declared configuration before constructing payload code.
func newTestEngineFromSettings(s *settings.InferenceSettings, opts ...EngineOption) (*OpenAIEngine, error) {
	c, err := connectionfactory.New(s, nil)
	if err != nil {
		return nil, err
	}
	return NewOpenAIEngine(c, s, opts...)
}

func newTestEngineWithSource(s *settings.InferenceSettings, source credentials.BearerTokenSource) (*OpenAIEngine, error) {
	c, err := connectionfactory.New(s, source)
	if err != nil {
		return nil, err
	}
	return NewOpenAIEngine(c, s)
}

func TestConstructorRequiresConnection(t *testing.T) {
	engine, err := NewOpenAIEngine(nil, &settings.InferenceSettings{Chat: &settings.ChatSettings{}})
	require.Error(t, err)
	require.Nil(t, engine)
}
