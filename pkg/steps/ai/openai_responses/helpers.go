package openai_responses

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/rs/zerolog/log"
)

// HTTP JSON models for OpenAI Responses API (minimal subset: text + reasoning + sampling)

type responsesRequest struct {
	Model             string           `json:"model"`
	Input             []responsesInput `json:"input"`
	Text              *responsesText   `json:"text,omitempty"`
	MaxOutputTokens   *int             `json:"max_output_tokens,omitempty"`
	Temperature       *float64         `json:"temperature,omitempty"`
	TopP              *float64         `json:"top_p,omitempty"`
	StopSequences     []string         `json:"stop_sequences,omitempty"`
	Reasoning         *reasoningParam  `json:"reasoning,omitempty"`
	Stream            bool             `json:"stream,omitempty"`
	Include           []string         `json:"include,omitempty"`
	Tools             []any            `json:"tools,omitempty"`
	ToolChoice        any              `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool            `json:"parallel_tool_calls,omitempty"`
	Store             *bool            `json:"store,omitempty"`
	ServiceTier       *string          `json:"service_tier,omitempty"`
}

type responsesText struct {
	Format *responsesTextFormat `json:"format,omitempty"`
}

type responsesTextFormat struct {
	Type        string         `json:"type"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema,omitempty"`
	Strict      *bool          `json:"strict,omitempty"`
}

type reasoningParam struct {
	Effort    string `json:"effort,omitempty"`
	Summary   string `json:"summary,omitempty"`
	MaxTokens *int   `json:"max_tokens,omitempty"`
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
	Type             string                   `json:"type,omitempty"`
	ID               string                   `json:"id,omitempty"`
	Name             string                   `json:"name,omitempty"`
	CallID           string                   `json:"call_id,omitempty"`
	Arguments        string                   `json:"arguments,omitempty"`
	Content          []responsesOutputContent `json:"content"`
	EncryptedContent string                   `json:"encrypted_content,omitempty"`
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
		// Some reasoning models (o1/o3/o4/gpt-5) do not accept temperature/top_p; omit for those.
		allowSampling := !isResponsesReasoningModel(req.Model)
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
		if req.Reasoning == nil {
			req.Reasoning = &reasoningParam{}
		}
		req.Reasoning.Effort = mapEffortString(*s.OpenAI.ReasoningEffort)
	}
	if s != nil && s.OpenAI != nil && s.OpenAI.ReasoningSummary != nil && *s.OpenAI.ReasoningSummary != "" {
		if req.Reasoning == nil {
			req.Reasoning = &reasoningParam{}
		}
		req.Reasoning.Summary = *s.OpenAI.ReasoningSummary
	}
	// Force include encrypted reasoning content on every request for stateless continuation.
	req.Include = append(req.Include, "reasoning.encrypted_content")
	// Apply provider-native structured output schema for Responses API when configured.
	if s != nil && s.Chat != nil && s.Chat.IsStructuredOutputEnabled() {
		cfg, err := s.Chat.StructuredOutputConfig()
		if err != nil {
			if s.Chat.StructuredOutputRequireValid {
				return req, err
			}
			log.Warn().Err(err).Msg("OpenAI Responses request: ignoring invalid structured output configuration")
		} else if cfg != nil {
			strict := cfg.StrictOrDefault()
			req.Text = &responsesText{
				Format: &responsesTextFormat{
					Type:        "json_schema",
					Name:        cfg.Name,
					Description: cfg.Description,
					Schema:      cfg.Schema,
					Strict:      &strict,
				},
			}
		}
	}
	// Apply per-turn InferenceConfig overrides (Turn.Data > StepSettings.Inference).
	var engineInference *engine.InferenceConfig
	if s != nil {
		engineInference = s.Inference
	}
	if infCfg := engine.ResolveInferenceConfig(t, engineInference); infCfg != nil {
		if infCfg.ReasoningEffort != nil {
			if req.Reasoning == nil {
				req.Reasoning = &reasoningParam{}
			}
			req.Reasoning.Effort = mapEffortString(*infCfg.ReasoningEffort)
		}
		if infCfg.ReasoningSummary != nil && *infCfg.ReasoningSummary != "" {
			if req.Reasoning == nil {
				req.Reasoning = &reasoningParam{}
			}
			req.Reasoning.Summary = *infCfg.ReasoningSummary
		}
		if infCfg.ThinkingBudget != nil && *infCfg.ThinkingBudget > 0 {
			if req.Reasoning == nil {
				req.Reasoning = &reasoningParam{}
			}
			req.Reasoning.MaxTokens = infCfg.ThinkingBudget
		}
		// Reasoning models (o1/o3/o4/gpt-5) do not accept temperature/top_p;
		// respect the same guard used for base chat settings above.
		overrideAllowSampling := !isResponsesReasoningModel(req.Model)
		if overrideAllowSampling && infCfg.Temperature != nil {
			req.Temperature = infCfg.Temperature
		}
		if overrideAllowSampling && infCfg.TopP != nil {
			req.TopP = infCfg.TopP
		}
		if infCfg.MaxResponseTokens != nil {
			req.MaxOutputTokens = infCfg.MaxResponseTokens
		}
		if len(infCfg.Stop) > 0 {
			req.StopSequences = infCfg.Stop
		}
	}

	// Apply OpenAI-specific per-turn overrides from Turn.Data.
	if oaiCfg := engine.ResolveOpenAIInferenceConfig(t); oaiCfg != nil {
		if oaiCfg.Store != nil {
			req.Store = oaiCfg.Store
		}
		if oaiCfg.ServiceTier != nil {
			req.ServiceTier = oaiCfg.ServiceTier
		}
	}

	// Apply StructuredOutputConfig from Turn.Data (per-turn override).
	if t != nil {
		if soCfg, ok, err := engine.KeyStructuredOutputConfig.Get(t.Data); err == nil && ok && soCfg.IsEnabled() {
			if err := soCfg.Validate(); err == nil {
				strict := soCfg.StrictOrDefault()
				req.Text = &responsesText{
					Format: &responsesTextFormat{
						Type:        "json_schema",
						Name:        soCfg.Name,
						Description: soCfg.Description,
						Schema:      soCfg.Schema,
						Strict:      &strict,
					},
				}
			}
		}
	}

	// NOTE: stream_options.include_usage is not supported broadly; ignore for now
	return req, nil
}

// isResponsesReasoningModel returns true for models that do not accept
// temperature/top_p (o1/o3/o4/gpt-5 families).
func isResponsesReasoningModel(model string) bool {
	m := strings.ToLower(model)
	return strings.HasPrefix(m, "o1") ||
		strings.HasPrefix(m, "o3") ||
		strings.HasPrefix(m, "o4") ||
		strings.HasPrefix(m, "gpt-5")
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
		case turns.BlockKindLLMText, turns.BlockKindToolCall, turns.BlockKindReasoning, turns.BlockKindOther:
			return "assistant"
		}
		return "assistant"
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
				ctype := "input_text"
				if role == "assistant" {
					ctype = "output_text"
				}
				parts = append(parts, responsesContentPart{Type: ctype, Text: s})
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
			if itemID != "" {
				ri.ID = itemID
			}
			items = append(items, ri)
		}
	}
	appendFunctionCallOutput := func(b turns.Block) {
		toolID, _ := b.Payload[turns.PayloadKeyID].(string)
		resultJSON := toolUsePayloadToJSONString(b.Payload)
		if toolID != "" {
			// Responses expects call_id on function_call_output
			items = append(items, responsesInput{Type: "function_call_output", CallID: toolID, Output: resultJSON})
		}
	}

	// 1) Pre-context: all blocks before latest reasoning, skipping older reasoning.
	//    Additionally, ensure that for the LAST assistant text preceding latest reasoning we do NOT
	//    emit it as a role-based message, because the provider will expect the assistant text to be
	//    item-based and follow the reasoning item immediately. We will re-emit it later as an
	//    item-based message paired with reasoning.
	lastAssistantBeforeReasoning := -1
	if latestReasoningIdx > 0 {
		for i := latestReasoningIdx - 1; i >= 0; i-- {
			if t.Blocks[i].Kind == turns.BlockKindLLMText {
				lastAssistantBeforeReasoning = i
				break
			}
		}
		for i := 0; i < latestReasoningIdx; i++ {
			b := t.Blocks[i]
			switch b.Kind {
			case turns.BlockKindReasoning:
				continue
			case turns.BlockKindLLMText:
				// Skip the last assistant text before reasoning; it will become the follower.
				if i == lastAssistantBeforeReasoning {
					continue
				}
				appendMessage(b)
			case turns.BlockKindToolCall:
				appendFunctionCall(b)
			case turns.BlockKindToolUse:
				appendFunctionCallOutput(b)
			case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindOther:
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
			case turns.BlockKindUser, turns.BlockKindLLMText, turns.BlockKindSystem, turns.BlockKindReasoning, turns.BlockKindOther:
				appendMessage(b)
			}
		}
		return items
	}

	// 2) If a reasoning item is present, it must be immediately followed by an item-based follower
	//    according to the Responses API: either a type:"message" (assistant output) or a type:"function_call".
	//    We therefore emit reasoning only when we can include a valid follower right after it. When the
	//    follower is an assistant text block, we encode it as an item-based message and later skip the
	//    role-based rendering of that same block to avoid duplication.
	consumedAssistantIdx := -1
	includedReasoning := false
	groupEndIdx := latestReasoningIdx + 1
	if latestReasoningIdx >= 0 {
		nextIdx := latestReasoningIdx + 1
		if nextIdx < len(t.Blocks) {
			next := t.Blocks[nextIdx]
			switch next.Kind {
			case turns.BlockKindLLMText:
				// Emit reasoning + immediate item-based message follower
				rb := t.Blocks[latestReasoningIdx]
				enc, _ := rb.Payload[turns.PayloadKeyEncryptedContent].(string)
				empty := make([]any, 0)
				ri := responsesInput{Type: "reasoning", ID: rb.ID, Summary: &empty}
				if enc != "" {
					ri.EncryptedContent = enc
				}
				// Build item-based assistant message
				if v, ok := next.Payload[turns.PayloadKeyText]; ok && v != nil {
					if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
						msgID, _ := next.Payload[turns.PayloadKeyItemID].(string)
						items = append(items, ri)
						items = append(items, responsesInput{
							Type:    "message",
							Role:    "assistant",
							ID:      msgID,
							Content: []responsesContentPart{{Type: "output_text", Text: s}},
						})
						includedReasoning = true
						consumedAssistantIdx = nextIdx
						groupEndIdx = nextIdx + 1
					}
				}
			case turns.BlockKindToolCall:
				// Emit reasoning then group contiguous tool_call/tool_use items
				rb := t.Blocks[latestReasoningIdx]
				enc, _ := rb.Payload[turns.PayloadKeyEncryptedContent].(string)
				empty := make([]any, 0)
				ri := responsesInput{Type: "reasoning", ID: rb.ID, Summary: &empty}
				if enc != "" {
					ri.EncryptedContent = enc
				}
				items = append(items, ri)
				j := nextIdx
				for j < len(t.Blocks) {
					nb := t.Blocks[j]
					if nb.Kind == turns.BlockKindToolCall {
						appendFunctionCall(nb)
						j++
						continue
					}
					if nb.Kind == turns.BlockKindToolUse {
						appendFunctionCallOutput(nb)
						j++
						continue
					}
					break
				}
				includedReasoning = true
				groupEndIdx = j
			case turns.BlockKindToolUse:
				// No valid immediate follower when reasoning is followed directly by tool output.
				// We intentionally fall through to omit reasoning.
			case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindReasoning, turns.BlockKindOther:
				// No valid immediate follower; omit reasoning to avoid 400s.
			}
		}
	}

	// 3) Remaining blocks after the grouped segment (or after reasoning+message follower)
	startIdx := latestReasoningIdx + 1
	if includedReasoning {
		startIdx = groupEndIdx
	}
	for k := startIdx; k < len(t.Blocks); k++ {
		if k == consumedAssistantIdx {
			continue
		}
		b := t.Blocks[k]
		switch b.Kind {
		case turns.BlockKindReasoning:
			// skip any additional reasoning items
			continue
		case turns.BlockKindToolCall:
			// Avoid duplicating the tool_call we already grouped with reasoning.
			appendFunctionCall(b)
		case turns.BlockKindToolUse:
			appendFunctionCallOutput(b)
		case turns.BlockKindUser, turns.BlockKindLLMText, turns.BlockKindSystem, turns.BlockKindOther:
			appendMessage(b)
		}
	}

	return items
}

func toolUsePayloadToJSONString(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	resultVal := payload[turns.PayloadKeyResult]
	errStr, _ := payload[turns.PayloadKeyError].(string)
	if errStr == "" {
		return anyToJSONString(resultVal)
	}

	out := map[string]any{"error": errStr}
	if resultVal != nil {
		if s, ok := resultVal.(string); ok {
			var obj any
			if json.Unmarshal([]byte(s), &obj) == nil {
				out["result"] = obj
			} else {
				out["result"] = s
			}
		} else {
			out["result"] = resultVal
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, errStr)
	}
	return string(b)
}

func anyToJSONString(v any) string {
	if v == nil {
		return ""
	}
	switch tv := v.(type) {
	case string:
		return tv
	case []byte:
		return string(tv)
	default:
		if bb, err := json.Marshal(v); err == nil {
			return string(bb)
		}
		return fmt.Sprintf("%v", v)
	}
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
