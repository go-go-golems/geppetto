package claude

import (
	"context"
	"fmt"
	"github.com/3JoB/anthropic-sdk-go/v2"
	_ "github.com/3JoB/anthropic-sdk-go/v2"
	"github.com/3JoB/anthropic-sdk-go/v2/resp"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
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

	claudeSettings := csf.Settings.Claude
	if claudeSettings.APIKey == nil {
		return nil, steps.ErrMissingClientAPIKey
	}

	chatSettings := csf.Settings.Chat

	if chatSettings.Engine == nil {
		return nil, errors.New("missing engine")
	}

	client, err := anthropic.New(&anthropic.Config{
		Key:          *claudeSettings.APIKey,
		DefaultModel: *chatSettings.Engine,
	})
	if err != nil {
		return nil, err
	}

	prompt := ""
	for _, msg := range messages {
		switch msg.Role {
		case geppetto_context.RoleUser:
			prompt += fmt.Sprintf("Human: %s\n\n", msg.Text)
		case geppetto_context.RoleAssistant:
			prompt += fmt.Sprintf("Assistant: %s\n\n", msg.Text)
		case geppetto_context.RoleSystem:
			prompt += fmt.Sprintf("System: %s\n\n", msg.Text)
		default:
			return nil, errors.Errorf("unknown role: %s", msg.Role)
		}
	}

	sender := resp.Sender{
		Prompt:        prompt,
		StopSequences: chatSettings.Stop,
		Stream:        chatSettings.Stream,
		MaxToken:      1024,
	}
	if chatSettings.MaxResponseTokens != nil {
		sender.MaxToken = uint(*chatSettings.MaxResponseTokens)
	}
	if chatSettings.Temperature != nil {
		sender.Temperature = *chatSettings.Temperature
	}
	// TODO(manuel, 2023-09-28): bug in the anthropic SDK
	//if claudeSettings.TopK != nil {
	//	sender.TopK = *claudeSettings.TopK
	//}
	//if chatSettings.TopP != nil {
	//	sender.TopP = *chatSettings.TopP
	//}
	if claudeSettings.UserID != nil {
		sender.MetaData = resp.MetaData{
			UserID: *claudeSettings.UserID,
		}
	}
	opts := &anthropic.Opts{
		Sender: sender,
	}
	opts.With(client)

	//id, _ := ulid.New(ulid.Timestamp(time.Now()), rand.New())
	//opts.ContextID = id.String()
	//ctx_, err = opts.Complete(ctx_)
	//if err != nil {
	//	return nil, err
	//}
	//d, err := client.Send(opts)
	//
	//msgs_ := []go_openai.ChatCompletionMessage{}
	//for _, msg := range messages {
	//	msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
	//		Role:    msg.Role,
	//		Content: msg.Text,
	//	})
	//}
	//
	//temperature := 0.0
	//if chatSettings.Temperature != nil {
	//	temperature = *chatSettings.Temperature
	//}
	//topP := 0.0
	//if chatSettings.TopP != nil {
	//	topP = *chatSettings.TopP
	//}
	//maxTokens := 32
	//if chatSettings.MaxResponseTokens != nil {
	//	maxTokens = *chatSettings.MaxResponseTokens
	//}
	//
	//openaiSettings := csf.Settings.OpenAI
	//n := 1
	//if openaiSettings.N != nil {
	//	n = *openaiSettings.N
	//}
	//stream := chatSettings.Stream
	//stop := chatSettings.Stop
	//presencePenalty := 0.0
	//if openaiSettings.PresencePenalty != nil {
	//	presencePenalty = *openaiSettings.PresencePenalty
	//}
	//frequencyPenalty := 0.0
	//if openaiSettings.FrequencyPenalty != nil {
	//	frequencyPenalty = *openaiSettings.FrequencyPenalty
	//}
	//
	//req := go_openai.ChatCompletionRequest{
	//	Model:            engine,
	//	Messages:         msgs_,
	//	MaxTokens:        maxTokens,
	//	Temperature:      float32(temperature),
	//	TopP:             float32(topP),
	//	N:                n,
	//	Stream:           stream,
	//	Stop:             stop,
	//	PresencePenalty:  float32(presencePenalty),
	//	FrequencyPenalty: float32(frequencyPenalty),
	//	// TODO(manuel, 2023-03-28) Properly load logit bias
	//	// See https://github.com/go-go-golems/geppetto/issues/48
	//	LogitBias: nil,
	//}
	//
	//if stream {
	//	stream, err := client.CreateChatCompletionStream(ctx, req)
	//	if err != nil {
	//		return steps.Reject[string](err), nil
	//	}
	//	c := make(chan helpers.Result[string])
	//	ret := steps.NewStepResult[string](c)
	//
	//	go func() {
	//		defer close(c)
	//		defer stream.Close()
	//		for {
	//			select {
	//			case <-ctx.Done():
	//				return
	//			default:
	//				response, err := stream.Recv()
	//				if errors.Is(err, io.EOF) {
	//					c <- helpers.NewValueResult[string]("")
	//					return
	//				}
	//				if err != nil {
	//					c <- helpers.NewErrorResult[string](err)
	//					return
	//				}
	//
	//				c <- helpers.NewPartialResult[string](response.Choices[0].Delta.Content)
	//			}
	//		}
	//	}()
	//
	//	return ret, nil
	//} else {
	//	resp, err := client.CreateChatCompletion(ctx, req)
	//
	//	if err != nil {
	//		return steps.Reject[string](err), nil
	//	}
	//
	//	return steps.Resolve(string(resp.Choices[0].Message.Content)), nil
	//}

	return steps.Reject[string](errors.New("not implemented")), nil
}

// Close is only called after the returned monad has been entirely consumed
func (csf *Step) Close(ctx context.Context) error {
	return nil
}
