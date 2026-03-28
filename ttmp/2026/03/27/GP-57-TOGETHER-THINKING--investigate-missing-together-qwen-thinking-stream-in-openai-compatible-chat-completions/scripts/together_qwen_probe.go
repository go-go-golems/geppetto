package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-go-golems/geppetto/pkg/events"
	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	stepopenai "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	gepsettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	openai "github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

type profileRegistry struct {
	Profiles map[string]profileEntry `yaml:"profiles"`
}

type profileEntry struct {
	Slug              string                        `yaml:"slug"`
	InferenceSettings gepsettings.InferenceSettings `yaml:"inference_settings"`
}

type capturingSink struct {
	mu     sync.Mutex
	events []events.Event
}

func (s *capturingSink) PublishEvent(event events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *capturingSink) Snapshot() []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]events.Event, len(s.events))
	copy(out, s.events)
	return out
}

func main() {
	var (
		mode        = flag.String("mode", "", "Experiment mode: raw-sse, go-openai, geppetto")
		profile     = flag.String("profile", "together-qwen-3.5-9b", "Profile slug to load")
		profilesYML = flag.String("profiles", filepath.Join(os.Getenv("HOME"), ".config", "pinocchio", "profiles.yaml"), "Path to profiles.yaml")
		prompt      = flag.String("prompt", "Solve 17 * 23 carefully and show your reasoning before the final answer.", "Prompt to send")
		maxTokens   = flag.Int("max-tokens", 512, "Maximum output tokens")
		temperature = flag.Float64("temperature", 1.0, "Sampling temperature")
		topP        = flag.Float64("top-p", 0.95, "Sampling top_p")
		topK        = flag.Int("top-k", 20, "Provider-native top_k")
		system      = flag.String("system", "", "Optional system prompt")
	)
	flag.Parse()

	if strings.TrimSpace(*mode) == "" {
		fmt.Fprintln(os.Stderr, "missing --mode (raw-sse|go-openai|geppetto)")
		os.Exit(2)
	}

	settings, err := loadProfile(*profilesYML, *profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load profile: %v\n", err)
		os.Exit(1)
	}

	switch *mode {
	case "raw-sse":
		err = runRawSSE(settings, *prompt, *system, *maxTokens, *temperature, *topP, *topK)
	case "go-openai":
		err = runGoOpenAI(settings, *prompt, *system, *maxTokens, *temperature, *topP)
	case "geppetto":
		err = runGeppetto(settings, *prompt, *system)
	default:
		err = fmt.Errorf("unknown mode %q", *mode)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "probe failed: %v\n", err)
		os.Exit(1)
	}
}

func loadProfile(path string, slug string) (*gepsettings.InferenceSettings, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var reg profileRegistry
	if err := yaml.NewDecoder(f).Decode(&reg); err != nil {
		return nil, err
	}
	entry, ok := reg.Profiles[slug]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in %s", slug, path)
	}

	// Normalize missing sub-sections to the same defaults Geppetto expects.
	s := entry.InferenceSettings
	if s.API == nil {
		s.API = gepsettings.NewAPISettings()
	}
	if s.Chat == nil {
		cs, err := gepsettings.NewChatSettings()
		if err != nil {
			return nil, err
		}
		s.Chat = cs
	}
	if s.Client == nil {
		s.Client = gepsettings.NewClientSettings()
	}
	if s.OpenAI == nil {
		osettings, err := gepsettings.NewInferenceSettings()
		if err != nil {
			return nil, err
		}
		s.OpenAI = osettings.OpenAI
	}
	return &s, nil
}

func apiKeyAndBaseURL(s *gepsettings.InferenceSettings) (string, string, error) {
	if s == nil || s.API == nil || s.Chat == nil || s.Chat.ApiType == nil {
		return "", "", fmt.Errorf("incomplete inference settings")
	}
	apiType := string(*s.Chat.ApiType)
	apiKey := s.API.APIKeys[apiType+"-api-key"]
	baseURL := s.API.BaseUrls[apiType+"-base-url"]
	if apiKey == "" || baseURL == "" {
		return "", "", fmt.Errorf("missing %s api key or base url", apiType)
	}
	return apiKey, baseURL, nil
}

func requestMessages(prompt string, system string) []map[string]any {
	msgs := make([]map[string]any, 0, 2)
	if strings.TrimSpace(system) != "" {
		msgs = append(msgs, map[string]any{
			"role":    "system",
			"content": system,
		})
	}
	msgs = append(msgs, map[string]any{
		"role":    "user",
		"content": prompt,
	})
	return msgs
}

func runRawSSE(s *gepsettings.InferenceSettings, prompt string, system string, maxTokens int, temperature float64, topP float64, topK int) error {
	apiKey, baseURL, err := apiKeyAndBaseURL(s)
	if err != nil {
		return err
	}
	model := ""
	if s.Chat != nil && s.Chat.Engine != nil {
		model = *s.Chat.Engine
	}
	if model == "" {
		return fmt.Errorf("missing model in chat settings")
	}

	body := map[string]any{
		"model":       model,
		"messages":    requestMessages(prompt, system),
		"stream":      true,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"top_p":       topP,
		"top_k":       topK,
		"chat_template_kwargs": map[string]any{
			"enable_thinking": true,
		},
	}
	payload, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("# raw-sse request\nPOST %s/chat/completions\n\n", strings.TrimRight(baseURL, "/"))
	fmt.Println(string(payload))
	fmt.Println("\n# raw-sse stream")

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client, err := gepsettings.EnsureHTTPClient(s.Client)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("# status %s\n", resp.Status)
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		fmt.Println(sc.Text())
	}
	return sc.Err()
}

func runGoOpenAI(s *gepsettings.InferenceSettings, prompt string, system string, maxTokens int, temperature float64, topP float64) error {
	apiKey, baseURL, err := apiKeyAndBaseURL(s)
	if err != nil {
		return err
	}
	model := ""
	if s.Chat != nil && s.Chat.Engine != nil {
		model = *s.Chat.Engine
	}
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	httpClient, err := gepsettings.EnsureHTTPClient(s.Client)
	if err != nil {
		return err
	}
	cfg.HTTPClient = httpClient
	client := openai.NewClientWithConfig(cfg)

	msgs := []openai.ChatCompletionMessage{}
	if strings.TrimSpace(system) != "" {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: system,
		})
	}
	msgs = append(msgs, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})

	req := openai.ChatCompletionRequest{
		Model:         model,
		Messages:      msgs,
		Stream:        true,
		MaxTokens:     maxTokens,
		Temperature:   float32(temperature),
		TopP:          float32(topP),
		StreamOptions: &openai.StreamOptions{IncludeUsage: true},
	}
	if payload, err := json.MarshalIndent(req, "", "  "); err == nil {
		fmt.Printf("# go-openai request body\n%s\n", string(payload))
	}

	fmt.Printf("# go-openai request\nmodel=%s baseURL=%s\n", model, baseURL)
	stream, err := client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		return err
	}
	defer stream.Close()

	var content, reasoning strings.Builder
	chunk := 0
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		chunk++
		if len(resp.Choices) == 0 {
			if resp.Usage != nil {
				reasoningTokens := 0
				if resp.Usage.CompletionTokensDetails != nil {
					reasoningTokens = resp.Usage.CompletionTokensDetails.ReasoningTokens
				}
				fmt.Printf("usage chunk: input=%d output=%d reasoning_tokens=%d\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens, reasoningTokens)
			}
			continue
		}
		d := resp.Choices[0].Delta
		if d.ReasoningContent != "" {
			reasoning.WriteString(d.ReasoningContent)
			fmt.Printf("chunk=%d reasoning=%q\n", chunk, d.ReasoningContent)
		}
		if d.Content != "" {
			content.WriteString(d.Content)
			fmt.Printf("chunk=%d content=%q\n", chunk, d.Content)
		}
		if d.Role != "" {
			fmt.Printf("chunk=%d role=%q\n", chunk, d.Role)
		}
		if resp.Choices[0].FinishReason != "" {
			fmt.Printf("finish_reason=%s\n", resp.Choices[0].FinishReason)
		}
	}

	fmt.Printf("\n# go-openai summary\nreasoning_len=%d\ncontent_len=%d\n", reasoning.Len(), content.Len())
	fmt.Printf("reasoning_text=%q\n", reasoning.String())
	fmt.Printf("content_text=%q\n", content.String())
	return nil
}

func runGeppetto(s *gepsettings.InferenceSettings, prompt string, system string) error {
	oae, err := stepopenai.NewOpenAIEngine(s)
	if err != nil {
		return err
	}
	eng, err := enginefactory.NewStandardEngineFactory().CreateEngine(s)
	if err != nil {
		return err
	}
	turn := &turns.Turn{}
	if strings.TrimSpace(system) != "" {
		turns.AppendBlock(turn, turns.NewSystemTextBlock(system))
	}
	turns.AppendBlock(turn, turns.NewUserTextBlock(prompt))
	req, err := oae.MakeCompletionRequestFromTurn(turn)
	if err != nil {
		return err
	}
	req.Stream = true
	if req.StreamOptions == nil && !strings.Contains(strings.ToLower(req.Model), "mistral") {
		req.StreamOptions = &stepopenai.ChatStreamOptions{IncludeUsage: true}
	}
	if payload, err := json.MarshalIndent(req, "", "  "); err == nil {
		fmt.Printf("# geppetto request body\n%s\n", string(payload))
	}

	sink := &capturingSink{}
	ctx := events.WithEventSinks(context.Background(), sink)

	out, err := eng.RunInference(ctx, turn)
	if err != nil {
		return err
	}

	fmt.Println("# geppetto events")
	for i, ev := range sink.Snapshot() {
		fmt.Printf("[%03d] type=%s", i, ev.Type())
		switch e := ev.(type) {
		case *events.EventThinkingPartial:
			fmt.Printf(" delta=%q cumulative=%q", e.Delta, e.Completion)
		case *events.EventPartialCompletion:
			fmt.Printf(" delta=%q cumulative=%q", e.Delta, e.Completion)
		case *events.EventFinal:
			fmt.Printf(" text=%q", e.Text)
		case *events.EventError:
			fmt.Printf(" error=%q", e.ErrorString)
		}
		if extra := ev.Metadata().Extra; len(extra) > 0 {
			if tt, ok := extra["thinking_text"].(string); ok && tt != "" {
				fmt.Printf(" thinking_text=%q", tt)
			}
		}
		fmt.Println()
	}

	fmt.Println("\n# geppetto blocks")
	for i, b := range out.Blocks {
		text := ""
		if t, ok := b.Payload[turns.PayloadKeyText].(string); ok {
			text = t
		}
		fmt.Printf("[%03d] kind=%s role=%s id=%s text=%q payload=%v\n", i, b.Kind, b.Role, b.ID, text, sortedPayloadKeys(b.Payload))
	}
	return nil
}

func sortedPayloadKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// Deterministic enough for quick debugging without importing sort? No.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
