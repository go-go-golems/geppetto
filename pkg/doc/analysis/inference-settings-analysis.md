# Inference Settings Analysis (SQLite-assisted)

## Methodology
- Extracted git history into SQLite (`commits` table) and filtered for inference-related changes to understand evolution of settings handling (e.g., commits introducing `RunInference` API and factories).【933eb0†L1-L14】
- Indexed inference-related files into SQLite (`files` table) to surface engine and settings modules, ensuring broad coverage of implementation and docs (e.g., `pkg/inference/engine`, provider engines, tutorials).【655ac6†L1-L10】
- Reviewed code and documentation to map symbols, configuration flows, and architectural principles.

## Core Architecture and Settings Flow
- **Engine interface**: `engine.Engine` exposes a single `RunInference(ctx, *turns.Turn)` method; engines focus on provider API calls and keep tool orchestration external.【555408†L9-L15】
- **Engine configuration options**: `engine.Option` functions mutate `engine.Config` (notably `EventSinks`) via helpers like `engine.WithSink`, applied through `engine.ApplyOptions` to stack streaming/event behavior per engine instance.【be21a9†L5-L38】
- **Provider-agnostic creation**: `factory.StandardEngineFactory.CreateEngine(settings, options...)` selects providers (OpenAI, Claude/Anthropic, Gemini, Anyscale, Fireworks) from `settings.Chat.ApiType`, validates provider-specific requirements, and passes engine options to the concrete provider engine.【f4c2cb†L19-L189】
- **Settings surface**: `settings.StepSettings` aggregates `API`, `Chat`, provider-specific (OpenAI/Claude/Gemini/Ollama), client, and embeddings settings; constructed via `NewStepSettings` and serializable from YAML or parsed layers for CLI/docs integration.【7403f8†L38-L119】
- **Metadata/telemetry**: `StepSettings.GetMetadata` compiles a provider-agnostic map (engine, API type, base URLs, sampling params, stop sequences, embeddings info) for observability or downstream routing; it captures provider-specific tuning knobs such as OpenAI penalties and Ollama temperature/top-k/top-p.【7403f8†L121-L217】
- **Summaries for UX**: `StepSettings.GetSummary(verbose)` formats human-readable digests of API keys (masked), base URLs, chat parameters, and provider/tooling extras, optionally including verbose knobs and embeddings cache settings for debugging exports.【fd014f†L282-L430】

## Architectural Principles (from docs/examples)
- Docs emphasize separation of concerns: engines handle API I/O/streaming, helpers handle tool orchestration, factories assemble engines from configuration layers, and middleware adds cross-cutting concerns like logging/events.【33a52f†L39-L146】
- Benefits include simplicity (single `RunInference`), provider agnosticism, testability through engine mocking, and composability of helpers and middleware, reinforcing settings design around a minimal core plus optional options/sinks.【33a52f†L52-L99】
- Tutorials demonstrate creating engines from parsed layers, applying options such as event sinks for streaming, and running basic inference, illustrating how settings feed engine creation and runtime behavior.【33a52f†L100-L183】

## Git History Highlights (via SQLite query)
- Recent commits show the introduction and refinement of inference APIs and factories: `20c0d3a` (“Add new RunInference API”), `0f52c11` (“Add inference factory”), and subsequent refactors for conversation/streaming alignment and documentation updates (e.g., `cc86d07`).【933eb0†L6-L14】
- The sequence indicates a migration from step-based inference toward the engine-first, option-driven architecture, validating the current settings layout.

## Key Filenames and Symbols to Track
- **Engine core**: `pkg/inference/engine/engine.go` (`Engine`, `RunInference`); `pkg/inference/engine/options.go` (`Option`, `WithSink`, `ApplyOptions`).
- **Factory layer**: `pkg/inference/engine/factory/factory.go` (`StandardEngineFactory`, `CreateEngine`, `validate*Settings`).
- **Settings aggregation**: `pkg/steps/ai/settings/settings-step.go` (`StepSettings`, `NewStepSettings*`, `GetMetadata`, `GetSummary`).
- **Docs/examples**: `pkg/doc/topics/06-inference-engines.md` (architecture, examples), `pkg/doc/tutorials/01-streaming-inference-with-tools.md` (CLI usage and streaming with tools), provider engines in `pkg/steps/ai/{openai,claude,gemini}` for concrete settings consumption.

## Architectural Takeaways
- **Option-driven extensibility**: Functional options (especially `WithSink`) let callers attach event sinks and middleware without embedding stream logic in engines, aligning with the separation-of-concerns principle.
- **Settings as a unified contract**: `StepSettings` merges provider-agnostic chat/client knobs with provider-specific tuning and embeddings, enabling factories to validate and wire engines consistently.
- **Provider validation upfront**: Factory-level validation prevents misconfigured inference runs by checking API keys/base URLs and provider-required sub-settings before engine construction.
- **Observability hooks**: Metadata and summaries derived from settings supply runtime tracing, UI display, and analytics with minimal coupling to providers.
