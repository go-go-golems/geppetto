package openai_responses

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/imageparts"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// HTTP JSON models for an Open Responses-compatible API (minimal subset: text + reasoning + sampling)

type responsesRequest struct {
	Model             string           `json:"model"`
	Input             []responsesInput `json:"input"`
	Text              *responsesText   `json:"text,omitempty"`
	MaxOutputTokens   *int             `json:"max_output_tokens,omitempty"`
	Temperature       *float64         `json:"temperature,omitempty"`
	TopP              *float64         `json:"top_p,omitempty"`
	StopSequences     []string         `json:"stop_sequences,omitempty"`
	Reasoning         *reasoningParam  `json:"reasoning,omitempty"`
	Include           []string         `json:"include,omitempty"`
	Tools             []any            `json:"tools,omitempty"`
	ToolChoice        any              `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool            `json:"parallel_tool_calls,omitempty"`
	Stream            *bool            `json:"stream,omitempty"`
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
	// Common item fields. For provider-originated items, ID is the provider item
	// id from payload.item_id, never the local turns.Block.ID.
	Type   string `json:"type,omitempty"`
	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`

	// Message-style item fields.
	Role    string                 `json:"role,omitempty"`
	Content []responsesContentPart `json:"content,omitempty"`

	// Reasoning item fields.
	EncryptedContent string `json:"encrypted_content,omitempty"`
	Summary          *[]any `json:"summary,omitempty"`

	// function_call fields.
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// function_call_output fields.
	// Some providers expect call_id for both function_call and function_call_output.
	ToolCallID string `json:"tool_call_id,omitempty"`
	Output     string `json:"output,omitempty"`
}

type responsesContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// For input_image content type
	ImageURL string `json:"image_url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	Detail   string `json:"detail,omitempty"`
	// For function_call content type
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	// For tool_result content type
	ToolCallID string `json:"tool_call_id,omitempty"`
	Content    string `json:"content,omitempty"`
}

type responsesResponse struct {
	ID     string                `json:"id,omitempty"`
	Output []responsesOutputItem `json:"output"`
	Usage  json.RawMessage       `json:"usage,omitempty"`
	// Some envelopes may nest usage under response.usage.
	Response *responsesResponseNested `json:"response,omitempty"`
	// Error field intentionally omitted; HTTP non-2xx will carry error body
}

type responsesResponseNested struct {
	Usage json.RawMessage `json:"usage,omitempty"`
}

type responsesOutputItem struct {
	Type             string                   `json:"type,omitempty"`
	ID               string                   `json:"id,omitempty"`
	Name             string                   `json:"name,omitempty"`
	CallID           string                   `json:"call_id,omitempty"`
	Arguments        string                   `json:"arguments,omitempty"`
	Status           string                   `json:"status,omitempty"`
	Content          []responsesOutputContent `json:"content"`
	Summary          []any                    `json:"summary,omitempty"`
	EncryptedContent string                   `json:"encrypted_content,omitempty"`
}

type responsesOutputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	JSON any    `json:"json,omitempty"`
}

func redactResponsesID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if len(id) <= 12 {
		return id
	}
	return id[:6] + "…" + id[len(id)-4:]
}

func previewResponsesInputItem(it responsesInput) map[string]any {
	parts := make([]map[string]any, 0, len(it.Content))
	for _, c := range it.Content {
		seg := c.Text
		if len(seg) > 80 {
			seg = seg[:80] + "…"
		}
		parts = append(parts, map[string]any{"type": c.Type, "len": len(c.Text), "text": seg})
	}
	preview := map[string]any{
		"type":                  it.Type,
		"role":                  it.Role,
		"parts":                 parts,
		"has_encrypted_content": it.EncryptedContent != "",
		"encrypted_content_len": len(it.EncryptedContent),
	}
	if it.ID != "" {
		preview["id"] = redactResponsesID(it.ID)
	}
	if it.Status != "" {
		preview["status"] = it.Status
	}
	if it.Summary != nil {
		preview["summary_count"] = len(*it.Summary)
	}
	if it.CallID != "" {
		preview["call_id"] = redactResponsesID(it.CallID)
	}
	if it.Name != "" {
		preview["name"] = it.Name
	}
	if it.Output != "" {
		preview["output_len"] = len(it.Output)
	}
	return preview
}

func previewResponsesInput(items []responsesInput) []map[string]any {
	preview := make([]map[string]any, 0, len(items))
	for _, it := range items {
		preview = append(preview, previewResponsesInputItem(it))
	}
	return preview
}

// buildResponsesRequest constructs a minimal Responses request from Turn + settings
func (e *Engine) buildResponsesRequest(t *turns.Turn) (responsesRequest, error) {
	s := e.settings
	req := responsesRequest{}
	if s != nil && s.Chat != nil && s.Chat.Engine != nil {
		req.Model = *s.Chat.Engine
	}
	req.Input = buildInputItemsFromTurn(t)
	if s != nil && s.Chat != nil {
		if s.Chat.MaxResponseTokens != nil {
			req.MaxOutputTokens = s.Chat.MaxResponseTokens
		}
		// Some reasoning models do not accept temperature/top_p; omit for those.
		allowSampling := !isResponsesReasoningModelForSettings(s, req.Model)
		if allowSampling && s.Chat.Temperature != nil {
			req.Temperature = s.Chat.Temperature
		}
		if allowSampling && s.Chat.TopP != nil {
			req.TopP = s.Chat.TopP
		}
		if len(s.Chat.Stop) > 0 {
			req.StopSequences = s.Chat.Stop
		}
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
			log.Warn().Err(err).Msg("Responses request: ignoring invalid structured output configuration")
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
	// Apply per-turn InferenceConfig overrides (Turn.Data > InferenceSettings.Inference).
	var engineInference *engine.InferenceConfig
	if s != nil {
		engineInference = s.Inference
	}
	if infCfg := engine.ResolveInferenceConfig(t, engineInference); infCfg != nil {
		// Reasoning models reject temperature/top_p; sanitize upfront.
		if isResponsesReasoningModelForSettings(s, req.Model) {
			infCfg = engine.SanitizeForReasoningModel(infCfg)
		}
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
		if infCfg.Temperature != nil {
			req.Temperature = infCfg.Temperature
		}
		if infCfg.TopP != nil {
			req.TopP = infCfg.TopP
		}
		if infCfg.MaxResponseTokens != nil {
			req.MaxOutputTokens = infCfg.MaxResponseTokens
		}
		if infCfg.Stop != nil {
			req.StopSequences = infCfg.Stop
		}
	}

	// Apply current OpenAI-specific per-turn overrides from Turn.Data.
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

func isResponsesReasoningModelForSettings(s *settings.InferenceSettings, model string) bool {
	if s != nil && s.ModelInfo != nil && s.ModelInfo.Reasoning != nil {
		return *s.ModelInfo.Reasoning
	}
	return isResponsesReasoningModel(model)
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

func reasoningSummaryEntriesFromText(text string) []any {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	return []any{map[string]any{
		"type": "summary_text",
		"text": text,
	}}
}

func setOpenAIResponsesBlockMetadata(b *turns.Block, responseID string, outputIndex *int, itemType string, status string) {
	if b == nil {
		return
	}
	if strings.TrimSpace(responseID) != "" {
		_ = keyOpenAIResponsesResponseID.Set(&b.Metadata, responseID)
	}
	if outputIndex != nil {
		_ = keyOpenAIResponsesOutputIndex.Set(&b.Metadata, *outputIndex)
	}
	if strings.TrimSpace(itemType) != "" {
		_ = keyOpenAIResponsesItemType.Set(&b.Metadata, itemType)
	}
	if strings.TrimSpace(status) != "" {
		_ = keyOpenAIResponsesStatus.Set(&b.Metadata, status)
	}
}

func intFromProviderNumber(raw any) (int, bool) {
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i), true
		}
	}
	return 0, false
}

func reasoningTextFromProviderContent(raw any) string {
	items, ok := raw.([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		if typ != "reasoning_text" && typ != "text" {
			continue
		}
		if s, ok := m["text"].(string); ok && strings.TrimSpace(s) != "" {
			b.WriteString(s)
		}
	}
	return strings.TrimSpace(b.String())
}

func reasoningSummaryEntriesFromPayload(payload map[string]any) []any {
	if payload == nil {
		return nil
	}
	if raw, ok := payload[turns.PayloadKeySummary]; ok {
		switch tv := raw.(type) {
		case []any:
			return append([]any(nil), tv...)
		case []map[string]any:
			ret := make([]any, 0, len(tv))
			for _, item := range tv {
				ret = append(ret, item)
			}
			return ret
		case string:
			return reasoningSummaryEntriesFromText(tv)
		}
	}
	return nil
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

	// Helpers
	appendMessage := func(b turns.Block) {
		role := roleFor(b.Kind)
		parts := buildResponsesMessageParts(role, b.Payload)
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

	reasoningItem := func(b turns.Block) (responsesInput, bool) {
		enc, _ := b.Payload[turns.PayloadKeyEncryptedContent].(string)
		summary := reasoningSummaryEntriesFromPayload(b.Payload)
		itemID, _ := b.Payload[turns.PayloadKeyItemID].(string)

		// OpenAI's public schema exposes optional reasoning_text content on reasoning
		// items, but live Responses requests currently reject non-empty reasoning
		// input content ("expected maximum length 0"). Preserve plaintext reasoning
		// locally for UI/debugging, but replay only encrypted_content and summaries.
		if enc == "" && len(summary) == 0 {
			return responsesInput{}, false
		}

		if summary == nil {
			summary = make([]any, 0)
		}
		ri := responsesInput{Type: "reasoning", Summary: &summary}
		// Provider item IDs are replay payload, not internal block identity. Use
		// the explicit item_id captured from the provider event when available;
		// never infer it from Block.ID, which may be a synthetic UUID or may follow
		// another provider's ID scheme.
		if strings.TrimSpace(itemID) != "" {
			ri.ID = itemID
		}
		if enc != "" {
			ri.EncryptedContent = enc
		}
		return ri, true
	}

	// Process blocks in-order so every function_call can retain its required reasoning predecessor.
	for i := 0; i < len(t.Blocks); i++ {
		b := t.Blocks[i]
		switch b.Kind {
		case turns.BlockKindReasoning:
			nextIdx := i + 1
			if nextIdx >= len(t.Blocks) {
				continue
			}
			next := t.Blocks[nextIdx]
			switch next.Kind {
			case turns.BlockKindLLMText:
				if v, ok := next.Payload[turns.PayloadKeyText]; ok && v != nil {
					if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
						msgID, _ := next.Payload[turns.PayloadKeyItemID].(string)
						if ri, ok := reasoningItem(b); ok {
							items = append(items, ri)
						}
						items = append(items, responsesInput{
							Type:    "message",
							Role:    "assistant",
							ID:      msgID,
							Content: []responsesContentPart{{Type: "output_text", Text: s}},
						})
						i = nextIdx
						continue
					}
				}
			case turns.BlockKindToolCall:
				if ri, ok := reasoningItem(b); ok {
					items = append(items, ri)
				}
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
				i = j - 1
				continue
			case turns.BlockKindToolUse:
				// No valid immediate follower when reasoning is followed directly by tool output.
				// Omit reasoning to avoid provider 400s.
				continue
			case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindReasoning, turns.BlockKindOther:
				// No valid immediate follower; omit reasoning to avoid provider 400s.
				continue
			}
		case turns.BlockKindToolCall:
			appendFunctionCall(b)
		case turns.BlockKindToolUse:
			appendFunctionCallOutput(b)
		case turns.BlockKindUser, turns.BlockKindLLMText, turns.BlockKindSystem, turns.BlockKindOther:
			appendMessage(b)
		}
	}

	return items
}

func buildResponsesMessageParts(role string, payload map[string]any) []responsesContentPart {
	var parts []responsesContentPart
	if payload == nil {
		return parts
	}
	if v, ok := payload[turns.PayloadKeyText]; ok && v != nil {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			ctype := "input_text"
			if role == "assistant" {
				ctype = "output_text"
			}
			parts = append(parts, responsesContentPart{Type: ctype, Text: s})
		}
	}
	if role == "assistant" {
		return parts
	}
	if imgs, ok := payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
		for _, img := range imgs {
			if part, ok := responsesImagePartFromMap(img); ok {
				parts = append(parts, part)
			}
		}
	}
	return parts
}

func responsesImagePartFromMap(img map[string]any) (responsesContentPart, bool) {
	part, ok, err := imageparts.NormalizeImageMap(img)
	if err != nil || !ok {
		return responsesContentPart{}, false
	}
	if part.URL != "" {
		return responsesContentPart{Type: "input_image", ImageURL: part.URL, Detail: part.Detail}, true
	}
	if len(part.Data) > 0 {
		return responsesContentPart{Type: "input_image", ImageURL: imageparts.DataURL(part.MediaType, part.Data), Detail: part.Detail}, true
	}
	if part.FileID != "" {
		return responsesContentPart{Type: "input_image", FileID: part.FileID, Detail: part.Detail}, true
	}
	return responsesContentPart{}, false
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
