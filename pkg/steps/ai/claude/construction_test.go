package claude

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connectionfactory"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/stretchr/testify/require"
)

// Test fixture: resolve declared configuration before constructing payload code.
func newTestEngineFromSettings(s *settings.InferenceSettings, opts ...EngineOption) (*ClaudeEngine, error) {
	c, err := connectionfactory.New(s, nil)
	if err != nil {
		return nil, err
	}
	return NewClaudeEngine(c, s, opts...)
}
func newTestCounterFromSettings(s *settings.InferenceSettings) (*TokenCounter, error) {
	c, err := connectionfactory.New(s, nil)
	if err != nil {
		return nil, err
	}
	return NewTokenCounter(c, s)
}

func TestConstructorRequiresConnection(t *testing.T) {
	engine, err := NewClaudeEngine(nil, &settings.InferenceSettings{Chat: &settings.ChatSettings{}})
	require.Error(t, err)
	require.Nil(t, engine)
}
func TestCounterRequiresConnection(t *testing.T) {
	counter, err := NewTokenCounter(nil, &settings.InferenceSettings{Chat: &settings.ChatSettings{}})
	require.Error(t, err)
	require.Nil(t, counter)
}
