package openai_responses

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connectionfactory"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/stretchr/testify/require"
)

// Test fixture: resolve declared configuration before constructing payload code.
func newTestEngineFromSettings(s *settings.InferenceSettings, opts ...EngineOption) (*Engine, error) {
	s = responsesTestSettings(s)
	c, err := connectionfactory.New(s, nil)
	if err != nil {
		return nil, err
	}
	return NewEngine(c, s, opts...)
}
func newTestCounterFromSettings(s *settings.InferenceSettings) (*TokenCounter, error) {
	s = responsesTestSettings(s)
	c, err := connectionfactory.New(s, nil)
	if err != nil {
		return nil, err
	}
	return NewTokenCounter(c, s)
}

func newTestEngineWithSource(s *settings.InferenceSettings, source credentials.BearerTokenSource) (*Engine, error) {
	s = responsesTestSettings(s)
	c, err := connectionfactory.New(s, source)
	if err != nil {
		return nil, err
	}
	return NewEngine(c, s)
}

func responsesTestSettings(s *settings.InferenceSettings) *settings.InferenceSettings {
	if s == nil {
		return nil
	}
	out := *s
	if s.Chat != nil {
		out.Chat = s.Chat.Clone()
	} else {
		out.Chat = &settings.ChatSettings{}
	}
	protocol := types.ApiTypeOpenResponses
	out.Chat.ApiType = &protocol
	return &out
}

func TestConstructorRequiresConnection(t *testing.T) {
	engine, err := NewEngine(nil, &settings.InferenceSettings{Chat: &settings.ChatSettings{}})
	require.Error(t, err)
	require.Nil(t, engine)
}
func TestCounterRequiresConnection(t *testing.T) {
	counter, err := NewTokenCounter(nil, &settings.InferenceSettings{Chat: &settings.ChatSettings{}})
	require.Error(t, err)
	require.Nil(t, counter)
}
