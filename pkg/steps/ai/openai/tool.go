package openai

import (
	"context"
	"encoding/json"
	"fmt"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	go_openai "github.com/sashabaranov/go-openai"
	"io"
)

type ToolStep struct {
	Settings *settings.StepSettings
	Tools    []go_openai.Tool
}

func (csf *ToolStep) SetStreaming(b bool) {
	csf.Settings.Chat.Stream = b
}

func (csf *ToolStep) Start(
	ctx context.Context,
	messages []*geppetto_context.Message,
) (*steps.StepResult[string], error) {
	client := makeClient(csf.Settings.OpenAI)

	req, err := makeCompletionRequest(csf.Settings, messages)
	if err != nil {
		return nil, err
	}

	req.Tools = csf.Tools
	stream := csf.Settings.Chat.Stream

	if stream {
		stream, err := client.CreateChatCompletionStream(ctx, *req)
		if err != nil {
			return steps.Reject[string](err), nil
		}
		c := make(chan helpers.Result[string])
		ret := steps.NewStepResult[string](c)

		// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
		go func() {
			defer close(c)
			defer stream.Close()

			toolCallMerger := NewToolCallMerger()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					response, err := stream.Recv()
					if errors.Is(err, io.EOF) {
						c <- helpers.NewValueResult[string]("")
						toolCalls := toolCallMerger.GetToolCalls()
						s, _ := json.MarshalIndent(toolCalls, "", " ")
						fmt.Printf("final toolcaalls:\n%s\n", s)

						return
					}
					if err != nil {
						c <- helpers.NewErrorResult[string](err)
						return
					}

					// TODO(manuel, 2023-11-28) Handle multiple choices
					toolCallMerger.AddToolCalls(response.Choices[0].Delta.ToolCalls)

					c <- helpers.NewPartialResult[string](response.Choices[0].Delta.Content)
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.CreateChatCompletion(ctx, *req)

		if err != nil {
			return steps.Reject[string](err), nil
		}

		// TODO(manuel, 2023-11-28) Handle multiple choices
		s, _ := json.MarshalIndent(resp.Choices[0].Message.ToolCalls, "", " ")
		fmt.Printf("final toolcalls:\n%s\n%s\n", resp.Choices[0].FinishReason, s)
		return steps.Resolve(string(resp.Choices[0].Message.Content)), nil
	}
}

// Close is only called after the returned monad has been entirely consumed
func (csf *ToolStep) Close(ctx context.Context) error {
	return nil
}
