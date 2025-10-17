package openai_responses

import (
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// HTTP JSON models for OpenAI Responses API (minimal subset: text + reasoning + sampling)

type responsesRequest struct {
    Model           string             `json:"model"`
    Input           []responsesInput   `json:"input"`
    MaxOutputTokens *int               `json:"max_output_tokens,omitempty"`
    Temperature     *float64           `json:"temperature,omitempty"`
    TopP            *float64           `json:"top_p,omitempty"`
    StopSequences   []string           `json:"stop_sequences,omitempty"`
    Reasoning       *reasoningParam    `json:"reasoning,omitempty"`
    Stream          bool               `json:"stream,omitempty"`
    Include         []string           `json:"include,omitempty"`
    Tools           []any              `json:"tools,omitempty"`
    ToolChoice      any                `json:"tool_choice,omitempty"`
    ParallelToolCalls *bool            `json:"parallel_tool_calls,omitempty"`
}

type reasoningParam struct {
    Effort string `json:"effort,omitempty"`
    Summary string `json:"summary,omitempty"`
}

type responsesInput struct {
    // Message-style item
    Role    string                 `json:"role,omitempty"`
    Content []responsesContentPart `json:"content,omitempty"`
    // Item-style entries (reasoning, function_call, function_call_output)
    Type             string `json:"type,omitempty"`
    ID               string `json:"id,omitempty"`
    EncryptedContent string `json:"encrypted_content,omitempty"`
    Summary          *[]any `json:"summary,omitempty"`
    // function_call
    CallID    string `json:"call_id,omitempty"`
    Name      string `json:"name,omitempty"`
    Arguments string `json:"arguments,omitempty"`
    // function_call_output
    // Some providers expect call_id for both function_call and function_call_output
    ToolCallID string `json:"tool_call_id,omitempty"`
    Output     string `json:"output,omitempty"`
}

type responsesContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// For function_call content type
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	// For tool_result content type
	ToolCallID string `json:"tool_call_id,omitempty"`
	Content    string `json:"content,omitempty"`
	// image/audio not supported in first cut
}

type responsesResponse struct {
    Output []responsesOutputItem `json:"output"`
    // Error field intentionally omitted; HTTP non-2xx will carry error body
}

type responsesOutputItem struct {
    Type    string                   `json:"type,omitempty"`
    ID      string                   `json:"id,omitempty"`
    Name    string                   `json:"name,omitempty"`
    CallID  string                   `json:"call_id,omitempty"`
    Arguments string                 `json:"arguments,omitempty"`
    Content []responsesOutputContent `json:"content"`
    EncryptedContent string          `json:"encrypted_content,omitempty"`
}

type responsesOutputContent struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
}

// buildResponsesRequest constructs a minimal Responses request from Turn + settings
func buildResponsesRequest(s *settings.StepSettings, t *turns.Turn) (responsesRequest, error) {
    req := responsesRequest{}
    if s != nil && s.Chat != nil && s.Chat.Engine != nil {
        req.Model = *s.Chat.Engine
    }
    req.Input = buildInputItemsFromTurn(t)
    if s != nil && s.Chat != nil {
        if s.Chat.MaxResponseTokens != nil {
            req.MaxOutputTokens = s.Chat.MaxResponseTokens
        }
        // Some reasoning models (o3*/o4*) do not accept temperature/top_p; omit for those
        m := strings.ToLower(req.Model)
        allowSampling := !strings.HasPrefix(m, "o3") && !strings.HasPrefix(m, "o4")
        if allowSampling && s.Chat.Temperature != nil {
            req.Temperature = s.Chat.Temperature
        }
        if allowSampling && s.Chat.TopP != nil {
            req.TopP = s.Chat.TopP
        }
        if len(s.Chat.Stop) > 0 {
            req.StopSequences = s.Chat.Stop
        }
        req.Stream = s.Chat.Stream
    }
    if s != nil && s.OpenAI != nil && s.OpenAI.ReasoningEffort != nil {
        if req.Reasoning == nil { req.Reasoning = &reasoningParam{} }
        req.Reasoning.Effort = mapEffortString(*s.OpenAI.ReasoningEffort)
    }
    if s != nil && s.OpenAI != nil && s.OpenAI.ReasoningSummary != nil && *s.OpenAI.ReasoningSummary != "" {
        if req.Reasoning == nil { req.Reasoning = &reasoningParam{} }
        req.Reasoning.Summary = *s.OpenAI.ReasoningSummary
    }
    // Force include encrypted reasoning content on every request for stateless continuation.
    req.Include = append(req.Include, "reasoning.encrypted_content")
    // NOTE: stream_options.include_usage is not supported broadly; ignore for now
    return req, nil
}

func mapEffortString(v string) string {
    switch strings.ToLower(strings.TrimSpace(v)) {
    case "low":
        return "low"
    case "high":
        return "high"
    default:
        return "medium"
    }
}

func buildInputItemsFromTurn(t *turns.Turn) []responsesInput {
    var items []responsesInput
    if t == nil {
        return items
    }
    roleFor := func(kind turns.BlockKind) string {
        switch kind {
        case turns.BlockKindSystem:
            return "system"
        case turns.BlockKindUser:
            return "user"
        case turns.BlockKindToolUse:
            return "tool"
        default:
            return "assistant"
        }
    }

    // Locate the latest reasoning block index (if any)
    latestReasoningIdx := -1
    for i := len(t.Blocks) - 1; i >= 0; i-- {
        if t.Blocks[i].Kind == turns.BlockKindReasoning {
            latestReasoningIdx = i
            break
        }
    }

    // Helpers
    appendMessage := func(b turns.Block) {
        role := roleFor(b.Kind)
        var parts []responsesContentPart
        if v, ok := b.Payload[turns.PayloadKeyText]; ok && v != nil {
            if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
                parts = append(parts, responsesContentPart{Type: "input_text", Text: s})
            }
        }
        if len(parts) > 0 {
            items = append(items, responsesInput{Role: role, Content: parts})
        }
    }
    appendFunctionCall := func(b turns.Block) {
        name, _ := b.Payload[turns.PayloadKeyName].(string)
        callID, _ := b.Payload[turns.PayloadKeyID].(string)
        itemID, _ := b.Payload[turns.PayloadKeyItemID].(string)
        args := b.Payload[turns.PayloadKeyArgs]
        var argsJSON string
        if args != nil {
            if s, ok := args.(string); ok {
                argsJSON = s
            } else if bb, err := json.Marshal(args); err == nil {
                argsJSON = string(bb)
            }
        }
        if callID != "" && name != "" {
            ri := responsesInput{Type: "function_call", CallID: callID, Name: name, Arguments: argsJSON}
            if itemID != "" { ri.ID = itemID }
            items = append(items, ri)
        }
    }
    appendFunctionCallOutput := func(b turns.Block) {
        toolID, _ := b.Payload[turns.PayloadKeyID].(string)
        result := b.Payload[turns.PayloadKeyResult]
        var resultJSON string
        if result != nil {
            if s, ok := result.(string); ok {
                resultJSON = s
            } else if bb, err := json.Marshal(result); err == nil {
                resultJSON = string(bb)
            }
        }
        if toolID != "" {
            // Responses expects call_id on function_call_output
            items = append(items, responsesInput{Type: "function_call_output", CallID: toolID, Output: resultJSON})
        }
    }

    // 1) Pre-context: all blocks before latest reasoning, skipping older reasoning
    if latestReasoningIdx > 0 {
        for i := 0; i < latestReasoningIdx; i++ {
            b := t.Blocks[i]
            switch b.Kind {
            case turns.BlockKindReasoning:
                continue
            case turns.BlockKindToolCall:
                appendFunctionCall(b)
            case turns.BlockKindToolUse:
                appendFunctionCallOutput(b)
            default:
                appendMessage(b)
            }
        }
    } else if latestReasoningIdx == -1 {
        // No reasoning present: just include all blocks as before
        for _, b := range t.Blocks {
            switch b.Kind {
            case turns.BlockKindToolCall:
                appendFunctionCall(b)
            case turns.BlockKindToolUse:
                appendFunctionCallOutput(b)
            default:
                appendMessage(b)
            }
        }
        return items
    }

    // 2) Latest reasoning item followed by its matching tool_call and tool_call_output items
    if latestReasoningIdx >= 0 {
        rb := t.Blocks[latestReasoningIdx]
        enc, _ := rb.Payload[turns.PayloadKeyEncryptedContent].(string)
        // Provide an empty summary array to satisfy provider requirement
        empty := make([]any, 0)
        ri := responsesInput{Type: "reasoning", ID: rb.ID, Summary: &empty}
        if enc != "" {
            ri.EncryptedContent = enc
        }
        items = append(items, ri)
        // Find the first tool_call after the reasoning item anywhere later in the turn
        firstCallIdx := -1
        for j := latestReasoningIdx + 1; j < len(t.Blocks); j++ {
            if t.Blocks[j].Kind == turns.BlockKindToolCall {
                firstCallIdx = j
                break
            }
        }
        includedToolCallID := ""
        if firstCallIdx >= 0 {
            appendFunctionCall(t.Blocks[firstCallIdx])
            if v, ok := t.Blocks[firstCallIdx].Payload[turns.PayloadKeyID].(string); ok { includedToolCallID = v }
            // Find matching tool_use (function_call_output) with same id anywhere after the call
            if includedToolCallID != "" {
                for k := firstCallIdx + 1; k < len(t.Blocks); k++ {
                    if t.Blocks[k].Kind == turns.BlockKindToolUse {
                        if id, _ := t.Blocks[k].Payload[turns.PayloadKeyID].(string); id == includedToolCallID {
                            appendFunctionCallOutput(t.Blocks[k])
                            break
                        }
                    }
                }
            }
        }
    }

    // 3) Remaining blocks after the grouped segment
    end := latestReasoningIdx + 1
    for end < len(t.Blocks) {
        if t.Blocks[end].Kind != turns.BlockKindToolCall && t.Blocks[end].Kind != turns.BlockKindToolUse {
            break
        }
        end++
    }
    for k := end; k < len(t.Blocks); k++ {
        b := t.Blocks[k]
        switch b.Kind {
        case turns.BlockKindReasoning:
            // skip any additional reasoning items
            continue
        case turns.BlockKindToolCall:
            // Avoid duplicating the tool_call we already grouped with reasoning
            // (we compare by ID)
            // Note: includedToolCallID is empty outside the latestReasoningIdx branch
            if latestReasoningIdx >= 0 {
                // fetch includedToolCallID from closure scope by recomputing
            }
            appendFunctionCall(b)
        case turns.BlockKindToolUse:
            appendFunctionCallOutput(b)
        default:
            appendMessage(b)
        }
    }

    return items
}

// PrepareToolsForResponses placeholder for parity; tools omitted in first cut.
func (e *Engine) PrepareToolsForResponses(toolDefs []engine.ToolDefinition, cfg engine.ToolConfig) (any, error) {
    if !cfg.Enabled {
        return nil, nil
    }
    // Convert engine.ToolDefinition to OpenAI Responses tool format
    // {"type":"function","function":{"name":...,"description":...,"parameters":{...}}}
    tools := make([]any, 0, len(toolDefs))
    for _, td := range toolDefs {
        tool := map[string]any{
            "type":        "function",
            "name":        td.Name,
            "description": td.Description,
        }
        if td.Parameters != nil {
            // Responses API expects top-level parameters for function tools
            tool["parameters"] = td.Parameters
        }
        tools = append(tools, tool)
    }
    return tools, nil
}


