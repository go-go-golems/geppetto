package claude

import (
	"context"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	go_openai "github.com/sashabaranov/go-openai"
	"io"
	"strings"
)

type Step struct {
	Settings *settings.StepSettings
}

func IsClaudeEngine(engine string) bool {
	if strings.HasPrefix(engine, "claude") {
		return true
	}

	return false
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

	claudeSettings := csf.Settings.Claude
	if claudeSettings.APIKey == nil {
		return nil, steps.ErrMissingClientAPIKey
	}

	client := go_openai.NewClient(*claudeSettings.APIKey)

	engine := ""

	chatSettings := csf.Settings.Chat
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	msgs_ := []go_openai.ChatCompletionMessage{}
	for _, msg := range messages {
		msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Text,
		})
	}

	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}
	topP := 0.0
	if chatSettings.TopP != nil {
		topP = *chatSettings.TopP
	}
	maxTokens := 32
	if chatSettings.MaxResponseTokens != nil {
		maxTokens = *chatSettings.MaxResponseTokens
	}

	openaiSettings := csf.Settings.OpenAI
	n := 1
	if openaiSettings.N != nil {
		n = *openaiSettings.N
	}
	stream := chatSettings.Stream
	stop := chatSettings.Stop
	presencePenalty := 0.0
	if openaiSettings.PresencePenalty != nil {
		presencePenalty = *openaiSettings.PresencePenalty
	}
	frequencyPenalty := 0.0
	if openaiSettings.FrequencyPenalty != nil {
		frequencyPenalty = *openaiSettings.FrequencyPenalty
	}

	req := go_openai.ChatCompletionRequest{
		Model:            engine,
		Messages:         msgs_,
		MaxTokens:        maxTokens,
		Temperature:      float32(temperature),
		TopP:             float32(topP),
		N:                n,
		Stream:           stream,
		Stop:             stop,
		PresencePenalty:  float32(presencePenalty),
		FrequencyPenalty: float32(frequencyPenalty),
		// TODO(manuel, 2023-03-28) Properly load logit bias
		// See https://github.com/go-go-golems/geppetto/issues/48
		LogitBias: nil,
	}

	if stream {
		stream, err := client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			return steps.Reject[string](err), nil
		}
		c := make(chan helpers.Result[string])
		ret := steps.NewStepResult[string](c)

		go func() {
			defer close(c)
			defer stream.Close()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					response, err := stream.Recv()
					if errors.Is(err, io.EOF) {
						c <- helpers.NewValueResult[string]("")
						return
					}
					if err != nil {
						c <- helpers.NewErrorResult[string](err)
						return
					}

					c <- helpers.NewPartialResult[string](response.Choices[0].Delta.Content)
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.CreateChatCompletion(ctx, req)

		if err != nil {
			return steps.Reject[string](err), nil
		}

		return steps.Resolve(string(resp.Choices[0].Message.Content)), nil
	}
}

// Close is only called after the returned monad has been entirely consumed
func (csf *Step) Close(ctx context.Context) error {
	return nil
}
