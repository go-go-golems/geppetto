package openai

import (
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
    Role    string                 `json:"role"`
    Content []responsesContentPart `json:"content"`
}

type responsesContentPart struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
    // image/audio not supported in first cut
}

type responsesResponse struct {
    Output []responsesOutputItem `json:"output"`
    // Error field intentionally omitted; HTTP non-2xx will carry error body
}

type responsesOutputItem struct {
    Content []responsesOutputContent `json:"content"`
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
    if s != nil && s.OpenAI != nil && s.OpenAI.IncludeReasoningEncrypted != nil && *s.OpenAI.IncludeReasoningEncrypted {
        req.Include = append(req.Include, "reasoning.encrypted_content")
    }
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
        default:
            return "assistant"
        }
    }
    for _, b := range t.Blocks {
        role := roleFor(b.Kind)
        // Only include text parts for first cut
        var parts []responsesContentPart
        if v, ok := b.Payload[turns.PayloadKeyText]; ok && v != nil {
            if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
                parts = append(parts, responsesContentPart{Type: "input_text", Text: s})
            }
        }
        if len(parts) == 0 {
            continue
        }
        items = append(items, responsesInput{Role: role, Content: parts})
    }
    return items
}

// PrepareToolsForResponses placeholder for parity; tools omitted in first cut.
func (e *OpenAIEngine) PrepareToolsForResponses(toolDefs []engine.ToolDefinition, cfg engine.ToolConfig) (any, error) {
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


