package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	gepevents "github.com/go-go-golems/geppetto/pkg/events"
	gepengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	geptools "github.com/go-go-golems/geppetto/pkg/inference/tools"
	gepsettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"gopkg.in/yaml.v3"
)

type eventSink struct {
	events []map[string]any
}

func (s *eventSink) PublishEvent(ev gepevents.Event) error {
	row := map[string]any{
		"type": string(ev.Type()),
		"meta": ev.Metadata(),
	}
	if b, err := json.Marshal(ev); err == nil {
		var payload any
		if json.Unmarshal(b, &payload) == nil {
			row["event"] = payload
		} else {
			row["event_json"] = string(b)
		}
	} else {
		row["marshal_error"] = err.Error()
	}
	s.events = append(s.events, row)
	return nil
}

type smokeSummary struct {
	Case            string            `json:"case"`
	Profile         string            `json:"profile"`
	Model           string            `json:"model"`
	OK              bool              `json:"ok"`
	Skipped         bool              `json:"skipped,omitempty"`
	SkipReason      string            `json:"skip_reason,omitempty"`
	Error           string            `json:"error,omitempty"`
	StartedAt       string            `json:"started_at"`
	DurationMS      int64             `json:"duration_ms"`
	EventCounts     map[string]int    `json:"event_counts"`
	FinalBlockKinds []string          `json:"final_block_kinds"`
	ToolCallIDs     []string          `json:"tool_call_ids,omitempty"`
	InferenceResult any               `json:"inference_result,omitempty"`
	Artifacts       map[string]string `json:"artifacts"`
	Interpretation  []string          `json:"interpretation,omitempty"`
}

func main() {
	caseName := flag.String("case", "plain-text", "smoke case: plain-text, tool-call, tool-loop, visible-thinking")
	profile := flag.String("profile", "gemini-2.5-flash", "Geppetto engine profile slug to resolve from the profile registry")
	profileRegistries := flag.String("profile-registries", strings.Join(defaultProfileRegistrySources(), ","), "comma-separated Geppetto profile registry sources")
	modelOverride := flag.String("model", "", "optional model override after profile resolution; normally leave empty so the profile owns the model")
	includeThoughts := flag.Bool("include-thoughts", false, "request Gemini thought parts when supported by the model")
	thinkingBudget := flag.Int("thinking-budget", 0, "optional Gemini thinking budget in tokens when supported by the model")
	thinkingLevel := flag.String("thinking-level", "", "optional Gemini thinking level (MINIMAL, LOW, MEDIUM, HIGH)")
	outDir := flag.String("out-dir", "", "artifact output directory")
	prompt := flag.String("prompt", "Reply with exactly: gemini smoke ok", "prompt for plain-text case")
	flag.Parse()

	if *outDir == "" {
		*outDir = filepath.Join("ttmp", "2026", "06", "05", "2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash", "scripts", "artifacts")
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	artifactNameSuffix := *profile
	if strings.TrimSpace(*modelOverride) != "" {
		artifactNameSuffix += "-" + strings.TrimSpace(*modelOverride)
	}
	name := safeName(*caseName + "-" + artifactNameSuffix)
	summaryPath := filepath.Join(*outDir, name+"-summary.json")
	eventsPath := filepath.Join(*outDir, name+"-events.ndjson")
	turnPath := filepath.Join(*outDir, name+"-turn.yaml")
	resultPath := filepath.Join(*outDir, name+"-inference-result.json")

	started := time.Now()
	summary := smokeSummary{
		Case:        *caseName,
		Profile:     *profile,
		StartedAt:   started.Format(time.RFC3339Nano),
		EventCounts: map[string]int{},
		Artifacts: map[string]string{
			"summary":          summaryPath,
			"events_ndjson":    eventsPath,
			"turn_yaml":        turnPath,
			"inference_result": resultPath,
		},
	}

	baseCtx := context.Background()
	settings, resolvedProfile, closeRegistry, err := resolveProfileSettings(baseCtx, *profileRegistries, *profile, *modelOverride, *includeThoughts, *thinkingBudget, *thinkingLevel)
	if err != nil {
		summary.Error = err.Error()
		summary.DurationMS = time.Since(started).Milliseconds()
		writeJSON(summaryPath, summary)
		fmt.Printf("FAIL: resolve profile %q: %s\nsummary: %s\n", *profile, summary.Error, summaryPath)
		os.Exit(1)
	}
	defer closeRegistry()
	if settings.Chat != nil && settings.Chat.Engine != nil {
		summary.Model = *settings.Chat.Engine
	}
	summary.Profile = resolvedProfile

	sink := &eventSink{}
	ctx := gepevents.WithEventSinks(baseCtx, sink)

	registry, err := buildRegistry()
	if err != nil {
		fatal(err)
	}
	if *caseName == "tool-call" || *caseName == "tool-loop" {
		ctx = geptools.WithRegistry(ctx, registry)
	}

	eng, err := newGeminiEngine(settings)
	if err != nil {
		fatal(err)
	}

	turn := buildTurn(*caseName, *prompt)
	var result *gepengine.InferenceResult
	turn, result, err = gepengine.RunInferenceWithResult(ctx, eng, turn)
	if err == nil && *caseName == "tool-loop" {
		turn, err = appendSyntheticToolResult(turn)
		if err == nil {
			turn, result, err = gepengine.RunInferenceWithResult(ctx, eng, turn)
		}
	}

	if err != nil {
		summary.Error = err.Error()
	} else {
		summary.OK = true
	}
	if result != nil {
		summary.InferenceResult = result
		writeJSON(resultPath, result)
	}
	summary.DurationMS = time.Since(started).Milliseconds()
	summary.EventCounts = countEvents(sink.events)
	summary.FinalBlockKinds = blockKinds(turn)
	summary.ToolCallIDs = toolCallIDs(turn)
	summary.Interpretation = interpret(*caseName, summary)

	writeEvents(eventsPath, sink.events)
	writeYAML(turnPath, turn)
	writeJSON(summaryPath, summary)

	if summary.OK {
		fmt.Printf("OK: %s on %s\nsummary: %s\n", summary.Case, summary.Model, summaryPath)
	} else {
		fmt.Printf("FAIL: %s on %s: %s\nsummary: %s\n", summary.Case, summary.Model, summary.Error, summaryPath)
		os.Exit(1)
	}
}

func resolveProfileSettings(ctx context.Context, registrySources string, profileSlug string, modelOverride string, includeThoughts bool, thinkingBudget int, thinkingLevel string) (*gepsettings.InferenceSettings, string, func(), error) {
	entries, err := gepprofiles.ParseEngineProfileRegistrySourceEntries(registrySources)
	if err != nil {
		return nil, "", func() {}, err
	}
	if len(entries) == 0 {
		return nil, "", func() {}, errors.New("profile registries are required; pass --profile-registries pointing at the Geppetto/Pinocchio profiles YAML")
	}
	specs, err := gepprofiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		return nil, "", func() {}, err
	}
	chain, err := gepprofiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return nil, "", func() {}, err
	}
	closeFn := func() { _ = chain.Close() }

	parsedProfileSlug, err := gepprofiles.ParseEngineProfileSlug(profileSlug)
	if err != nil {
		closeFn()
		return nil, "", func() {}, err
	}
	resolved, err := chain.ResolveEngineProfile(ctx, gepprofiles.ResolveInput{EngineProfileSlug: parsedProfileSlug})
	if err != nil {
		closeFn()
		return nil, "", func() {}, err
	}
	base, err := gepsettings.NewInferenceSettings()
	if err != nil {
		closeFn()
		return nil, "", func() {}, err
	}
	settings, err := gepprofiles.MergeInferenceSettings(base, resolved.InferenceSettings)
	if err != nil {
		closeFn()
		return nil, "", func() {}, err
	}
	if modelOverride != "" {
		settings.Chat.Engine = &modelOverride
	}
	if settings.Gemini != nil {
		if includeThoughts {
			settings.Gemini.IncludeThoughts = &includeThoughts
		}
		if thinkingBudget != 0 {
			settings.Gemini.ThinkingBudget = &thinkingBudget
		}
		if strings.TrimSpace(thinkingLevel) != "" {
			settings.Gemini.ThinkingLevel = strings.TrimSpace(thinkingLevel)
		}
	}
	settings.Chat.Stream = true
	return settings, resolved.EngineProfileSlug.String(), closeFn, nil
}

func newGeminiEngine(settings *gepsettings.InferenceSettings) (gepengine.Engine, error) {
	if settings == nil {
		return nil, errors.New("resolved inference settings are nil")
	}
	return enginefactory.NewEngineFromSettings(settings)
}

func buildTurn(caseName, prompt string) *turns.Turn {
	system := "You are a terse Geppetto Gemini smoke-test assistant. If a tool is useful, call it instead of guessing."
	switch caseName {
	case "plain-text":
		return turns.NewTurnBuilder().WithSystemPrompt(system).WithUserPrompt(prompt).Build()
	case "visible-thinking":
		return turns.NewTurnBuilder().WithSystemPrompt(system).WithUserPrompt("Think carefully, then answer with exactly: visible thinking smoke ok").Build()
	case "tool-call", "tool-loop":
		return turns.NewTurnBuilder().WithSystemPrompt(system).WithUserPrompt("What is the weather in Zurich? Use the lookup_weather tool.").Build()
	default:
		return turns.NewTurnBuilder().WithSystemPrompt(system).WithUserPrompt(prompt).Build()
	}
}

type weatherArgs struct {
	Location string `json:"location" jsonschema:"description=City or place name"`
}

func buildRegistry() (geptools.ToolRegistry, error) {
	registry := geptools.NewInMemoryToolRegistry()
	weather, err := geptools.NewToolFromFunc("lookup_weather", "Return deterministic fake weather for a city.", func(args weatherArgs) (map[string]any, error) {
		location := strings.TrimSpace(args.Location)
		if location == "" {
			location = "unknown"
		}
		return map[string]any{
			"location":    location,
			"condition":   "clear",
			"temperature": "21 C",
			"source":      "local smoke fixture",
		}, nil
	})
	if err != nil {
		return nil, err
	}
	if err := registry.RegisterTool(weather.Name, *weather); err != nil {
		return nil, err
	}
	return registry, nil
}

func appendSyntheticToolResult(t *turns.Turn) (*turns.Turn, error) {
	if t == nil {
		return t, errors.New("nil turn")
	}
	for i := len(t.Blocks) - 1; i >= 0; i-- {
		b := t.Blocks[i]
		if b.Kind != turns.BlockKindToolCall {
			continue
		}
		id, _ := b.Payload[turns.PayloadKeyID].(string)
		if id == "" {
			id = b.ID
		}
		if id == "" {
			return t, errors.New("Gemini tool call block did not contain a usable id")
		}
		turns.AppendBlock(t, turns.NewToolUseBlock(id, map[string]any{
			"location":    "Zurich",
			"condition":   "clear",
			"temperature": "21 C",
			"source":      "client-side smoke test result",
		}))
		turns.AppendBlock(t, turns.NewUserTextBlock("Please summarize the tool result in one sentence."))
		return t, nil
	}
	return t, errors.New("Gemini did not produce a tool call block to continue")
}

func countEvents(events []map[string]any) map[string]int {
	counts := map[string]int{}
	for _, ev := range events {
		if typ, ok := ev["type"].(string); ok && typ != "" {
			counts[typ]++
		}
	}
	return counts
}

func blockKinds(t *turns.Turn) []string {
	if t == nil {
		return nil
	}
	out := make([]string, 0, len(t.Blocks))
	for _, b := range t.Blocks {
		out = append(out, b.Kind.String())
	}
	return out
}

func toolCallIDs(t *turns.Turn) []string {
	if t == nil {
		return nil
	}
	var out []string
	for _, b := range t.Blocks {
		if b.Kind != turns.BlockKindToolCall {
			continue
		}
		id, _ := b.Payload[turns.PayloadKeyID].(string)
		if id == "" {
			id = b.ID
		}
		out = append(out, id)
	}
	return out
}

func interpret(caseName string, s smokeSummary) []string {
	var out []string
	if s.Error != "" {
		out = append(out, "provider or Geppetto returned an error; inspect turn_yaml and events_ndjson")
		return out
	}
	if caseName == "tool-call" || caseName == "tool-loop" {
		if len(s.ToolCallIDs) == 0 {
			out = append(out, "no tool_call blocks were captured; this is a Gemini provider/tool advertisement gap")
		} else {
			out = append(out, fmt.Sprintf("captured %d tool_call id(s): %s", len(s.ToolCallIDs), strings.Join(s.ToolCallIDs, ", ")))
		}
	}
	if s.EventCounts["tool-call-requested"] == 0 && (caseName == "tool-call" || caseName == "tool-loop") {
		out = append(out, "canonical tool-call-requested event count is zero; compare block output against stream reducer behavior")
	}
	return out
}

func writeEvents(path string, events []map[string]any) {
	f, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			fatal(err)
		}
	}
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		fatal(err)
	}
}

func writeYAML(path string, v any) {
	b, err := yaml.Marshal(v)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fatal(err)
	}
}

func defaultProfileRegistrySources() []string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".config", "pinocchio", "profiles.yaml"),
		filepath.Join(home, ".pinocchio", "config", "profiles.yaml"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return []string{candidate}
		}
	}
	return nil
}

func safeName(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
