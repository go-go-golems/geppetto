package gemini

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/stretchr/testify/require"
)

func TestConstructorRequiresConnection(t *testing.T) {
	engine, err := NewGeminiEngine(nil, &settings.InferenceSettings{Chat: &settings.ChatSettings{}})
	require.Error(t, err)
	require.Nil(t, engine)
}
