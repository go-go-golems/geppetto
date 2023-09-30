package claude

import (
	"context"
	"encoding/json"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	"strings"
)

type Step struct {
	Settings *settings.StepSettings
}

func IsClaudeEngine(engine string) bool {
	return strings.HasPrefix(engine, "claude")
}

func (csf *Step) SetStreaming(b bool) {
	csf.Settings.Chat.Stream = b
}

func (csf *Step) Start(
	ctx context.Context,
	messages []*geppetto_context.Message,
) (*steps.StepResult[string], error) {
	clientSettings := csf.Settings.Client
	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}
	anthropicSettings := csf.Settings.Claude
	if anthropicSettings == nil {
		return nil, errors.New("no claude settings")
	}

	if anthropicSettings.APIKey == nil {
		return nil, steps.ErrMissingClientAPIKey
	}

	client := NewClient(*anthropicSettings.APIKey)

	engine := ""

	chatSettings := csf.Settings.Chat
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	// Combine all the messages into a single prompt
	prompt := ""
	for _, msg := range messages {
		rolePrefix := "Human"
		switch msg.Role {
		case geppetto_context.RoleSystem:
			rolePrefix = "System"
		case geppetto_context.RoleAssistant:
			rolePrefix = "Assistant"
		case geppetto_context.RoleUser:
			rolePrefix = "Human"
		}
		prompt += "\n\n" + rolePrefix + ": " + msg.Text
	}
	prompt += "\n\nAssistant: "

	maxTokens := 32
	if chatSettings.MaxResponseTokens != nil {
		maxTokens = *chatSettings.MaxResponseTokens
	}

	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}
	topP := 0.0
	if chatSettings.TopP != nil {
		topP = *chatSettings.TopP
	}

	req := Request{
		Model:             engine,
		Prompt:            prompt,
		MaxTokensToSample: maxTokens,
		Temperature:       &temperature,
		TopP:              &topP,
		Stream:            chatSettings.Stream,
	}

	if chatSettings.Stream {
		events, err := client.StreamComplete(&req)
		if err != nil {
			return steps.Reject[string](err), nil
		}
		c := make(chan helpers.Result[string])
		ret := steps.NewStepResult[string](c)

		go func() {
			defer close(c)

			isFirstEvent := true
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-events:
					if !ok {
						c <- helpers.NewValueResult[string]("")
						return
					}
					decoded := map[string]interface{}{}
					err := json.Unmarshal([]byte(event.Data), &decoded)
					if err != nil {
						c <- helpers.NewErrorResult[string](err)
						return
					}
					if completion, exists := decoded["completion"].(string); exists {
						if isFirstEvent {
							completion = strings.TrimLeft(completion, " ")
							isFirstEvent = false
						}
						c <- helpers.NewPartialResult[string](completion)
					}
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.Complete(&req)

		if err != nil {
			return steps.Reject[string](err), nil
		}

		return steps.Resolve(resp.Completion), nil
	}
}

// Close is only called after the returned monad has been entirely consumed
func (csf *Step) Close(ctx context.Context) error {
	return nil
}
