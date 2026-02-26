# Inference Settings Learning Lab

This toolkit derives from the accompanying analysis document to help you internalize how inference settings and engines work.

## Multiple-Choice Quizzes
1. **Engine Responsibilities**: What does an `engine.Engine` intentionally *not* handle?
   - A) Provider HTTP requests and streaming events
   - B) Tool orchestration loops and tool execution
   - C) Parsing responses into `turns.Block`s
   - D) Returning an updated `turns.Turn`

2. **Provider Selection**: Which setting drives `factory.StandardEngineFactory` to pick a provider implementation by default?
   - A) `settings.OpenAI.N`
   - B) `settings.Chat.ApiType`
   - C) `settings.Client.Timeout`
   - D) `settings.Gemini.Type`

3. **Event Streaming**: Which option stacks event sinks onto an engine during creation?
   - A) `engine.WithMiddleware`
   - B) `engine.WithSink`
   - C) `engine.WithToolConfig`
   - D) `engine.WithTurn`

4. **Validation Gate**: Why does the factory check `APIKeys` and `BaseUrls` before creating a provider engine?
   - A) To lazily fetch missing credentials at runtime
   - B) To fail fast on misconfiguration and avoid provider-specific runtime errors
   - C) To auto-generate sample requests for docs
   - D) To bypass embedding settings

5. **Metadata Utility**: What does `StepSettings.GetMetadata` supply to downstream systems?
   - A) Compiled binaries
   - B) Provider-neutral telemetry like engine, API type, sampling knobs, and embeddings info
   - C) Only OpenAI-specific penalties
   - D) Tool execution results

## Essay Prompts
- Explain how the separation of concerns between engines, factories, and helpers influences the design of inference settings. Reference specific structs and methods from the analysis to illustrate your reasoning.
- Compare the validation logic across providers (OpenAI vs. Claude vs. Gemini) and argue how it shapes runtime reliability and developer ergonomics.
- Discuss how metadata and summary generation from `StepSettings` can be used to instrument observability dashboards for inference workloads.

## Mini-Projects
1. **Streaming Trace Experiment**: Write a small Go snippet that builds `StepSettings` from YAML, creates an engine via `StandardEngineFactory`, attaches two custom sinks using `engine.WithSink`, and logs the resulting `GetMetadata()` map before running `RunInference` on a seed `turns.Turn`.
2. **Provider Validation Harness**: Build a CLI that accepts provider flags and intentionally omits required API keys/base URLs. Use the factory to demonstrate validation errors for each provider and print friendly guidance derived from `GetSummary(true)`.
3. **Settings Diff Viewer**: Implement a tool that loads two YAML configurations into `StepSettings`, computes field-by-field differences (e.g., temperature, max tokens, embeddings cache settings), and outputs both the raw diff and the masked summaries to illustrate safe sharing of configuration snapshots.
4. **Docs-to-Code Crosswalk**: Create a test that walks `pkg/doc/topics/06-inference-engines.md` examples and asserts that referenced symbols (`engine.WithSink`, `RunInference`, `StandardEngineFactory`) exist in the codebase, reinforcing alignment between documentation and implementation.
5. **Contextual Tooling Lab**: Extend the streaming inference tutorial to register a tool registry on `context.Context`, run a simple tool-calling loop, and record how tool-related settings would flow through metadata and summaries for auditability.

Use these exercises to connect configuration structures, factory behaviors, and runtime observability into a cohesive mental model of Geppetto's inference settings.
