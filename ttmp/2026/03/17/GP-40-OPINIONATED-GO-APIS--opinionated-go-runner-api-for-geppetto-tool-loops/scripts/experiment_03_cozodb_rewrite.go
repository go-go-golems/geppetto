//go:build ignore

// Experiment 03: CozoDB Editor Rewrite with Opinionated Runner
//
// Shows how the CozoDB hint engine (currently ~80 lines in engine.go)
// would look with the proposed runner API. This is a direct comparison.
//
// This file is a design sketch and is excluded from normal builds.

package main

import (
	"context"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/runner"
)

// --- Current implementation: ~80 lines ---
//
// func NewEngine() (*Engine, error) {
//     stepSettings := aisettings.NewStepSettings()
//     apiType := aitypes.ApiTypeClaude
//     stepSettings.Chat.ApiType = &apiType
//     streaming := true
//     stepSettings.Chat.Stream = &streaming
//     model := "claude-sonnet-4-20250514"
//     stepSettings.Chat.ModelName = &model
//     maxTokens := 8192
//     stepSettings.Chat.MaxResponseTokens = &maxTokens
//     stepSettings.Chat.APIKeys = map[string]string{
//         "claude": os.Getenv("ANTHROPIC_API_KEY"),
//     }
//     stepSettings.Chat.BaseUrls = map[string]string{}
//
//     eng, err := factory.NewEngineFromStepSettings(stepSettings)
//     if err != nil { return nil, err }
//
//     stepCtrl := toolloop.NewStepController()
//     return &Engine{eng: eng, stepCtrl: stepCtrl}, nil
// }
//
// func (e *Engine) runInference(ctx context.Context, systemPrompt, userMessage string,
//     onDelta func(string), extractors []structuredsink.Extractor,
//     externalSinks ...gepevents.EventSink) (string, *structuredsink.FilteringSinkWithContext, error) {
//
//     turn, _ := turns.NewTurnBuilder().
//         AddSystemBlock(systemPrompt).
//         AddUserBlock(userMessage).Build()
//
//     streamingSink := newStreamingTextSink(onDelta)
//     filterSink := structuredsink.NewFilteringSinkWithContext(ctx, extractors...)
//
//     sess := session.NewSession()
//     sess.Builder = enginebuilder.New(
//         enginebuilder.WithBase(e.eng),
//         enginebuilder.WithEventSinks(filterSink),
//         enginebuilder.WithStepController(e.stepCtrl),
//     )
//     sess.Append(turn)
//     handle, _ := sess.StartInference(ctx)
//     resultTurn, _ := handle.Wait()
//
//     text := extractAssistantText(resultTurn)
//     return text, filterSink, nil
// }

// --- Proposed implementation: ~10 lines ---

func generateHint(ctx context.Context, schema, question string, onDelta func(string)) (string, error) {
	result, err := runner.Run(ctx, question,
		runner.System(buildCozoSystemPrompt(schema)),
		runner.Model("claude-sonnet-4-20250514"),
		runner.MaxTokens(8192),
		runner.Stream(onDelta),
	)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

func diagnoseError(ctx context.Context, schema, errorMsg, failedScript string, onDelta func(string)) (string, error) {
	result, err := runner.Run(ctx,
		fmt.Sprintf("Error: %s\n\nFailed script:\n%s", errorMsg, failedScript),
		runner.System(buildDiagnosisSystemPrompt(schema)),
		runner.Model("claude-sonnet-4-20250514"),
		runner.MaxTokens(8192),
		runner.Stream(onDelta),
	)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// Stubs for compilation
func buildCozoSystemPrompt(schema string) string { return "" }
func buildDiagnosisSystemPrompt(schema string) string { return "" }

func main() {
	ctx := context.Background()
	text, err := generateHint(ctx, "CREATE TABLE users (id INT, name TEXT)", "How do I list all users?", func(d string) { fmt.Print(d) })
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("\n---")
	fmt.Println(text)
}
