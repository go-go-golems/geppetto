package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type chatStreamConfig struct {
	baseURL    string
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

type chatStreamUsage struct {
	promptTokens     int
	completionTokens int
	cachedTokens     int
	reasoningTokens  int
}

type chatStreamEvent struct {
	DeltaText      string
	DeltaReasoning string
	ToolCalls      []ChatToolCall
	Usage          *chatStreamUsage
	FinishReason   *string
	RawPayload     map[string]any
}

type sseFrame struct {
	Event string
	Data  string
}

type chatCompletionStream struct {
	resp   *http.Response
	reader *bufio.Reader
}

func resolveChatStreamConfig(
	apiSettings *settings.APISettings,
	clientSettings *settings.ClientSettings,
	apiType ai_types.ApiType,
) (chatStreamConfig, error) {
	apiKey, ok := apiSettings.APIKeys[string(apiType)+"-api-key"]
	if !ok || strings.TrimSpace(apiKey) == "" {
		return chatStreamConfig{}, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[string(apiType)+"-base-url"]
	if !ok || strings.TrimSpace(baseURL) == "" {
		return chatStreamConfig{}, errors.Errorf("no base URL for %s", apiType)
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	if err := security.ValidateOutboundURL(endpoint, security.OutboundURLOptions{
		AllowHTTP: false,
	}); err != nil {
		return chatStreamConfig{}, errors.Wrap(err, "invalid chat completion URL")
	}
	httpClient, err := settings.EnsureHTTPClient(clientSettings)
	if err != nil {
		return chatStreamConfig{}, err
	}
	return chatStreamConfig{
		baseURL:    baseURL,
		endpoint:   endpoint,
		apiKey:     apiKey,
		httpClient: httpClient,
	}, nil
}

func openChatCompletionStream(
	ctx context.Context,
	cfg chatStreamConfig,
	reqBody any,
) (*chatCompletionStream, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "marshal chat completion request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "create chat completion request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if strings.TrimSpace(cfg.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	}

	resp, err := cfg.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		hint := chatCompletionEndpointHint(resp.StatusCode, cfg)
		if hint != "" {
			log.Warn().
				Int("status", resp.StatusCode).
				Str("base_url", cfg.baseURL).
				Str("endpoint", cfg.endpoint).
				Msg("OpenAI-compatible chat completions request failed with suspicious base URL")
		}
		if len(raw) == 0 {
			return nil, fmt.Errorf("chat completions error: status=%d%s", resp.StatusCode, hint)
		}
		return nil, fmt.Errorf("chat completions error: status=%d body=%s%s", resp.StatusCode, strings.TrimSpace(string(raw)), hint)
	}

	return &chatCompletionStream{
		resp:   resp,
		reader: bufio.NewReader(resp.Body),
	}, nil
}

func chatCompletionEndpointHint(statusCode int, cfg chatStreamConfig) string {
	if statusCode != http.StatusNotFound {
		return ""
	}
	reason, ok := suspiciousChatCompletionBaseURLReason(cfg.baseURL)
	if !ok {
		return ""
	}
	endpoint := cfg.endpoint
	if parsed, err := url.Parse(endpoint); err == nil {
		endpoint = parsed.Redacted()
	}
	return fmt.Sprintf(
		"; possible OpenAI-compatible base URL misconfiguration: %s; Geppetto appends /chat/completions internally (computed endpoint: %s)",
		reason,
		endpoint,
	)
}

func suspiciousChatCompletionBaseURLReason(baseURL string) (string, bool) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", false
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	path := strings.TrimRight(strings.ToLower(parsed.EscapedPath()), "/")
	if path == "" || path == "/" {
		return "", false
	}
	if strings.Contains(path, "/chat/completion") {
		return fmt.Sprintf("configured base URL path %q already looks like a chat completions endpoint", parsed.EscapedPath()), true
	}
	for _, versionPrefix := range []string{"/v1/", "/v2/"} {
		if strings.HasPrefix(path, versionPrefix) {
			return fmt.Sprintf("configured base URL path %q has extra path components after %s; expected provider root like https://host%s", parsed.EscapedPath(), strings.TrimSuffix(versionPrefix, "/"), strings.TrimSuffix(versionPrefix, "/")), true
		}
	}
	return "", false
}

func (s *chatCompletionStream) Close() error {
	if s == nil || s.resp == nil || s.resp.Body == nil {
		return nil
	}
	return s.resp.Body.Close()
}

func (s *chatCompletionStream) Recv() (chatStreamEvent, error) {
	for {
		frame, err := readSSEFrame(s.reader)
		if err != nil {
			return chatStreamEvent{}, err
		}
		if strings.TrimSpace(frame.Data) == "" {
			continue
		}
		if strings.TrimSpace(frame.Data) == "[DONE]" {
			return chatStreamEvent{}, io.EOF
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(frame.Data), &raw); err != nil {
			return chatStreamEvent{}, errors.Wrap(err, "decode chat stream event")
		}
		return normalizeChatStreamEvent(raw), nil
	}
}

func readSSEFrame(reader *bufio.Reader) (sseFrame, error) {
	var frame sseFrame
	var haveContent bool

	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return sseFrame{}, err
		}
		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if haveContent {
				return frame, nil
			}
			if errors.Is(err, io.EOF) {
				return sseFrame{}, io.EOF
			}
			continue
		}

		switch {
		case strings.HasPrefix(line, ":"):
			// Comment/heartbeat frame. Ignore.
		case strings.HasPrefix(line, "event:"):
			frame.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			haveContent = true
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if frame.Data != "" {
				frame.Data += "\n"
			}
			frame.Data += data
			haveContent = true
		}

		if errors.Is(err, io.EOF) {
			if haveContent {
				return frame, nil
			}
			return sseFrame{}, io.EOF
		}
	}
}

func normalizeChatStreamEvent(raw map[string]any) chatStreamEvent {
	ret := chatStreamEvent{RawPayload: raw}
	choice := firstChoice(raw)
	delta := mapValue(choice["delta"])

	if s, ok := stringValue(delta["content"]); ok {
		ret.DeltaText = s
	}
	if s, ok := stringValue(delta["reasoning"]); ok && s != "" {
		ret.DeltaReasoning = s
	} else if s, ok := stringValue(delta["reasoning_content"]); ok && s != "" {
		ret.DeltaReasoning = s
	}
	ret.ToolCalls = normalizeChatToolCalls(delta["tool_calls"])
	if usage := normalizeChatUsage(raw["usage"]); usage != nil {
		ret.Usage = usage
	}
	if s, ok := stringValue(choice["finish_reason"]); ok && s != "" {
		ret.FinishReason = &s
	}
	return ret
}

func firstChoice(raw map[string]any) map[string]any {
	choices, ok := raw["choices"].([]any)
	if !ok || len(choices) == 0 {
		return nil
	}
	return mapValue(choices[0])
}

func normalizeChatToolCalls(v any) []ChatToolCall {
	items, ok := v.([]any)
	if !ok || len(items) == 0 {
		return nil
	}

	ret := make([]ChatToolCall, 0, len(items))
	for _, item := range items {
		callMap := mapValue(item)
		if len(callMap) == 0 {
			continue
		}
		call := ChatToolCall{}
		if idx, ok := intValue(callMap["index"]); ok {
			call.Index = &idx
		}
		if id, ok := stringValue(callMap["id"]); ok {
			call.ID = id
		}
		if typ, ok := stringValue(callMap["type"]); ok && typ != "" {
			call.Type = typ
		} else {
			call.Type = chatToolTypeFunction
		}
		fn := mapValue(callMap["function"])
		if name, ok := stringValue(fn["name"]); ok {
			call.Function.Name = name
		}
		if args, ok := stringValue(fn["arguments"]); ok {
			call.Function.Arguments = args
		}
		ret = append(ret, call)
	}

	if len(ret) == 0 {
		return nil
	}
	return ret
}

func normalizeChatUsage(v any) *chatStreamUsage {
	usageMap := mapValue(v)
	if len(usageMap) == 0 {
		return nil
	}
	ret := &chatStreamUsage{}
	if n, ok := intValue(usageMap["prompt_tokens"]); ok {
		ret.promptTokens = n
	}
	if n, ok := intValue(usageMap["completion_tokens"]); ok {
		ret.completionTokens = n
	}
	if n, ok := intValue(mapValue(usageMap["prompt_tokens_details"])["cached_tokens"]); ok {
		ret.cachedTokens = n
	}
	outputDetails := mapValue(usageMap["completion_tokens_details"])
	if n, ok := intValue(outputDetails["reasoning_tokens"]); ok {
		ret.reasoningTokens = n
	} else if n, ok := intValue(usageMap["reasoning_tokens"]); ok {
		ret.reasoningTokens = n
	}
	return ret
}

func mapValue(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func stringValue(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func intValue(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}
