package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	go_openai "github.com/sashabaranov/go-openai"
	"io"
)

type ToolStep struct {
	Settings *settings.StepSettings
	Tools    []go_openai.Tool
}

var _ steps.Step[[]*conversation.Message, ToolCompletionResponse] = (*ToolStep)(nil)

func (csf *ToolStep) SetStreaming(b bool) {
	csf.Settings.Chat.Stream = b
}

type ToolCompletionResponse struct {
	Role      string               `json:"role"`
	Content   string               `json:"content"`
	ToolCalls []go_openai.ToolCall `json:"tool_calls"`
}

func (csf *ToolStep) Start(
	ctx context.Context,
	messages []*conversation.Message,
) (steps.StepResult[ToolCompletionResponse], error) {
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
			return steps.Reject[ToolCompletionResponse](err), nil
		}
		c := make(chan helpers.Result[ToolCompletionResponse])
		ret := steps.NewStepResult[ToolCompletionResponse](c)

		// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
		go func() {
			defer close(c)
			defer stream.Close()

			toolCallMerger := NewToolCallMerger()

			ret := ToolCompletionResponse{}

			for {
				select {
				case <-ctx.Done():
					return
				default:
					response, err := stream.Recv()
					if errors.Is(err, io.EOF) {
						toolCalls := toolCallMerger.GetToolCalls()

						ret.ToolCalls = toolCalls
						c <- helpers.NewValueResult[ToolCompletionResponse](ret)

						return
					}
					if err != nil {
						c <- helpers.NewErrorResult[ToolCompletionResponse](err)
						return
					}

					// TODO(manuel, 2023-11-28) Handle multiple choices
					delta := response.Choices[0].Delta
					toolCallMerger.AddToolCalls(delta.ToolCalls)

					if delta.Role != "" {
						ret.Role = delta.Role
					}
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.CreateChatCompletion(ctx, *req)

		if err != nil {
			return steps.Reject[ToolCompletionResponse](err), nil
		}

		// TODO(manuel, 2023-11-28) Handle multiple choices
		s, _ := json.MarshalIndent(resp.Choices[0].Message.ToolCalls, "", " ")
		fmt.Printf("final toolcalls:\n%s\n%s\n", resp.Choices[0].FinishReason, s)
		ret := ToolCompletionResponse{
			Role:      resp.Choices[0].Message.Role,
			Content:   string(resp.Choices[0].Message.Content),
			ToolCalls: resp.Choices[0].Message.ToolCalls,
		}
		return steps.Resolve(ret), nil
	}
}

var _ steps.Step[ToolCompletionResponse, map[string]interface{}] = (*ExecuteToolStep)(nil)

type ExecuteToolStep struct {
	Tools map[string]interface{}
}

func (e ExecuteToolStep) Start(
	ctx context.Context,
	input ToolCompletionResponse,
) (steps.StepResult[map[string]interface{}], error) {
	res := map[string]interface{}{}
	for _, toolCall := range input.ToolCalls {
		if toolCall.Type != "function" {
			log.Warn().Str("type", string(toolCall.Type)).Msg("Unknown tool type")
			continue
		}
		tool := e.Tools[toolCall.Function.Name]
		if tool == nil {
			return steps.Reject[map[string]interface{}](fmt.Errorf("could not find tool %s", toolCall.Function.Name)), nil
		}

		var v interface{}
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &v)
		if err != nil {
			return steps.Reject[map[string]interface{}](err), nil
		}

		vs_, err := helpers.CallFunctionFromJson(tool, v)
		if err != nil {
			return steps.Reject[map[string]interface{}](err), nil
		}

		if len(vs_) == 1 {
			res[toolCall.Function.Name] = vs_[0].Interface()
		} else {
			vals := []interface{}{}
			for _, v_ := range vs_ {
				vals = append(vals, v_.Interface())
			}
			res[toolCall.Function.Name] = vals
		}
	}

	return steps.Resolve(res), nil
}
