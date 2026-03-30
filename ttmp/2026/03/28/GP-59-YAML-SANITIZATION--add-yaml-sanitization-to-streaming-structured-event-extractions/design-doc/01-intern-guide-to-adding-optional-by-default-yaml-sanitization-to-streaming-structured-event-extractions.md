---
Title: Intern guide to adding optional-by-default YAML sanitization to streaming structured event extractions
Ticket: GP-59-YAML-SANITIZATION
Status: active
Topics:
    - geppetto
    - events
    - streaming
    - yaml
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/context.go
      Note: event sink attachment and publish path
    - Path: geppetto/pkg/events/structuredsink/filtering_sink.go
      Note: structured tag scanning and extractor ownership boundary
    - Path: geppetto/pkg/events/structuredsink/parsehelpers/helpers.go
      Note: current YAML parsing helper and proposed sanitization insertion point
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: provider streaming partial/final event emission
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: responses streaming partial/final event emission
    - Path: sanitize/pkg/yaml/sanitize.go
      Note: sanitize-backed YAML cleanup used by the shipped parsehelpers implementation
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: downstream-only SEM translation boundary
ExternalSources: []
Summary: The correct implementation point for YAML sanitization is Geppetto's structured-sink parsehelpers layer. FilteringSink should remain tag-oriented, and Pinocchio should remain downstream-only. The shipped implementation adds a default-on sanitization path to parsehelpers using `github.com/go-go-golems/sanitize/pkg/yaml`, adds helper-focused tests, and updates docs that previously showed direct `yaml.Unmarshal` usage.
LastUpdated: 2026-03-30T18:30:00-04:00
WhatFor: Onboard an unfamiliar engineer and give them a safe, evidence-backed implementation plan for default-on YAML sanitization in streaming structured event extraction.
WhenToUse: Use when implementing or reviewing structured-streaming YAML parsing in Geppetto, especially if deciding between provider, sink, extractor, helper, or Pinocchio layers.
---


# Intern guide to adding optional-by-default YAML sanitization to streaming structured event extractions

## Executive Summary

The change should land in `geppetto/pkg/events/structuredsink/parsehelpers`, not in `pinocchio` and not in `FilteringSink` itself. The evidence is straightforward:

- provider engines emit plain `EventPartialCompletion` and `EventFinal` events into the event-sink pipeline;
- `FilteringSink` only strips `<package:type:version>` blocks and forwards raw inner bytes to extractor sessions;
- YAML parsing currently happens in helper code and in example extractor code paths via direct `yaml.Unmarshal`;
- Pinocchio only consumes already-emitted events and translates them to SEM `llm.delta` and `llm.final` frames.

The implemented approach adds a default-on sanitization step to the Geppetto YAML parsing helpers, backed by `github.com/go-go-golems/sanitize/pkg/yaml`. The option remains disableable via `DebounceConfig.SanitizeYAML`, but the zero-config default sanitizes. The sink API stays generic and tag-oriented.

This guide is written for a new intern. It explains the runtime path end-to-end, names the files to read in order, calls out current doc/code mismatches that can cause confusion, proposes the minimal API change, and outlines tests and rollout steps.

## Problem Statement And Scope

The user-facing problem is that structured event extraction often receives "almost YAML" from the LLM provider. Typical failures include unquoted values containing `:`, markdown fences, and other minor formatting issues. Today the structured-streaming flow has no built-in YAML sanitization in Geppetto's extraction helpers, so `yaml.Unmarshal` fails earlier and more often than necessary.

This is a design problem with a placement question, not just a parser tweak. If you put the fix in the wrong layer, you will either make the system more coupled than it needs to be or fail to cover the real call path.

The scope of this ticket is:

1. Add YAML sanitization support for streaming structured event extraction helpers in Geppetto.
2. Make that sanitization optional, but enabled by default.
3. Keep the sink generic and tag-oriented rather than YAML-specific.
4. Update docs and examples so new extractor implementations naturally take the sanitized path.

The scope explicitly does not include:

1. Rewriting `FilteringSink` to parse YAML itself.
2. Putting YAML cleanup in Pinocchio UI code or SEM translation.
3. Building a general-purpose "fix any broken LLM output" subsystem.
4. Solving arbitrary prose-inside-tags failures beyond the existing sanitizer's reach.

## Definitions And Mental Model

Before touching code, keep these terms straight:

- **Provider engine**: OpenAI, Responses API, Gemini, Claude, and similar code that converts provider streaming into Geppetto events.
- **Event sink**: A consumer of Geppetto events. Sinks are attached to `context.Context` and receive events during inference.
- **FilteringSink**: A specific event sink that scans text for tagged structured blocks and splits the stream into clean prose plus typed structured events.
- **Extractor**: Application-specific code registered with `FilteringSink` for one tag triple such as `<geppetto:citations:v1>`.
- **Extractor session**: Per-block runtime object that receives `OnStart`, `OnRaw`, and `OnCompleted`.
- **parsehelpers**: Geppetto helper package for code-fence stripping and debounced YAML parsing used by extractor implementations.
- **SEM translation**: Pinocchio's downstream conversion of Geppetto events into webchat-friendly frames.

If you remember only one sentence, remember this one: Geppetto owns extraction and parsing; Pinocchio owns rendering and transport.

## Current-State Architecture

### Step 1: Provider engines emit streaming text events

Geppetto provider engines publish textual streaming through `EventPartialCompletion` and `EventFinal`.

Evidence:

- `geppetto/pkg/steps/ai/openai/engine_openai.go:324-333` publishes partial completion events for non-empty deltas.
- `geppetto/pkg/steps/ai/openai/engine_openai.go:417-433` publishes the final event at the end of streaming.
- `geppetto/pkg/steps/ai/openai_responses/engine.go:292-299` does the same in the Responses engine.
- `geppetto/pkg/steps/ai/openai_responses/engine.go:875-875` and `geppetto/pkg/steps/ai/openai_responses/engine.go:984-984` publish final events in streaming and non-streaming paths.

Relevant event types:

- `geppetto/pkg/events/chat-events.go:178-195` defines `EventFinal`.
- `geppetto/pkg/events/chat-events.go:324-347` defines `EventPartialCompletion`.

### Step 2: Event sinks are attached to the run context

Sinks are context-carried, not hard-coded into provider engines.

Evidence:

- `geppetto/pkg/events/context.go:16-26` attaches sinks with `events.WithEventSinks`.
- `geppetto/pkg/events/context.go:39-51` fan-outs published events to all sinks in context.
- `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:158-166` shows the runner attaching configured sinks to the run context before inference.

This means any structured extraction behavior is downstream of the provider and upstream of the UI.

### Step 3: FilteringSink scans tags and delegates payload bytes to extractors

`FilteringSink` does not parse YAML. It:

1. intercepts `EventPartialCompletion` and `EventFinal`;
2. tracks per-stream parser state;
3. detects open tags and close tags;
4. forwards text outside tags;
5. sends bytes inside tags to the registered extractor session.

Evidence:

- `geppetto/pkg/events/structuredsink/filtering_sink.go:48-61` defines `Extractor` and `ExtractorSession`.
- `geppetto/pkg/events/structuredsink/filtering_sink.go:63-77` states the sink boundary explicitly: filter text, emit typed events, but avoid durable domain-state commitment at streaming time.
- `geppetto/pkg/events/structuredsink/filtering_sink.go:303-330` handles `EventFinal`.
- `geppetto/pkg/events/structuredsink/filtering_sink.go:340-465` contains the core scanner and routing logic.
- `geppetto/pkg/events/structuredsink/filtering_sink.go:468-496` handles malformed captured blocks.

The critical code path for extractor ownership is:

```text
open tag detected
  -> create extractor session
  -> call OnStart()
payload bytes accumulate
  -> call OnRaw(chunk)
close tag detected
  -> call OnCompleted(raw, success=true)
or stream ends malformed
  -> call OnCompleted(raw, success=false, err)
```

That means the extractor, not the sink, owns YAML parsing semantics.

### Step 4: YAML parsing helpers exist, but sanitization does not

Geppetto already has helper code for fence stripping and debounced YAML parsing:

- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:12-31` strips code fences.
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:34-39` defines `DebounceConfig`.
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:41-50` defines `YAMLController`.
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:55-89` implements `FeedBytes` and `FinalBytes`.
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:91-127` performs parsing via direct `yaml.Unmarshal`.

There is no sanitization step between `StripCodeFenceBytes(...)` and `yaml.Unmarshal(...)`.

### Step 5: A reusable YAML sanitizer exists in the sibling sanitize module

This workspace now uses the dedicated `sanitize` module for LLM-oriented YAML cleanup:

- `sanitize/pkg/yaml/sanitize.go:3-17` exposes `Sanitize(...)` and `SanitizeWithOptions(...)`.
- `sanitize/pkg/yaml/sanitize.go:19-80` shows the iterative cleanup pipeline and `Result` shape.
- `pinocchio/pkg/middlewares/agentmode/parser.go:28-38` demonstrates the same sanitize-backed parser pattern already working in application code.

This matters because Geppetto can now depend on a small focused module (`github.com/go-go-golems/sanitize`) instead of reaching through Glazed helpers, and the same sanitize API is already in use elsewhere in the workspace.

### Step 6: Pinocchio is downstream-only in this flow

Pinocchio translates Geppetto events into SEM frames after Geppetto has already emitted them.

Evidence:

- `pinocchio/pkg/webchat/sem_translator.go:284-301` maps `EventPartialCompletion` to `llm.delta`.
- `pinocchio/pkg/webchat/sem_translator.go:304-318` maps `EventFinal` to `llm.final`.

There is no YAML parsing in this layer. By the time Pinocchio sees the event, the question of "did the extractor successfully parse the YAML" should already be settled.

### Diagram: End-to-end runtime

```text
LLM Provider Stream
    |
    v
OpenAI / Responses engine
    |
    |  emits EventPartialCompletion / EventFinal
    v
events.PublishEventToContext(ctx, ev)
    |
    v
FilteringSink
    |
    |  strips tags, forwards prose, routes inner bytes
    v
ExtractorSession
    |
    |  StripCodeFenceBytes -> sanitize? -> yaml.Unmarshal
    v
Typed Geppetto events
    |
    v
Downstream sink / Watermill / logs
    |
    v
Pinocchio SEM translator
    |
    v
llm.delta / llm.final / custom UI events
```

## Gap Analysis

### Gap 1: Helper parsing is strict, while provider output is often not

The parsing helper directly calls `yaml.Unmarshal` on extracted bytes. That is stricter than real LLM output quality. Small formatting imperfections become hard failures even though a safe cleanup step could recover them.

### Gap 2: There is no single default path for final YAML parsing

The helper has `FinalBytes(...)`, but docs and examples still show direct `yaml.Unmarshal(raw, &payload)` in extractor `OnCompleted`.

Evidence:

- `geppetto/pkg/doc/topics/11-structured-sinks.md:240-247` uses direct `yaml.Unmarshal`.
- `geppetto/pkg/doc/playbooks/03-progressive-structured-data.md:207-214` uses direct `yaml.Unmarshal`.
- `geppetto/pkg/doc/tutorials/04-structured-data-extraction.md:239-246` uses direct `yaml.Unmarshal`.

If you only add sanitization to `YAMLController`, extractors that follow the docs but skip the controller still miss the feature.

### Gap 3: The docs contain stale helper API examples

The docs show helper APIs that do not match current code:

- `geppetto/pkg/doc/playbooks/03-progressive-structured-data.md:160-178` refers to `*parsehelpers.DebouncedYAML[...]` and `s.parser.Feed(chunk)`.
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:41-50` defines `YAMLController`, not `DebouncedYAML`.
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:55-70` exposes `FeedBytes`, not `Feed`.
- `geppetto/pkg/doc/tutorials/04-structured-data-extraction.md:462-479` repeats the stale `Feed(...)` usage.

This is not the main feature request, but it is directly relevant because a new intern can easily implement against the wrong API.

### Gap 4: Sink-level adjacent option drift exists

`FilteringSink.Options` exposes `MaxCaptureBytes`, but the current tests acknowledge that it is not enforced yet:

- `geppetto/pkg/events/structuredsink/filtering_sink.go:15-20` defines the option.
- `geppetto/pkg/events/structuredsink/filtering_sink_test.go:849-862` explicitly says the limit is not implemented.

This is not the YAML sanitization change, but it is a useful reminder that adjacent options may look finished while still being incomplete.

## Design Goals

The design should satisfy all of these goals:

1. **Correct placement**: put the change where YAML is parsed today.
2. **Default-on behavior**: existing callers using zero-value config should get sanitization automatically.
3. **Opt-out support**: callers must be able to disable cleanup when they want strict parsing.
4. **Minimal coupling**: do not make `FilteringSink` understand YAML-specific semantics.
5. **Reusable final parse API**: support both debounced and final-only extractors.
6. **Doc alignment**: update examples so new extractors take the right path by default.

## Proposed Solution

### Core Decision

Add YAML sanitization to `geppetto/pkg/events/structuredsink/parsehelpers` and make it enabled by default through configuration semantics that preserve the zero value.

Do not add YAML-specific behavior to `FilteringSink`.

Do not add the behavior in Pinocchio.

### API Shape

The smallest practical API is:

```go
type DebounceConfig struct {
    SnapshotEveryBytes int
    SnapshotOnNewline  bool
    ParseTimeout       time.Duration
    MaxBytes           int
    SanitizeYAML       *bool `json:"sanitize_yaml,omitempty" yaml:"sanitize_yaml,omitempty"`
}
```

Why this shape:

- `SanitizeYAML *bool` preserves zero-config default-on behavior while still allowing explicit opt-out.
- The config shape matches the pattern already used in `pinocchio/pkg/middlewares/agentmode`.
- `YAMLController.FinalBytes` and `YAMLController.tryParse` delegate to the same internal normalization/parsing function, so partial and final parsing cannot drift.

### Recommended Internal Structure

Inside `parsehelpers/helpers.go`, the normalization pipeline should look like this:

```go
func (c *YAMLController[T]) normalizedYAML(raw []byte) ([]byte, error) {
    _, body := StripCodeFenceBytes(raw)

    if c.cfg.MaxBytes > 0 && len(body) > c.cfg.MaxBytes {
        return nil, errors.New("payload too large")
    }

    src := strings.TrimSpace(string(body))
    if src == "" {
        return nil, errors.New("empty")
    }

    if c.cfg.SanitizeEnabled() {
        result := yamlsanitize.Sanitize(strings.TrimSpace(string(body)))
        if trimmed := strings.TrimSpace(result.Sanitized); trimmed != "" {
            src = trimmed
        }
    }

    return []byte(src), nil
}

func (c *YAMLController[T]) FinalBytes(raw []byte) (*T, error) {
    if len(raw) > 0 {
        return c.parseYAML(raw)
    }
    return c.tryParse()
}
```

For `tryParse()`, keep the timeout logic, but move the pre-unmarshal normalization outside the goroutine so both timeout and non-timeout modes see the same sanitized body. The implementation should also centralize the actual unmarshal path in one helper (`parseYAML(...)`) so `FinalBytes(...)` and `FeedBytes(...)` share behavior.

### Why Not Put Sanitization In FilteringSink

This is the most important design rejection.

`FilteringSink` currently knows:

- tag syntax;
- stream boundaries;
- malformed block policy;
- extractor registration.

It does not know:

- whether a payload is YAML, JSON, plaintext, or something custom;
- whether a specific extractor wants strict or permissive parsing;
- whether the extractor wants raw bytes preserved exactly.

If you sanitize in the sink, you silently change the byte contract of `OnRaw` and `OnCompleted` for every extractor. That is too invasive for a generic sink.

### Why Not Put Sanitization In Pinocchio

Pinocchio only sees Geppetto events after extraction logic has already run. It is too late in the pipeline. Also, Pinocchio's `sem_translator.go` only maps event types to SEM frames; it does not interpret structured extraction payloads.

### Why Reuse The Glazed Helper

The Glazed helper is already:

- written for "LLM cleanup" YAML input;
- used in a CLI flow in the same ecosystem;
- available as a dependency already pulled into Geppetto.

Reusing it avoids creating two divergent YAML cleanup heuristics inside sibling repos.

## Detailed Call Flow After The Change

### Progressive path

```text
OnRaw(chunk)
  -> YAMLController.FeedBytes(chunk)
      -> append to internal buffer
      -> if cadence says "attempt parse"
          -> normalizedYAMLBody(buffer, cfg)
          -> sanitize unless disabled
          -> yaml.Unmarshal(normalizedBody)
          -> return partial typed value or error
  -> extractor emits partial event if wanted
```

### Final path

```text
OnCompleted(raw, success=true)
  -> parser.FinalBytes(raw)
      -> StripCodeFenceBytes(raw)
      -> sanitize unless disabled
      -> yaml.Unmarshal
  -> extractor emits final typed event
```

### Malformed block path

```text
OnCompleted(raw, success=false, err=malformed)
  -> extractor should not attempt YAML parse unless it intentionally wants best-effort recovery
  -> surface failure event with err string
```

The malformed-block behavior remains owned by `FilteringSink` and extractor logic, not by the sanitization helper.

## File-Level Implementation Plan

### Phase 1: Implement helper-layer sanitization

Modify:

- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`

Tasks:

1. Add `SanitizeYAML *bool` plus helper methods (`withDefaults`, `SanitizeEnabled`, `WithSanitizeYAML`) to `DebounceConfig`.
2. Add a shared `normalizedYAML(...)` helper that strips fences, enforces size limits, trims empty payloads, and runs `yamlsanitize.Sanitize(...)` when enabled.
3. Refactor `FinalBytes(...)` and `tryParse()` to reuse the same `parseYAML(...)` path.
4. Preserve existing timeout behavior while moving normalization outside the goroutine.
5. Pin the new module dependency in `geppetto/go.mod` and `go.sum`.

### Phase 2: Add tests around the helper, not the sink

Create:

- `geppetto/pkg/events/structuredsink/parsehelpers/helpers_test.go`

Recommended test cases:

1. `FinalBytes(...)` succeeds on already-valid YAML.
2. `FinalBytes(...)` succeeds on YAML that needs cleanup when sanitization is enabled.
3. `FinalBytes(...)` fails on the same input when `SanitizeYAML` is explicitly disabled.
4. Fence stripping plus sanitization works together.
5. Empty payload still returns the existing "empty" style error.
6. `MaxBytes` is enforced before and after cleanup.
7. `FeedBytes` progressive parse uses the same sanitization path as final parse.
8. Timeout mode still sanitizes before the goroutine unmarshal.

Representative fixture idea:

```yaml
citations:
  - title: GPT-4 Technical Report
    note: OpenAI: technical report
```

That `note` field is a good simple case because the colon in the scalar is a classic failure mode.

### Phase 3: Update docs so new extractors follow the correct path

Modify:

- `geppetto/pkg/doc/topics/11-structured-sinks.md`
- `geppetto/pkg/doc/playbooks/03-progressive-structured-data.md`
- `geppetto/pkg/doc/tutorials/04-structured-data-extraction.md`

Doc changes:

1. Replace direct `yaml.Unmarshal(raw, &payload)` final parse examples with `parsehelpers.YAMLController.FinalBytes(...)`.
2. Replace stale `DebouncedYAML` concrete type references with `YAMLController`.
3. Replace `Feed(...)` with `FeedBytes(...)`.
4. State explicitly that sanitization is enabled by default and can be disabled with `WithSanitizeYAML(false)` or `sanitize_yaml: false`.

### Phase 4: Optional validation pass with a small example extractor

This repo does not appear to contain a first-party production YAML extractor implementation in active runtime code. Because of that, a small helper-focused test suite is the minimum requirement, but an example extractor smoke test would still be useful for confidence.

If you add one, keep it narrowly scoped:

- fake extractor;
- fake payload struct;
- feed a tagged block with slightly broken YAML through `FilteringSink`;
- verify final typed event payload matches the sanitized parse.

## Pseudocode For The Recommended Extractor Style

```go
type citationsSession struct {
    meta   events.EventMetadata
    itemID string
    parser *parsehelpers.YAMLController[CitationsPayload]
}

func (s *citationsSession) OnStart(ctx context.Context) []events.Event {
    s.parser = parsehelpers.NewDebouncedYAML[CitationsPayload](parsehelpers.DebounceConfig{
        SnapshotEveryBytes: 256,
        SnapshotOnNewline:  true,
        MaxBytes:           64 << 10,
        // SanitizeYAML omitted -> sanitize by default
    })
    return nil
}

func (s *citationsSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    payload, err := s.parser.FeedBytes(chunk)
    if err != nil || payload == nil {
        return nil
    }
    return []events.Event{
        NewCitationPartialEvent(s.meta, s.itemID, *payload),
    }
}

func (s *citationsSession) OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event {
    if !success {
        return []events.Event{
            NewCitationCompleteErrorEvent(s.meta, s.itemID, err),
        }
    }

    payload, parseErr := s.parser.FinalBytes(raw)
    if parseErr != nil || payload == nil {
        return []events.Event{
            NewCitationCompleteErrorEvent(s.meta, s.itemID, parseErr),
        }
    }

    return []events.Event{
        NewCitationCompleteEvent(s.meta, s.itemID, *payload),
    }
}
```

## Risks And Tradeoffs

### Risk 1: Sanitization can change semantics

Any cleanup helper is heuristic. The `sanitize` module may quote values that a strict caller would prefer to reject. That is why opt-out support matters.

Mitigation:

- keep sanitization disableable;
- expose the opt-out explicitly as `SanitizeYAML: false` or `WithSanitizeYAML(false)`;
- keep the default helper narrow and documented;
- add tests for representative repaired inputs.

### Risk 2: Some callers may rely on raw strict failure

If an external extractor uses `YAMLController` with zero-value config today and assumes strict parsing, the behavior will change after this feature.

Mitigation:

- make the change explicit in release notes and docs;
- support `SanitizeYAML: false` and `WithSanitizeYAML(false)`;
- prefer a ticketed implementation note in changelog/docs.

### Risk 3: Existing docs are already slightly stale

If you only add code and skip docs, new extractor implementations may continue to bypass the helper and therefore bypass the new sanitization behavior.

Mitigation:

- treat doc updates as part of the feature, not optional polish.

### Risk 4: The adjacent `MaxCaptureBytes` option may distract review

Reviewers may notice that `FilteringSink.Options.MaxCaptureBytes` is not enforced yet and ask for both fixes at once.

Recommendation:

- keep this ticket scoped to parsehelpers sanitization;
- mention `MaxCaptureBytes` as adjacent but out of scope.

## Alternatives Considered

### Alternative A: Put sanitization directly in `FilteringSink`

Rejected because it changes the sink contract from "raw payload router" to "YAML-aware mutating router." That is too high in the stack and too broad in blast radius.

### Alternative B: Put sanitization in provider engines

Rejected because provider engines emit generic text deltas. They do not know where structured blocks begin or which tags imply YAML.

### Alternative C: Put sanitization in Pinocchio

Rejected because Pinocchio is downstream of extraction and only translates emitted events.

### Alternative D: Add docs only and let extractors sanitize themselves

Rejected because it does not produce a default-on library behavior. It would keep the footgun in place.

### Alternative E: Add a positive `Sanitize bool` field

Rejected because a plain `bool` would make the zero value ambiguous. The shipped `*bool` plus helper methods preserves zero-config default-on behavior while still serializing cleanly.

## Open Questions

1. Do we want a future exported final-only helper for extractors that do not keep a session-local parser, or is `FinalBytes(...)` enough? The shipped implementation keeps the smaller API surface and relies on `FinalBytes(...)`.
2. Do we want a future `parsehelpers.ParseJSONBytes(...)` equivalent for parity? Out of scope here, but the pattern would be similar.
3. Should Geppetto eventually surface sanitize metadata (for example, whether the payload changed) in the helper API? Out of scope here, but possible if downstream consumers need repair observability.

## Implementation Checklist

1. Read the files in the reference order below.
2. Implement helper-layer sanitization and the shared `FinalBytes(...)` / `FeedBytes(...)` parse path.
3. Add helper tests first; do not start with sink tests.
4. Update the three structured-sink docs so examples stop bypassing the helper.
5. Run focused tests for `parsehelpers` and `structuredsink`.
6. Run `docmgr doctor` on the ticket after updating docs.

## Reference Reading Order

Read these files in this order if you are new to the system:

1. `geppetto/pkg/doc/topics/11-structured-sinks.md`
2. `geppetto/pkg/events/structuredsink/filtering_sink.go`
3. `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`
4. `geppetto/pkg/events/context.go`
5. `geppetto/pkg/steps/ai/openai/engine_openai.go`
6. `geppetto/pkg/steps/ai/openai_responses/engine.go`
7. `pinocchio/pkg/webchat/sem_translator.go`
8. `sanitize/pkg/yaml/sanitize.go`

## References

Primary code references:

- `geppetto/pkg/events/context.go:16-26`
- `geppetto/pkg/events/context.go:39-51`
- `geppetto/pkg/events/chat-events.go:178-195`
- `geppetto/pkg/events/chat-events.go:324-347`
- `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:158-166`
- `geppetto/pkg/steps/ai/openai/engine_openai.go:324-333`
- `geppetto/pkg/steps/ai/openai/engine_openai.go:417-433`
- `geppetto/pkg/steps/ai/openai_responses/engine.go:292-299`
- `geppetto/pkg/steps/ai/openai_responses/engine.go:875-875`
- `geppetto/pkg/steps/ai/openai_responses/engine.go:984-984`
- `geppetto/pkg/events/structuredsink/filtering_sink.go:48-61`
- `geppetto/pkg/events/structuredsink/filtering_sink.go:63-77`
- `geppetto/pkg/events/structuredsink/filtering_sink.go:303-330`
- `geppetto/pkg/events/structuredsink/filtering_sink.go:340-465`
- `geppetto/pkg/events/structuredsink/filtering_sink.go:468-496`
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:12-31`
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:34-39`
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:41-50`
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:55-127`
- `geppetto/pkg/events/structuredsink/filtering_sink_test.go:849-862`
- `geppetto/pkg/doc/topics/11-structured-sinks.md:156-190`
- `geppetto/pkg/doc/topics/11-structured-sinks.md:234-247`
- `geppetto/pkg/doc/playbooks/03-progressive-structured-data.md:157-178`
- `geppetto/pkg/doc/playbooks/03-progressive-structured-data.md:197-214`
- `geppetto/pkg/doc/tutorials/04-structured-data-extraction.md:190-246`
- `geppetto/pkg/doc/tutorials/04-structured-data-extraction.md:462-479`
- `pinocchio/pkg/webchat/sem_translator.go:284-318`
- `sanitize/pkg/yaml/sanitize.go:3-80`
- `glazed/cmd/glaze/cmds/yaml.go:94-100`
