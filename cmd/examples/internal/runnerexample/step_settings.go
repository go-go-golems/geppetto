package runnerexample

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

// OpenAIStepSettingsFromEnv returns basic OpenAI-backed step settings suitable
// for example programs.
func OpenAIStepSettingsFromEnv(model string, stream bool) (*settings.StepSettings, error) {
	ss, err := settings.NewStepSettings()
	if err != nil {
		return nil, err
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini"
	}

	apiType := types.ApiTypeOpenAI
	ss.Chat.ApiType = &apiType
	ss.Chat.Engine = &model
	ss.Chat.Stream = stream
	ss.API.APIKeys["openai-api-key"] = apiKey

	return ss, nil
}
