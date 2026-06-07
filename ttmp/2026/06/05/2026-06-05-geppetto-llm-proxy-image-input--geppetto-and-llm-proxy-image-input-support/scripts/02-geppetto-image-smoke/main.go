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
	gepsettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"gopkg.in/yaml.v3"
)

const tinyPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="

type eventSink struct{ events []map[string]any }

func (s *eventSink) PublishEvent(ev gepevents.Event) error {
	row := map[string]any{"type": string(ev.Type()), "meta": ev.Metadata()}
	if b, err := json.Marshal(ev); err == nil {
		var payload any
		if json.Unmarshal(b, &payload) == nil {
			row["event"] = payload
		}
	}
	s.events = append(s.events, row)
	return nil
}

type summary struct {
	Profile         string            `json:"profile"`
	Model           string            `json:"model"`
	OK              bool              `json:"ok"`
	Error           string            `json:"error,omitempty"`
	StartedAt       string            `json:"started_at"`
	DurationMS      int64             `json:"duration_ms"`
	EventCounts     map[string]int    `json:"event_counts"`
	FinalBlockKinds []string          `json:"final_block_kinds"`
	InferenceResult any               `json:"inference_result,omitempty"`
	Artifacts       map[string]string `json:"artifacts"`
}

func main() {
	profile := flag.String("profile", "gemini-3-flash-preview", "Geppetto profile slug")
	profileRegistries := flag.String("profile-registries", strings.Join(defaultProfileRegistrySources(), ","), "comma-separated profile registries")
	outDir := flag.String("out-dir", filepath.Join("ttmp", "2026", "06", "05", "2026-06-05-geppetto-llm-proxy-image-input--geppetto-and-llm-proxy-image-input-support", "scripts", "artifacts"), "artifact output directory")
	flag.Parse()
	_ = os.MkdirAll(*outDir, 0o755)
	name := "geppetto-image-" + safeName(*profile)
	summaryPath := filepath.Join(*outDir, name+"-summary.json")
	eventsPath := filepath.Join(*outDir, name+"-events.ndjson")
	turnPath := filepath.Join(*outDir, name+"-turn.yaml")
	resultPath := filepath.Join(*outDir, name+"-inference-result.json")
	started := time.Now()
	s := summary{Profile: *profile, StartedAt: started.Format(time.RFC3339Nano), EventCounts: map[string]int{}, Artifacts: map[string]string{"summary": summaryPath, "events_ndjson": eventsPath, "turn_yaml": turnPath, "inference_result": resultPath}}

	settings, resolved, closeFn, err := resolveProfileSettings(context.Background(), *profileRegistries, *profile)
	if err != nil {
		finish(summaryPath, s, err, started)
	}
	defer closeFn()
	s.Profile = resolved
	if settings.Chat != nil && settings.Chat.Engine != nil {
		s.Model = *settings.Chat.Engine
	}
	sink := &eventSink{}
	ctx := gepevents.WithEventSinks(context.Background(), sink)
	eng, err := enginefactory.NewEngineFromSettings(settings)
	if err != nil {
		finish(summaryPath, s, err, started)
	}
	turn := &turns.Turn{ID: "image-smoke-turn"}
	turns.AppendBlock(turn, turns.NewUserMultimodalBlock("You are given an image. Reply exactly: image smoke ok", []map[string]any{{"content": tinyPNGDataURL}}))
	turn, result, err := gepengine.RunInferenceWithResult(ctx, eng, turn)
	if err != nil {
		s.Error = err.Error()
	} else {
		s.OK = true
	}
	if result != nil {
		s.InferenceResult = result
		writeJSON(resultPath, result)
	}
	s.DurationMS = time.Since(started).Milliseconds()
	s.EventCounts = countEvents(sink.events)
	s.FinalBlockKinds = blockKinds(turn)
	writeEvents(eventsPath, sink.events)
	writeYAML(turnPath, turn)
	writeJSON(summaryPath, s)
	if !s.OK {
		fmt.Printf("FAIL: %s\nsummary: %s\n", s.Error, summaryPath)
		os.Exit(1)
	}
	fmt.Printf("OK: image smoke on %s\nsummary: %s\n", s.Model, summaryPath)
}

func resolveProfileSettings(ctx context.Context, registrySources string, profileSlug string) (*gepsettings.InferenceSettings, string, func(), error) {
	entries, err := gepprofiles.ParseEngineProfileRegistrySourceEntries(registrySources)
	if err != nil {
		return nil, "", func() {}, err
	}
	if len(entries) == 0 {
		return nil, "", func() {}, errors.New("profile registries are required")
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
	parsed, err := gepprofiles.ParseEngineProfileSlug(profileSlug)
	if err != nil {
		closeFn()
		return nil, "", func() {}, err
	}
	resolved, err := chain.ResolveEngineProfile(ctx, gepprofiles.ResolveInput{EngineProfileSlug: parsed})
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
	return settings, resolved.EngineProfileSlug.String(), closeFn, nil
}

func defaultProfileRegistrySources() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []string{filepath.Join(home, ".config", "pinocchio", "profiles.yaml"), filepath.Join(home, ".pinocchio", "config", "profiles.yaml")}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return []string{candidate}
		}
	}
	return candidates[:1]
}
func safeName(s string) string {
	r := strings.NewReplacer("/", "-", " ", "-", ":", "-", ",", "-")
	return strings.Trim(r.Replace(s), "-")
}
func countEvents(events []map[string]any) map[string]int {
	out := map[string]int{}
	for _, ev := range events {
		if typ, _ := ev["type"].(string); typ != "" {
			out[typ]++
		}
	}
	return out
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
func writeJSON(path string, v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	_ = os.WriteFile(path, append(b, '\n'), 0o644)
}
func writeYAML(path string, v any) { b, _ := yaml.Marshal(v); _ = os.WriteFile(path, b, 0o644) }
func writeEvents(path string, rows []map[string]any) {
	f, _ := os.Create(path)
	if f == nil {
		return
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	for _, row := range rows {
		_ = enc.Encode(row)
	}
}
func finish(path string, s summary, err error, started time.Time) {
	s.Error = err.Error()
	s.DurationMS = time.Since(started).Milliseconds()
	writeJSON(path, s)
	fmt.Printf("FAIL: %s\nsummary: %s\n", err, path)
	os.Exit(1)
}
