---
Title: Provider Gap Evidence Line Anchors
Ticket: 2026-06-05-geppetto-provider-gap-audit
Status: active
Topics:
  - geppetto
  - providers
  - reasoning
  - streaming
  - tools
DocType: reference
Intent: short-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Line-anchored source excerpts captured during the static provider gap audit.
LastUpdated: 2026-06-05T09:20:00-04:00
WhatFor: Use as raw evidence for analysis/01-provider-gap-audit-findings.md.
WhenToUse: Read when verifying exact code snippets cited by the provider gap audit findings.
---

## pkg/events/canonical_events.go:73-210
```go
    73	
    74	func (e *EventProviderCallStarted) Correlation() Correlation { return e.Correlation_ }
    75	
    76	var _ CorrelatedEvent = &EventProviderCallStarted{}
    77	
    78	type EventProviderCallMetadataUpdated struct {
    79		EventImpl
    80		Correlation_ Correlation `json:"correlation"`
    81		StopReason   string      `json:"stop_reason,omitempty"`
    82		StopSequence string      `json:"stop_sequence,omitempty"`
    83		Usage        *Usage      `json:"usage,omitempty"`
    84	}
    85	
    86	func NewProviderCallMetadataUpdatedEvent(metadata EventMetadata, corr Correlation, stopReason, stopSequence string, usage *Usage) *EventProviderCallMetadataUpdated {
    87		return &EventProviderCallMetadataUpdated{EventImpl: EventImpl{Type_: EventTypeProviderCallMetadataUpdated, Metadata_: metadata}, Correlation_: corr, StopReason: stopReason, StopSequence: stopSequence, Usage: usage}
    88	}
    89	
    90	func (e *EventProviderCallMetadataUpdated) Correlation() Correlation { return e.Correlation_ }
    91	
    92	var _ CorrelatedEvent = &EventProviderCallMetadataUpdated{}
    93	
    94	type EventProviderCallFinished struct {
    95		EventImpl
    96		Correlation_ Correlation `json:"correlation"`
    97		StopReason   string      `json:"stop_reason,omitempty"`
    98		FinishClass  string      `json:"finish_class,omitempty"`
    99		Usage        *Usage      `json:"usage,omitempty"`
   100		DurationMs   *int64      `json:"duration_ms,omitempty"`
   101		HasToolCalls bool        `json:"has_tool_calls,omitempty"`
   102	}
   103	
   104	func NewProviderCallFinishedEvent(metadata EventMetadata, corr Correlation, stopReason, finishClass string, usage *Usage, durationMs *int64, hasToolCalls bool) *EventProviderCallFinished {
   105		return &EventProviderCallFinished{EventImpl: EventImpl{Type_: EventTypeProviderCallFinished, Metadata_: metadata}, Correlation_: corr, StopReason: stopReason, FinishClass: finishClass, Usage: usage, DurationMs: durationMs, HasToolCalls: hasToolCalls}
   106	}
   107	
   108	func (e *EventProviderCallFinished) Correlation() Correlation { return e.Correlation_ }
   109	
   110	var _ CorrelatedEvent = &EventProviderCallFinished{}
   111	
   112	type EventTextSegmentStarted struct {
   113		EventImpl
   114		Correlation_ Correlation `json:"correlation"`
   115		Role         string      `json:"role,omitempty"`
   116	}
   117	
   118	func NewTextSegmentStartedEvent(metadata EventMetadata, corr Correlation, role string) *EventTextSegmentStarted {
   119		return &EventTextSegmentStarted{EventImpl: EventImpl{Type_: EventTypeTextSegmentStarted, Metadata_: metadata}, Correlation_: corr, Role: role}
   120	}
   121	
   122	func (e *EventTextSegmentStarted) Correlation() Correlation { return e.Correlation_ }
   123	
   124	var _ CorrelatedEvent = &EventTextSegmentStarted{}
   125	
   126	type EventTextDelta struct {
   127		EventImpl
   128		Correlation_ Correlation `json:"correlation"`
   129		Delta        string      `json:"delta"`
   130		Text         string      `json:"text"`
   131		Sequence     int64       `json:"sequence,omitempty"`
   132	}
   133	
   134	func NewTextDeltaEvent(metadata EventMetadata, corr Correlation, delta, text string, sequence int64) *EventTextDelta {
   135		return &EventTextDelta{EventImpl: EventImpl{Type_: EventTypeTextDelta, Metadata_: metadata}, Correlation_: corr, Delta: delta, Text: text, Sequence: sequence}
   136	}
   137	
   138	func (e *EventTextDelta) Correlation() Correlation { return e.Correlation_ }
   139	
   140	var _ CorrelatedEvent = &EventTextDelta{}
   141	
   142	type EventTextSegmentFinished struct {
   143		EventImpl
   144		Correlation_ Correlation `json:"correlation"`
   145		Text         string      `json:"text"`
   146		FinishReason string      `json:"finish_reason,omitempty"`
   147	}
   148	
   149	func NewTextSegmentFinishedEvent(metadata EventMetadata, corr Correlation, text, finishReason string) *EventTextSegmentFinished {
   150		return &EventTextSegmentFinished{EventImpl: EventImpl{Type_: EventTypeTextSegmentFinished, Metadata_: metadata}, Correlation_: corr, Text: text, FinishReason: finishReason}
   151	}
   152	
   153	func (e *EventTextSegmentFinished) Correlation() Correlation { return e.Correlation_ }
   154	
   155	var _ CorrelatedEvent = &EventTextSegmentFinished{}
   156	
   157	type EventReasoningSegmentStarted struct {
   158		EventImpl
   159		Correlation_ Correlation `json:"correlation"`
   160		Source       string      `json:"source,omitempty"`
   161	}
   162	
   163	func NewReasoningSegmentStartedEvent(metadata EventMetadata, corr Correlation, source string) *EventReasoningSegmentStarted {
   164		return &EventReasoningSegmentStarted{EventImpl: EventImpl{Type_: EventTypeReasoningSegmentStarted, Metadata_: metadata}, Correlation_: corr, Source: source}
   165	}
   166	
   167	func (e *EventReasoningSegmentStarted) Correlation() Correlation { return e.Correlation_ }
   168	
   169	var _ CorrelatedEvent = &EventReasoningSegmentStarted{}
   170	
   171	type EventReasoningDelta struct {
   172		EventImpl
   173		Correlation_ Correlation `json:"correlation"`
   174		Delta        string      `json:"delta"`
   175		Text         string      `json:"text"`
   176		Sequence     int64       `json:"sequence,omitempty"`
   177		Source       string      `json:"source,omitempty"`
   178	}
   179	
   180	func NewReasoningDeltaEvent(metadata EventMetadata, corr Correlation, delta, text string, sequence int64) *EventReasoningDelta {
   181		return NewReasoningDeltaEventWithSource(metadata, corr, "", delta, text, sequence)
   182	}
   183	
   184	func NewReasoningDeltaEventWithSource(metadata EventMetadata, corr Correlation, source, delta, text string, sequence int64) *EventReasoningDelta {
   185		return &EventReasoningDelta{EventImpl: EventImpl{Type_: EventTypeReasoningDelta, Metadata_: metadata}, Correlation_: corr, Delta: delta, Text: text, Sequence: sequence, Source: source}
   186	}
   187	
   188	func (e *EventReasoningDelta) Correlation() Correlation { return e.Correlation_ }
   189	
   190	var _ CorrelatedEvent = &EventReasoningDelta{}
   191	
   192	type EventReasoningSegmentFinished struct {
   193		EventImpl
   194		Correlation_ Correlation `json:"correlation"`
   195		Text         string      `json:"text,omitempty"`
   196		FinishReason string      `json:"finish_reason,omitempty"`
   197		Source       string      `json:"source,omitempty"`
   198	}
   199	
   200	func NewReasoningSegmentFinishedEvent(metadata EventMetadata, corr Correlation, text, finishReason string) *EventReasoningSegmentFinished {
   201		return NewReasoningSegmentFinishedEventWithSource(metadata, corr, "", text, finishReason)
   202	}
   203	
   204	func NewReasoningSegmentFinishedEventWithSource(metadata EventMetadata, corr Correlation, source, text, finishReason string) *EventReasoningSegmentFinished {
   205		return &EventReasoningSegmentFinished{EventImpl: EventImpl{Type_: EventTypeReasoningSegmentFinished, Metadata_: metadata}, Correlation_: corr, Text: text, FinishReason: finishReason, Source: source}
   206	}
   207	
   208	func (e *EventReasoningSegmentFinished) Correlation() Correlation { return e.Correlation_ }
   209	
   210	var _ CorrelatedEvent = &EventReasoningSegmentFinished{}
```

## pkg/events/canonical_tool_events.go:1-98
```go
     1	package events
     2	
     3	type EventToolCallStarted struct {
     4		EventImpl
     5		Correlation_ Correlation `json:"correlation"`
     6		ToolCallID   string      `json:"tool_call_id"`
     7		ToolName     string      `json:"tool_name,omitempty"`
     8	}
     9	
    10	func NewToolCallStartedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName string) *EventToolCallStarted {
    11		return &EventToolCallStarted{EventImpl: EventImpl{Type_: EventTypeToolCallStarted, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName}
    12	}
    13	
    14	func (e *EventToolCallStarted) Correlation() Correlation { return e.Correlation_ }
    15	
    16	var _ CorrelatedEvent = &EventToolCallStarted{}
    17	
    18	type EventToolCallArgumentsDelta struct {
    19		EventImpl
    20		Correlation_ Correlation `json:"correlation"`
    21		ToolCallID   string      `json:"tool_call_id"`
    22		Delta        string      `json:"delta"`
    23		Arguments    string      `json:"arguments"`
    24		Sequence     int64       `json:"sequence,omitempty"`
    25	}
    26	
    27	func NewToolCallArgumentsDeltaEvent(metadata EventMetadata, corr Correlation, toolCallID, delta, arguments string, sequence int64) *EventToolCallArgumentsDelta {
    28		return &EventToolCallArgumentsDelta{EventImpl: EventImpl{Type_: EventTypeToolCallArgumentsDelta, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, Delta: delta, Arguments: arguments, Sequence: sequence}
    29	}
    30	
    31	func (e *EventToolCallArgumentsDelta) Correlation() Correlation { return e.Correlation_ }
    32	
    33	var _ CorrelatedEvent = &EventToolCallArgumentsDelta{}
    34	
    35	type EventToolCallRequested struct {
    36		EventImpl
    37		Correlation_ Correlation `json:"correlation"`
    38		ToolCallID   string      `json:"tool_call_id"`
    39		ToolName     string      `json:"tool_name"`
    40		Input        string      `json:"input"`
    41	}
    42	
    43	func NewToolCallRequestedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, input string) *EventToolCallRequested {
    44		return &EventToolCallRequested{EventImpl: EventImpl{Type_: EventTypeToolCallRequested, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Input: input}
    45	}
    46	
    47	func (e *EventToolCallRequested) Correlation() Correlation { return e.Correlation_ }
    48	
    49	var _ CorrelatedEvent = &EventToolCallRequested{}
    50	
    51	type EventToolExecutionStarted struct {
    52		EventImpl
    53		Correlation_ Correlation `json:"correlation"`
    54		ToolCallID   string      `json:"tool_call_id"`
    55		ToolName     string      `json:"tool_name,omitempty"`
    56		Input        string      `json:"input,omitempty"`
    57	}
    58	
    59	func NewToolExecutionStartedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, input string) *EventToolExecutionStarted {
    60		return &EventToolExecutionStarted{EventImpl: EventImpl{Type_: EventTypeToolExecutionStarted, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Input: input}
    61	}
    62	
    63	func (e *EventToolExecutionStarted) Correlation() Correlation { return e.Correlation_ }
    64	
    65	var _ CorrelatedEvent = &EventToolExecutionStarted{}
    66	
    67	type EventToolResultReady struct {
    68		EventImpl
    69		Correlation_ Correlation `json:"correlation"`
    70		ToolCallID   string      `json:"tool_call_id"`
    71		ToolName     string      `json:"tool_name,omitempty"`
    72		Result       string      `json:"result"`
    73		Status       string      `json:"status,omitempty"`
    74	}
    75	
    76	func NewToolResultReadyEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, result, status string) *EventToolResultReady {
    77		return &EventToolResultReady{EventImpl: EventImpl{Type_: EventTypeToolResultReady, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Result: result, Status: status}
    78	}
    79	
    80	func (e *EventToolResultReady) Correlation() Correlation { return e.Correlation_ }
    81	
    82	var _ CorrelatedEvent = &EventToolResultReady{}
    83	
    84	type EventToolCallFinished struct {
    85		EventImpl
    86		Correlation_ Correlation `json:"correlation"`
    87		ToolCallID   string      `json:"tool_call_id"`
    88		ToolName     string      `json:"tool_name,omitempty"`
    89		Status       string      `json:"status,omitempty"`
    90	}
    91	
    92	func NewToolCallFinishedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, status string) *EventToolCallFinished {
    93		return &EventToolCallFinished{EventImpl: EventImpl{Type_: EventTypeToolCallFinished, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Status: status}
    94	}
    95	
    96	func (e *EventToolCallFinished) Correlation() Correlation { return e.Correlation_ }
    97	
    98	var _ CorrelatedEvent = &EventToolCallFinished{}
```

## pkg/inference/engine/run_with_result.go:32-132
```go
    32	// RunInferenceWithResult runs inference and returns a normalized canonical inference result.
    33	//
    34	// Preferred path:
    35	// - Prefer EngineWithResult when implemented by the engine.
    36	// - Otherwise call RunInference and read canonical inference_result from turn metadata.
    37	// - If an engine returns no canonical metadata, synthesize a minimal result from turn shape.
    38	func RunInferenceWithResult(ctx context.Context, eng Engine, t *turns.Turn) (*turns.Turn, *InferenceResult, error) {
    39		if eng == nil {
    40			return nil, nil, ErrEngineNil
    41		}
    42		if ctx == nil {
    43			ctx = context.Background()
    44		}
    45		preInferenceBlockCount := 0
    46		if t != nil {
    47			preInferenceBlockCount = len(t.Blocks)
    48		}
    49	
    50		if withResult, ok := eng.(EngineWithResult); ok {
    51			out, result, err := withResult.RunInferenceWithResult(ctx, t)
    52			if out == nil {
    53				out = t
    54			}
    55			if out == nil {
    56				out = &turns.Turn{}
    57			}
    58			if err != nil {
    59				return out, result, err
    60			}
    61			if result == nil {
    62				synth := SynthesizeInferenceResult(out)
    63				result = &synth
    64			} else {
    65				normalizeInferenceResult(result, out)
    66			}
    67			if setErr := turns.KeyTurnMetaInferenceResult.Set(&out.Metadata, *result); setErr != nil {
    68				return out, result, errors.Wrap(setErr, "set canonical inference_result")
    69			}
    70			if setErr := StampInferenceResultOnGeneratedBlocksFromIndex(out, *result, preInferenceBlockCount); setErr != nil {
    71				return out, result, setErr
    72			}
    73			return out, result, nil
    74		}
    75	
    76		out, err := eng.RunInference(ctx, t)
    77		if out == nil {
    78			out = t
    79		}
    80		if out == nil {
    81			out = &turns.Turn{}
    82		}
    83		if err != nil {
    84			return out, nil, err
    85		}
    86	
    87		result, ok, getErr := ExtractInferenceResult(out)
    88		if getErr != nil {
    89			return out, nil, getErr
    90		}
    91		if !ok {
    92			result = SynthesizeInferenceResult(out)
    93		} else {
    94			normalizeInferenceResult(&result, out)
    95		}
    96	
    97		if setErr := turns.KeyTurnMetaInferenceResult.Set(&out.Metadata, result); setErr != nil {
    98			return out, nil, errors.Wrap(setErr, "set canonical inference_result")
    99		}
   100		if setErr := StampInferenceResultOnGeneratedBlocksFromIndex(out, result, preInferenceBlockCount); setErr != nil {
   101			return out, nil, setErr
   102		}
   103		return out, &result, nil
   104	}
   105	
   106	// StampInferenceResultOnGeneratedBlocks projects canonical inference metadata
   107	// onto generated output blocks so downstream consumers can render per-block metadata.
   108	func StampInferenceResultOnGeneratedBlocks(t *turns.Turn, result InferenceResult) error {
   109		return StampInferenceResultOnGeneratedBlocksFromIndex(t, result, 0)
   110	}
   111	
   112	// StampInferenceResultOnGeneratedBlocksFromIndex projects canonical inference metadata
   113	// onto generated output blocks starting at startIndex.
   114	func StampInferenceResultOnGeneratedBlocksFromIndex(t *turns.Turn, result InferenceResult, startIndex int) error {
   115		if t == nil {
   116			return nil
   117		}
   118		if startIndex < 0 {
   119			startIndex = 0
   120		}
   121		if startIndex > len(t.Blocks) {
   122			startIndex = len(t.Blocks)
   123		}
   124		for i := startIndex; i < len(t.Blocks); i++ {
   125			block := &t.Blocks[i]
   126			if block.Role != turns.RoleAssistant && block.Kind != turns.BlockKindToolCall {
   127				continue
   128			}
   129			if err := turns.KeyBlockMetaInferenceResult.Set(&block.Metadata, result); err != nil {
   130				return errors.Wrapf(err, "set block inference_result index=%d", i)
   131			}
   132		}
```

## pkg/inference/engine/inference_result_metadata.go:10-51
```go
    10	// InferenceUsageFromEventUsage converts event usage metadata to canonical turn usage.
    11	func InferenceUsageFromEventUsage(u *events.Usage) *turns.InferenceUsage {
    12		if u == nil {
    13			return nil
    14		}
    15		return &turns.InferenceUsage{
    16			InputTokens:              u.InputTokens,
    17			OutputTokens:             u.OutputTokens,
    18			CachedTokens:             u.CachedTokens,
    19			CacheCreationInputTokens: u.CacheCreationInputTokens,
    20			CacheReadInputTokens:     u.CacheReadInputTokens,
    21		}
    22	}
    23	
    24	// BuildInferenceResultFromEventMetadata maps final event metadata to canonical inference_result.
    25	func BuildInferenceResultFromEventMetadata(metadata events.EventMetadata, provider string, hasToolCalls bool) InferenceResult {
    26		var maxTokens *int
    27		if metadata.MaxTokens != nil {
    28			v := *metadata.MaxTokens
    29			maxTokens = &v
    30		}
    31	
    32		ret := InferenceResult{
    33			Provider:   strings.TrimSpace(provider),
    34			Model:      strings.TrimSpace(metadata.Model),
    35			Usage:      InferenceUsageFromEventUsage(metadata.Usage),
    36			MaxTokens:  maxTokens,
    37			DurationMs: metadata.DurationMs,
    38		}
    39		if metadata.StopReason != nil {
    40			ret.StopReason = strings.TrimSpace(*metadata.StopReason)
    41		}
    42		if len(metadata.Extra) > 0 {
    43			ret.Extra = make(map[string]any, len(metadata.Extra))
    44			for k, v := range metadata.Extra {
    45				ret.Extra[k] = v
    46			}
    47		}
    48		ret.FinishClass = InferFinishClass(ret.StopReason, hasToolCalls)
    49		ret.Truncated = isTruncatedStopReason(ret.StopReason)
    50		return ret
    51	}
```

## pkg/steps/ai/openai/chat_stream.go:248-268
```go
   248		ret := chatStreamEvent{RawPayload: raw}
   249		choice := firstChoice(raw)
   250		if idx, ok := intValue(choice["index"]); ok {
   251			ret.ChoiceIndex = &idx
   252		}
   253		delta := mapValue(choice["delta"])
   254	
   255		if s, ok := stringValue(delta["content"]); ok {
   256			ret.DeltaText = s
   257		}
   258		if s, ok := stringValue(delta["reasoning"]); ok && s != "" {
   259			ret.DeltaReasoning = s
   260		} else if s, ok := stringValue(delta["reasoning_content"]); ok && s != "" {
   261			ret.DeltaReasoning = s
   262		}
   263		ret.ToolCalls = normalizeChatToolCalls(delta["tool_calls"])
   264		if usage := normalizeChatUsage(raw["usage"]); usage != nil {
   265			ret.Usage = usage
   266		}
   267		if s, ok := stringValue(choice["finish_reason"]); ok && s != "" {
   268			ret.FinishReason = &s
```

## pkg/steps/ai/openai/chat_stream.go:321-341
```go
   321	func normalizeChatUsage(v any) *chatStreamUsage {
   322		usageMap := mapValue(v)
   323		if len(usageMap) == 0 {
   324			return nil
   325		}
   326		ret := &chatStreamUsage{}
   327		if n, ok := intValue(usageMap["prompt_tokens"]); ok {
   328			ret.promptTokens = n
   329		}
   330		if n, ok := intValue(usageMap["completion_tokens"]); ok {
   331			ret.completionTokens = n
   332		}
   333		if n, ok := intValue(mapValue(usageMap["prompt_tokens_details"])["cached_tokens"]); ok {
   334			ret.cachedTokens = n
   335		}
   336		outputDetails := mapValue(usageMap["completion_tokens_details"])
   337		if n, ok := intValue(outputDetails["reasoning_tokens"]); ok {
   338			ret.reasoningTokens = n
   339		} else if n, ok := intValue(usageMap["reasoning_tokens"]); ok {
   340			ret.reasoningTokens = n
   341		}
```

## pkg/steps/ai/openai/chat_stream_reducer.go:127-260
```go
   127	func reduceOpenAIChatChunk(
   128		state openAIChatStreamState,
   129		chunk chatStreamEvent,
   130	) (openAIChatStreamState, []openAIChatStreamEffect) {
   131		state.ChunkCount++
   132	
   133		if id := stringFromRawMap(chunk.RawPayload, "id"); id != "" {
   134			state.CurrentResponseID = id
   135		}
   136		if chunk.ChoiceIndex != nil {
   137			state.CurrentChoiceIndex = cloneIntPtr(chunk.ChoiceIndex)
   138		}
   139	
   140		chunk = state.ToolCallIDTracker.Enrich(chunk)
   141		effects := []openAIChatStreamEffect{{ObserveProviderEvent: &chunk}}
   142	
   143		state, effects = reduceOpenAIChatReasoningDelta(state, chunk, effects)
   144		state, effects = reduceOpenAIChatToolDeltas(state, chunk, effects)
   145		state, effects = reduceOpenAIChatUsageAndFinish(state, chunk, effects)
   146		state, effects = reduceOpenAIChatTextDelta(state, chunk, effects)
   147	
   148		return state, effects
   149	}
   150	
   151	func reduceOpenAIChatTextDelta(
   152		state openAIChatStreamState,
   153		chunk chatStreamEvent,
   154		effects []openAIChatStreamEffect,
   155	) (openAIChatStreamState, []openAIChatStreamEffect) {
   156		if chunk.DeltaText == "" {
   157			return state, effects
   158		}
   159	
   160		state.Message += chunk.DeltaText
   161		corr := state.chatCorrelation(chunk.ChoiceIndex, events.StreamKindContent, "", nil)
   162	
   163		if !state.TextSegmentStarted {
   164			state.TextSegmentStarted = true
   165			effects = appendOpenAIChatEvent(effects, events.NewTextSegmentStartedEvent(state.Metadata, corr, "assistant"))
   166		}
   167	
   168		effects = appendOpenAIChatEvent(effects, events.NewTextDeltaEvent(state.Metadata, corr, chunk.DeltaText, state.Message, 0))
   169		return state, effects
   170	}
   171	
   172	func reduceOpenAIChatReasoningDelta(
   173		state openAIChatStreamState,
   174		chunk chatStreamEvent,
   175		effects []openAIChatStreamEffect,
   176	) (openAIChatStreamState, []openAIChatStreamEffect) {
   177		if chunk.DeltaReasoning == "" {
   178			return state, effects
   179		}
   180	
   181		corr := state.chatCorrelation(chunk.ChoiceIndex, events.StreamKindReasoning, "", nil)
   182		if !state.ReasoningSegmentStarted {
   183			state.ReasoningSegmentStarted = true
   184			effects = appendOpenAIChatEvent(effects, events.NewReasoningSegmentStartedEvent(state.Metadata, corr, "provider"))
   185		}
   186	
   187		before := len(state.Reasoning)
   188		normalized := streamhelpers.NormalizeReasoningDelta(state.Reasoning, chunk.DeltaReasoning)
   189		state.Reasoning += normalized
   190		effects = append(effects, openAIChatStreamEffect{ObserveNormalizedReason: &openAIReasoningNormalizeObservation{
   191			Chunk:            chunk,
   192			RawLength:        len(chunk.DeltaReasoning),
   193			NormalizedLength: len(normalized),
   194			TotalLength:      before + len(normalized),
   195		}})
   196		effects = appendOpenAIChatEvent(effects, events.NewReasoningDeltaEvent(state.Metadata, corr, chunk.DeltaReasoning, state.Reasoning, 0))
   197		return state, effects
   198	}
   199	
   200	func reduceOpenAIChatToolDeltas(
   201		state openAIChatStreamState,
   202		chunk chatStreamEvent,
   203		effects []openAIChatStreamEffect,
   204	) (openAIChatStreamState, []openAIChatStreamEffect) {
   205		if len(chunk.ToolCalls) == 0 {
   206			return state, effects
   207		}
   208	
   209		for _, tc := range chunk.ToolCalls {
   210			corr := state.chatCorrelation(chunk.ChoiceIndex, events.StreamKindToolCall, tc.ID, tc.Index)
   211			key := openAIChatToolStreamKey(corr, tc)
   212			if !state.StartedToolStreams[key] {
   213				state.StartedToolStreams[key] = true
   214				effects = appendOpenAIChatEvent(effects, events.NewToolCallStartedEvent(state.Metadata, corr, corr.ToolCallID, tc.Function.Name))
   215			}
   216			if tc.Function.Arguments != "" {
   217				state.ToolArgBuffers[key] += tc.Function.Arguments
   218				state.ToolArgSequences[key]++
   219				effects = appendOpenAIChatEvent(effects, events.NewToolCallArgumentsDeltaEvent(state.Metadata, corr, corr.ToolCallID, tc.Function.Arguments, state.ToolArgBuffers[key], state.ToolArgSequences[key]))
   220			}
   221		}
   222	
   223		state.ToolCallMerger.AddToolCalls(chunk.ToolCalls)
   224		return state, effects
   225	}
   226	
   227	func reduceOpenAIChatUsageAndFinish(
   228		state openAIChatStreamState,
   229		chunk chatStreamEvent,
   230		effects []openAIChatStreamEffect,
   231	) (openAIChatStreamState, []openAIChatStreamEffect) {
   232		if chunk.Usage != nil {
   233			state.UsageInputTokens = chunk.Usage.promptTokens
   234			state.UsageOutputTokens = chunk.Usage.completionTokens
   235			state.CachedTokens = chunk.Usage.cachedTokens
   236			state.ReasoningTokens = chunk.Usage.reasoningTokens
   237		}
   238		if chunk.FinishReason != nil && *chunk.FinishReason != "" {
   239			state.StopReason = chunk.FinishReason
   240		}
   241		if chunk.Usage == nil && stopReasonString(chunk.FinishReason) == "" {
   242			return state, effects
   243		}
   244	
   245		var usage *events.Usage
   246		if chunk.Usage != nil {
   247			usage = &events.Usage{
   248				InputTokens:  chunk.Usage.promptTokens,
   249				OutputTokens: chunk.Usage.completionTokens,
   250				CachedTokens: chunk.Usage.cachedTokens,
   251			}
   252		}
   253	
   254		effects = appendOpenAIChatEvent(effects, events.NewProviderCallMetadataUpdatedEvent(
   255			state.Metadata,
   256			providerCallCorrWithResponse(state.ProviderCallCorr, state.CurrentResponseID),
   257			stopReasonString(chunk.FinishReason),
   258			"",
   259			usage,
   260		))
```

## pkg/steps/ai/openai/chat_stream_reducer.go:264-349
```go
   264	func reduceOpenAIChatTerminal(
   265		state openAIChatStreamState,
   266		terminal openAIChatTerminal,
   267	) (openAIChatStreamState, []openAIChatStreamEffect) {
   268		state.Done = true
   269		var effects []openAIChatStreamEffect
   270	
   271		finishReason := stopReasonString(state.StopReason)
   272		finishClass := "completed"
   273		hasToolCalls := len(state.mergedToolCalls()) > 0
   274		emitToolRequests := false
   275	
   276		switch terminal.Kind {
   277		case openAIChatTerminalEOF:
   278			emitToolRequests = true
   279			if hasToolCalls {
   280				finishClass = "tool_calls_pending"
   281			}
   282		case openAIChatTerminalCancelled:
   283			finishReason = "cancelled"
   284			finishClass = "cancelled"
   285		case openAIChatTerminalError:
   286			finishReason = "error"
   287			finishClass = "failed"
   288			state.Failed = true
   289		}
   290	
   291		state, effects = finishOpenAIChatSegments(state, finishReason, effects)
   292	
   293		if emitToolRequests {
   294			for _, tc := range state.mergedToolCalls() {
   295				effects = appendOpenAIChatEvent(effects, events.NewToolCallRequestedEvent(
   296					state.Metadata,
   297					state.chatCorrelation(state.CurrentChoiceIndex, events.StreamKindToolCall, tc.ID, tc.Index),
   298					tc.ID,
   299					tc.Function.Name,
   300					tc.Function.Arguments,
   301				))
   302			}
   303		}
   304	
   305		switch terminal.Kind {
   306		case openAIChatTerminalEOF:
   307			// Normal completion needs no extra terminal event beyond provider-call finish.
   308		case openAIChatTerminalCancelled:
   309			effects = appendOpenAIChatEvent(effects, events.NewInterruptEvent(state.Metadata, state.Message))
   310		case openAIChatTerminalError:
   311			effects = appendOpenAIChatEvent(effects, events.NewErrorEvent(state.Metadata, terminal.Err))
   312		}
   313	
   314		effects = appendOpenAIChatEvent(effects, events.NewProviderCallFinishedEvent(
   315			state.Metadata,
   316			providerCallCorrWithResponse(state.ProviderCallCorr, state.CurrentResponseID),
   317			finishReason,
   318			finishClass,
   319			finalOpenAIChatUsage(state),
   320			state.Metadata.DurationMs,
   321			emitToolRequests && hasToolCalls,
   322		))
   323	
   324		return state, effects
   325	}
   326	
   327	func finishOpenAIChatSegments(
   328		state openAIChatStreamState,
   329		finishReason string,
   330		effects []openAIChatStreamEffect,
   331	) (openAIChatStreamState, []openAIChatStreamEffect) {
   332		if state.ReasoningSegmentStarted && !state.ReasoningSegmentFinished {
   333			state.ReasoningSegmentFinished = true
   334			effects = appendOpenAIChatEvent(effects, events.NewReasoningSegmentFinishedEvent(
   335				state.Metadata,
   336				state.chatCorrelation(state.CurrentChoiceIndex, events.StreamKindReasoning, "", nil),
   337				state.Reasoning,
   338				finishReason,
   339			))
   340		}
   341	
   342		if state.TextSegmentStarted && !state.TextSegmentFinished {
   343			state.TextSegmentFinished = true
   344			effects = appendOpenAIChatEvent(effects, events.NewTextSegmentFinishedEvent(
   345				state.Metadata,
   346				state.chatCorrelation(state.CurrentChoiceIndex, events.StreamKindContent, "", nil),
   347				state.Message,
   348				finishReason,
   349			))
```

## pkg/steps/ai/openai/engine_openai.go:72-179
```go
    72		req, err := e.MakeCompletionRequestFromTurn(t)
    73		if err != nil {
    74			return nil, err
    75		}
    76		// RunInference always executes through the streaming path, regardless of the
    77		// profile's chat.stream default. The SSE decoder below requires an actual
    78		// streaming response body, so force the request shape here.
    79		req.Stream = true
    80		if req.StreamOptions == nil && !strings.Contains(strings.ToLower(req.Model), "mistral") {
    81			req.StreamOptions = &ChatStreamOptions{IncludeUsage: true}
    82		}
    83	
    84		// Debug: confirm adjacency constraints before sending
    85		if req != nil {
    86			// Check that any assistant message with tool_calls is followed by tool messages
    87			for i, m := range req.Messages {
    88				if len(m.ToolCalls) > 0 {
    89					missing := []string{}
    90					// Collect tool_call ids in this assistant message
    91					idset := map[string]bool{}
    92					for _, tc := range m.ToolCalls {
    93						if tc.ID != "" {
    94							idset[tc.ID] = false
    95						}
    96					}
    97					// Look ahead until next non-tool message
    98					for j := i + 1; j < len(req.Messages); j++ {
    99						nm := req.Messages[j]
   100						if nm.Role != "tool" {
   101							break
   102						}
   103						if nm.ToolCallID != "" {
   104							if _, ok := idset[nm.ToolCallID]; ok {
   105								idset[nm.ToolCallID] = true
   106							}
   107						}
   108					}
   109					for id, ok := range idset {
   110						if !ok {
   111							missing = append(missing, id)
   112						}
   113					}
   114					if len(missing) > 0 {
   115						log.Warn().
   116							Int("assistant_idx", i).
   117							Strs("missing_tool_result_ids", missing).
   118							Msg("OpenAI request: assistant tool_calls missing immediate tool results in following messages")
   119					}
   120				}
   121			}
   122		}
   123	
   124		// Add tools to the request if present in context (no Turn.Data registry).
   125		engineTools := tools.AdvertisedToolDefinitionsFromContext(ctx)
   126	
   127		var toolCfg engine.ToolConfig
   128		if t != nil {
   129			if cfg, ok, err := engine.KeyToolConfig.Get(t.Data); err != nil {
   130				return nil, errors.Wrap(err, "get tool config")
   131			} else if ok {
   132				toolCfg = cfg
   133			}
   134		}
   135	
   136		if len(engineTools) > 0 {
   137			log.Debug().Int("tool_count", len(engineTools)).Msg("Adding tools to OpenAI request")
   138	
   139			// Convert our tools to chat request tool format
   140			var openaiTools []ChatCompletionTool
   141			for _, tool := range engineTools {
   142				openaiTool := ChatCompletionTool{
   143					Type: chatToolTypeFunction,
   144					Function: &ChatFunctionDefinition{
   145						Name:        tool.Name,
   146						Description: tool.Description,
   147						Parameters:  tool.Parameters,
   148					},
   149				}
   150				openaiTools = append(openaiTools, openaiTool)
   151			}
   152	
   153			// Set tools in request
   154			req.Tools = openaiTools
   155	
   156			// Set tool choice if specified
   157			switch toolCfg.ToolChoice {
   158			case engine.ToolChoiceNone:
   159				req.ToolChoice = "none"
   160			case engine.ToolChoiceRequired:
   161				req.ToolChoice = "required"
   162			case engine.ToolChoiceAuto:
   163				req.ToolChoice = "auto"
   164			default:
   165				req.ToolChoice = "auto"
   166			}
   167	
   168			// Set parallel tool calls preference
   169			if toolCfg.MaxParallelTools > 1 {
   170				req.ParallelToolCalls = boolRef(true)
   171			} else if toolCfg.MaxParallelTools == 1 {
   172				req.ParallelToolCalls = boolRef(false)
   173			}
   174	
   175			log.Debug().
   176				Int("openai_tool_count", len(openaiTools)).
   177				Interface("tool_choice", req.ToolChoice).
   178				Interface("parallel_tool_calls", req.ParallelToolCalls).
   179				Msg("Tools added to OpenAI request")
```

## pkg/steps/ai/openai/engine_openai.go:330-340
```go
   330		toolCallCount := appendOpenAIChatTurnBlocks(t, state, includeToolCalls)
   331		log.Debug().
   332			Int("final_text_length", len(state.Message)).
   333			Int("tool_call_count", toolCallCount).
   334			Str("terminal", string(terminal.Kind)).
   335			Msg("OpenAI streaming complete, preparing messages")
   336	
   337		result := engine.BuildInferenceResultFromEventMetadata(metadata, "openai", includeToolCalls && toolCallCount > 0)
   338		settings.ApplyModelInfoCost(&result, e.settings.ModelInfo)
   339		if err := engine.PersistInferenceResult(t, result); err != nil {
   340			log.Warn().Err(err).Msg("OpenAI: failed to persist canonical inference_result")
```

## pkg/steps/ai/openai/engine_openai.go:346-370
```go
   346	func appendOpenAIChatTurnBlocks(t *turns.Turn, state openAIChatStreamState, includeToolCalls bool) int {
   347		if state.Reasoning != "" {
   348			turns.AppendBlock(t, turns.Block{
   349				ID:   uuid.NewString(),
   350				Kind: turns.BlockKindReasoning,
   351				Payload: map[string]any{
   352					turns.PayloadKeyText: state.Reasoning,
   353				},
   354			})
   355		}
   356		if state.Message != "" {
   357			turns.AppendBlock(t, turns.NewAssistantTextBlock(state.Message))
   358		}
   359		if !includeToolCalls {
   360			return 0
   361		}
   362	
   363		mergedToolCalls := state.mergedToolCalls()
   364		for _, tc := range mergedToolCalls {
   365			var args any
   366			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
   367			corr := state.chatCorrelation(state.CurrentChoiceIndex, events.StreamKindToolCall, tc.ID, tc.Index)
   368			turns.AppendBlock(t, toolblocks.NewToolCallBlockWithCorrelation(tc.ID, tc.Function.Name, args, corr))
   369		}
   370		return len(mergedToolCalls)
```

## pkg/steps/ai/openai/engine_openai.go:412-429
```go
   412	func finalizeOpenAIChatMetadata(metadata events.EventMetadata, state openAIChatStreamState, startTime time.Time) events.EventMetadata {
   413		if state.UsageInputTokens > 0 || state.UsageOutputTokens > 0 || state.CachedTokens > 0 {
   414			if metadata.Usage == nil {
   415				metadata.Usage = &events.Usage{}
   416			}
   417			metadata.Usage.InputTokens = state.UsageInputTokens
   418			metadata.Usage.OutputTokens = state.UsageOutputTokens
   419			metadata.Usage.CachedTokens = state.CachedTokens
   420		}
   421		if metadata.Extra == nil {
   422			metadata.Extra = map[string]any{}
   423		}
   424		metadata.Extra["thinking_text"] = state.Reasoning
   425		metadata.Extra["saying_text"] = state.Message
   426		if state.ReasoningTokens > 0 {
   427			metadata.Extra["reasoning_tokens"] = state.ReasoningTokens
   428		}
   429		metadata.StopReason = state.StopReason
```

## pkg/steps/ai/openai/helpers.go:135-180
```go
   135			engine = *chatSettings.Engine
   136		} else {
   137			return nil, errors.New("no engine specified")
   138		}
   139	
   140		var msgs_ []ChatCompletionMessage
   141	
   142		// Accumulate tool-calls and ensure any tool results are placed immediately after
   143		pendingToolCalls := []ChatToolCall{}
   144		pendingReasoningContent := ""
   145		toolPhaseActive := false // true after flushing assistant tool_calls, until we exit tool result sequence
   146		delayedChats := []ChatCompletionMessage{}
   147		// Track expected tool_call ids after a flush so we do not end the tool phase
   148		// until all of them have corresponding tool messages. This prevents interleaving
   149		// user/system messages between tool results when external middleware injects blocks.
   150		expectedToolIDs := map[string]bool{}
   151		remainingExpected := 0
   152		flushToolCalls := func() {
   153			if len(pendingToolCalls) == 0 {
   154				return
   155			}
   156			msgs_ = append(msgs_, ChatCompletionMessage{
   157				Role:             "assistant",
   158				ReasoningContent: pendingReasoningContent,
   159				ToolCalls:        pendingToolCalls,
   160			})
   161			// Enter tool phase and prepare expected ids; clear pending calls so we don't re-emit them later
   162			expectedToolIDs = map[string]bool{}
   163			for _, tc := range pendingToolCalls {
   164				if tc.ID != "" {
   165					expectedToolIDs[tc.ID] = false
   166				}
   167			}
   168			remainingExpected = len(pendingToolCalls)
   169			log.Debug().Int("expected_tool_uses", remainingExpected).Msg("OpenAI request: flushed assistant tool_calls; starting tool phase")
   170			toolPhaseActive = true
   171			pendingToolCalls = nil
   172			pendingReasoningContent = ""
   173		}
   174		endToolPhase := func() {
   175			if toolPhaseActive {
   176				// After finishing tool_use sequence, emit any delayed chat messages
   177				if len(delayedChats) > 0 {
   178					msgs_ = append(msgs_, delayedChats...)
   179					delayedChats = nil
   180				}
```

## pkg/steps/ai/openai/helpers.go:542-603
```go
   542			}
   543		}
   544	
   545		// Apply per-turn InferenceConfig overrides (Turn.Data > InferenceSettings.Inference).
   546		// ResolveInferenceConfig performs field-level merge: turn fields override engine defaults.
   547		if infCfg := infengine.ResolveInferenceConfig(t, settings.Inference); infCfg != nil {
   548			reasoning := isReasoningModelForSettings(settings, engine)
   549			// Reasoning models reject temperature/top_p; sanitize upfront.
   550			if reasoning {
   551				infCfg = infengine.SanitizeForReasoningModel(infCfg)
   552			}
   553			if infCfg.Temperature != nil {
   554				req.Temperature = float32(*infCfg.Temperature)
   555			}
   556			if infCfg.TopP != nil {
   557				req.TopP = float32(*infCfg.TopP)
   558			}
   559			if infCfg.MaxResponseTokens != nil && *infCfg.MaxResponseTokens > 0 {
   560				if reasoning {
   561					req.MaxCompletionTokens = *infCfg.MaxResponseTokens
   562				} else {
   563					req.MaxTokens = *infCfg.MaxResponseTokens
   564				}
   565			}
   566			if infCfg.Stop != nil {
   567				req.Stop = infCfg.Stop
   568			}
   569			if infCfg.Seed != nil {
   570				req.Seed = infCfg.Seed
   571			}
   572			if infCfg.ReasoningEffort != nil {
   573				req.ReasoningEffort = chatReasoningEffortValue(*infCfg.ReasoningEffort)
   574			}
   575			if infCfg.ThinkingType != nil {
   576				thinkingType, err := normalizeChatThinkingType(*infCfg.ThinkingType)
   577				if err != nil {
   578					return nil, err
   579				}
   580				if thinkingType == "" {
   581					req.Thinking = nil
   582				} else {
   583					req.Thinking = &ChatThinkingControl{Type: thinkingType}
   584				}
   585			}
   586		}
   587	
   588		// Apply OpenAI-specific per-turn overrides from Turn.Data.
   589		if oaiCfg := infengine.ResolveOpenAIInferenceConfig(t); oaiCfg != nil {
   590			// Reasoning models reject penalties and N>1; sanitize upfront.
   591			if isReasoningModelForSettings(settings, engine) {
   592				oaiCfg = infengine.SanitizeOpenAIForReasoningModel(oaiCfg)
   593			}
   594			if oaiCfg.N != nil {
   595				req.N = *oaiCfg.N
   596			}
   597			if oaiCfg.PresencePenalty != nil {
   598				req.PresencePenalty = float32(*oaiCfg.PresencePenalty)
   599			}
   600			if oaiCfg.FrequencyPenalty != nil {
   601				req.FrequencyPenalty = float32(*oaiCfg.FrequencyPenalty)
   602			}
   603		}
```

## pkg/steps/ai/openai_responses/helpers.go:180-218
```go
   180	
   181	// buildResponsesRequest constructs a minimal Responses request from Turn + settings
   182	func (e *Engine) buildResponsesRequest(t *turns.Turn) (responsesRequest, error) {
   183		s := e.settings
   184		req := responsesRequest{}
   185		if s != nil && s.Chat != nil && s.Chat.Engine != nil {
   186			req.Model = *s.Chat.Engine
   187		}
   188		req.Input = buildInputItemsFromTurn(t)
   189		if s != nil && s.Chat != nil {
   190			if s.Chat.MaxResponseTokens != nil {
   191				req.MaxOutputTokens = s.Chat.MaxResponseTokens
   192			}
   193			// Some reasoning models do not accept temperature/top_p; omit for those.
   194			allowSampling := !isResponsesReasoningModelForSettings(s, req.Model)
   195			if allowSampling && s.Chat.Temperature != nil {
   196				req.Temperature = s.Chat.Temperature
   197			}
   198			if allowSampling && s.Chat.TopP != nil {
   199				req.TopP = s.Chat.TopP
   200			}
   201			if len(s.Chat.Stop) > 0 {
   202				req.StopSequences = s.Chat.Stop
   203			}
   204		}
   205		if s != nil && s.OpenAI != nil && s.OpenAI.ReasoningEffort != nil {
   206			if req.Reasoning == nil {
   207				req.Reasoning = &reasoningParam{}
   208			}
   209			req.Reasoning.Effort = mapEffortString(*s.OpenAI.ReasoningEffort)
   210		}
   211		if s != nil && s.OpenAI != nil && s.OpenAI.ReasoningSummary != nil && *s.OpenAI.ReasoningSummary != "" {
   212			if req.Reasoning == nil {
   213				req.Reasoning = &reasoningParam{}
   214			}
   215			req.Reasoning.Summary = *s.OpenAI.ReasoningSummary
   216		}
   217		// Force include encrypted reasoning content on every request for stateless continuation.
   218		req.Include = append(req.Include, "reasoning.encrypted_content")
```

## pkg/steps/ai/openai_responses/helpers.go:242-280
```go
   242		if s != nil {
   243			engineInference = s.Inference
   244		}
   245		if infCfg := engine.ResolveInferenceConfig(t, engineInference); infCfg != nil {
   246			// Reasoning models reject temperature/top_p; sanitize upfront.
   247			if isResponsesReasoningModelForSettings(s, req.Model) {
   248				infCfg = engine.SanitizeForReasoningModel(infCfg)
   249			}
   250			if infCfg.ReasoningEffort != nil {
   251				if req.Reasoning == nil {
   252					req.Reasoning = &reasoningParam{}
   253				}
   254				req.Reasoning.Effort = mapEffortString(*infCfg.ReasoningEffort)
   255			}
   256			if infCfg.ReasoningSummary != nil && *infCfg.ReasoningSummary != "" {
   257				if req.Reasoning == nil {
   258					req.Reasoning = &reasoningParam{}
   259				}
   260				req.Reasoning.Summary = *infCfg.ReasoningSummary
   261			}
   262			if infCfg.ThinkingBudget != nil && *infCfg.ThinkingBudget > 0 {
   263				if req.Reasoning == nil {
   264					req.Reasoning = &reasoningParam{}
   265				}
   266				req.Reasoning.MaxTokens = infCfg.ThinkingBudget
   267			}
   268			if infCfg.Temperature != nil {
   269				req.Temperature = infCfg.Temperature
   270			}
   271			if infCfg.TopP != nil {
   272				req.TopP = infCfg.TopP
   273			}
   274			if infCfg.MaxResponseTokens != nil {
   275				req.MaxOutputTokens = infCfg.MaxResponseTokens
   276			}
   277			if infCfg.Stop != nil {
   278				req.StopSequences = infCfg.Stop
   279			}
   280		}
```

## pkg/steps/ai/openai_responses/helpers.go:488-518
```go
   488		reasoningItem := func(b turns.Block) (responsesInput, bool) {
   489			enc, _ := b.Payload[turns.PayloadKeyEncryptedContent].(string)
   490			summary := reasoningSummaryEntriesFromPayload(b.Payload)
   491			itemID, _ := b.Payload[turns.PayloadKeyItemID].(string)
   492	
   493			// OpenAI's public schema exposes optional reasoning_text content on reasoning
   494			// items, but live Responses requests currently reject non-empty reasoning
   495			// input content ("expected maximum length 0"). Preserve plaintext reasoning
   496			// locally for UI/debugging, but replay only encrypted_content and summaries.
   497			if enc == "" && len(summary) == 0 {
   498				return responsesInput{}, false
   499			}
   500	
   501			if summary == nil {
   502				summary = make([]any, 0)
   503			}
   504			ri := responsesInput{Type: "reasoning", Summary: &summary}
   505			// Provider item IDs are replay payload, not internal block identity. Use
   506			// the explicit item_id captured from the provider event when available;
   507			// never infer it from Block.ID, which may be a synthetic UUID or may follow
   508			// another provider's ID scheme.
   509			if strings.TrimSpace(itemID) != "" {
   510				ri.ID = itemID
   511			}
   512			if enc != "" {
   513				ri.EncryptedContent = enc
   514			}
   515			return ri, true
   516		}
   517	
   518		// Process blocks in-order so every function_call can retain its required reasoning predecessor.
```

## pkg/steps/ai/openai_responses/helpers.go:522-579
```go
   522			case turns.BlockKindReasoning:
   523				nextIdx := i + 1
   524				if nextIdx >= len(t.Blocks) {
   525					continue
   526				}
   527				next := t.Blocks[nextIdx]
   528				switch next.Kind {
   529				case turns.BlockKindLLMText:
   530					if v, ok := next.Payload[turns.PayloadKeyText]; ok && v != nil {
   531						if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
   532							msgID, _ := next.Payload[turns.PayloadKeyItemID].(string)
   533							if ri, ok := reasoningItem(b); ok {
   534								items = append(items, ri)
   535							}
   536							items = append(items, responsesInput{
   537								Type:    "message",
   538								Role:    "assistant",
   539								ID:      msgID,
   540								Content: []responsesContentPart{{Type: "output_text", Text: s}},
   541							})
   542							i = nextIdx
   543							continue
   544						}
   545					}
   546				case turns.BlockKindToolCall:
   547					if ri, ok := reasoningItem(b); ok {
   548						items = append(items, ri)
   549					}
   550					j := nextIdx
   551					for j < len(t.Blocks) {
   552						nb := t.Blocks[j]
   553						if nb.Kind == turns.BlockKindToolCall {
   554							appendFunctionCall(nb)
   555							j++
   556							continue
   557						}
   558						if nb.Kind == turns.BlockKindToolUse {
   559							appendFunctionCallOutput(nb)
   560							j++
   561							continue
   562						}
   563						break
   564					}
   565					i = j - 1
   566					continue
   567				case turns.BlockKindToolUse:
   568					// No valid immediate follower when reasoning is followed directly by tool output.
   569					// Omit reasoning to avoid provider 400s.
   570					continue
   571				case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindReasoning, turns.BlockKindOther:
   572					// No valid immediate follower; omit reasoning to avoid provider 400s.
   573					continue
   574				}
   575			case turns.BlockKindToolCall:
   576				appendFunctionCall(b)
   577			case turns.BlockKindToolUse:
   578				appendFunctionCallOutput(b)
   579			case turns.BlockKindUser, turns.BlockKindLLMText, turns.BlockKindSystem, turns.BlockKindOther:
```

## pkg/steps/ai/openai_responses/stream_events.go:140-170
```go
   140		switch providerEventType {
   141		case "response.output_item.added":
   142			if it, ok := m["item"].(map[string]any); ok {
   143				if typ, ok := it["type"].(string); ok {
   144					switch typ {
   145					case "reasoning":
   146						streamState.currentReasoningItemID = ""
   147						streamState.currentReasoningText.Reset()
   148						streamState.currentReasoningSummary.Reset()
   149						streamState.currentReasoningEncryptedContent = ""
   150						streamState.currentReasoningOutputIndex = nil
   151						streamState.currentReasoningSummaryIndex = nil
   152						streamState.currentReasoningStatus = ""
   153						if v, ok := it["id"].(string); ok && v != "" {
   154							streamState.currentReasoningItemID = v
   155						}
   156						if status, ok := it["status"].(string); ok && status != "" {
   157							streamState.currentReasoningStatus = status
   158						}
   159						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
   160							streamState.currentReasoningOutputIndex = &idx
   161						}
   162						e.publishEvent(ctx, events.NewReasoningSegmentStartedEvent(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "provider"))
   163						// Capture encrypted reasoning content when present.
   164						if enc, ok := it["encrypted_content"].(string); ok && enc != "" {
   165							streamState.currentReasoningEncryptedContent = enc
   166						}
   167					case "message":
   168						if v, ok := it["id"].(string); ok && v != "" {
   169							streamState.latestMessageItemID = v
   170						}
```

## pkg/steps/ai/openai_responses/stream_events.go:298-359
```go
   298		case "response.reasoning_summary_part.added":
   299			if itemID := itemIDFromProviderObject(m); itemID != "" {
   300				streamState.currentReasoningItemID = itemID
   301			}
   302			if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
   303				streamState.currentReasoningSummaryIndex = &idx
   304				streamState.lastReasoningSummaryIndex = &idx
   305			}
   306			// Start of a summary piece – forward as streaming info event
   307			e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary-started", providerData("openai_responses", streamState.currentResponseID, streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex)))
   308		case "response.reasoning_summary_text.delta":
   309			if itemID := itemIDFromProviderObject(m); itemID != "" {
   310				streamState.currentReasoningItemID = itemID
   311				streamState.lastReasoningItemID = itemID
   312			}
   313			if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
   314				streamState.currentReasoningSummaryIndex = &idx
   315				streamState.lastReasoningSummaryIndex = &idx
   316			}
   317			if v, ok := m["delta"].(string); ok && v != "" {
   318				before := streamState.summaryBuf.Len()
   319				normalized := streamhelpers.NormalizeReasoningSummaryDelta(streamState.summaryBuf.String(), v)
   320				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, streamState.currentResponseID, providerEventType, m, len(v), len(normalized), before+len(normalized))
   321				streamState.summaryBuf.WriteString(normalized)
   322				streamState.currentReasoningSummary.WriteString(normalized)
   323				e.publishEvent(ctx, events.NewReasoningDeltaEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "summary", normalized, streamState.summaryBuf.String(), 0))
   324			} else if s, ok := m["text"].(string); ok && s != "" {
   325				before := streamState.summaryBuf.Len()
   326				normalized := streamhelpers.NormalizeReasoningSummaryDelta(streamState.summaryBuf.String(), s)
   327				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, streamState.currentResponseID, providerEventType, m, len(s), len(normalized), before+len(normalized))
   328				streamState.summaryBuf.WriteString(normalized)
   329				streamState.currentReasoningSummary.WriteString(normalized)
   330				e.publishEvent(ctx, events.NewReasoningDeltaEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "summary", normalized, streamState.summaryBuf.String(), 0))
   331			}
   332		case "response.reasoning_summary_part.done":
   333			if itemID := itemIDFromProviderObject(m); itemID != "" {
   334				streamState.currentReasoningItemID = itemID
   335				streamState.lastReasoningItemID = itemID
   336			}
   337			if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
   338				streamState.currentReasoningSummaryIndex = &idx
   339				streamState.lastReasoningSummaryIndex = &idx
   340			}
   341			// End of a summary piece – forward as streaming info event
   342			e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary-ended", providerData("openai_responses", streamState.currentResponseID, streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex)))
   343		case "response.reasoning_text.delta":
   344			if itemID := itemIDFromProviderObject(m); itemID != "" {
   345				streamState.currentReasoningItemID = itemID
   346				streamState.lastReasoningItemID = itemID
   347			}
   348			if idx, ok := intFromProviderNumber(m["output_index"]); ok {
   349				streamState.currentReasoningOutputIndex = &idx
   350				streamState.lastReasoningOutputIndex = &idx
   351			}
   352			if d, ok := m["delta"].(string); ok && d != "" {
   353				before := streamState.thinkBuf.Len()
   354				normalized := streamhelpers.NormalizeReasoningDelta(streamState.thinkBuf.String(), d)
   355				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, streamState.currentResponseID, providerEventType, m, len(d), len(normalized), before+len(normalized))
   356				streamState.thinkBuf.WriteString(normalized)
   357				streamState.currentReasoningText.WriteString(d)
   358				e.publishEvent(ctx, events.NewReasoningDeltaEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "thinking", d, streamState.thinkBuf.String(), 0))
   359			} else if s, ok := m["text"].(string); ok && s != "" {
```

## pkg/steps/ai/openai_responses/stream_events.go:367-431
```go
   367		case "response.reasoning_text.done":
   368			if s, ok := m["text"].(string); ok && s != "" {
   369				// Done payloads can repeat already-streamed deltas for the current
   370				// item, but some providers send reasoning text only in the done
   371				// event. Backfill any missing suffix and emit the canonical
   372				// reasoning delta so live reasoning renderers see the update.
   373				backfillReasoningText(s)
   374			}
   375		case "response.output_item.done":
   376			if it, ok := m["item"].(map[string]any); ok {
   377				if typ, ok := it["type"].(string); ok {
   378					switch typ {
   379					case "reasoning":
   380						if itemID := itemIDFromProviderObject(m); itemID != "" {
   381							streamState.currentReasoningItemID = itemID
   382							streamState.lastReasoningItemID = itemID
   383						}
   384						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
   385							streamState.currentReasoningOutputIndex = &idx
   386							streamState.lastReasoningOutputIndex = &idx
   387						}
   388						// Append a reasoning block with encrypted content if present.
   389						rb := turns.Block{Kind: turns.BlockKindReasoning}
   390						payload := map[string]any{}
   391						if id, ok := it["id"].(string); ok && id != "" {
   392							rb.ID = id
   393							payload[turns.PayloadKeyItemID] = id
   394						}
   395						if streamState.currentReasoningItemID != "" && rb.ID == "" {
   396							rb.ID = streamState.currentReasoningItemID
   397							payload[turns.PayloadKeyItemID] = streamState.currentReasoningItemID
   398						}
   399						if status, ok := it["status"].(string); ok && status != "" {
   400							streamState.currentReasoningStatus = status
   401						}
   402						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
   403							streamState.currentReasoningOutputIndex = &idx
   404							streamState.lastReasoningOutputIndex = &idx
   405						}
   406						if terminalText := reasoningTextFromProviderContent(it["content"]); terminalText != "" {
   407							backfillReasoningText(terminalText)
   408						}
   409						if text := strings.TrimSpace(streamState.currentReasoningText.String()); text != "" {
   410							payload[turns.PayloadKeyText] = text
   411						}
   412						enc := streamState.currentReasoningEncryptedContent
   413						if v, ok := it["encrypted_content"].(string); ok && v != "" {
   414							enc = v
   415						}
   416						if enc != "" {
   417							payload[turns.PayloadKeyEncryptedContent] = enc
   418						}
   419						summary := reasoningSummaryEntriesFromPayload(it)
   420						if len(summary) == 0 {
   421							summary = reasoningSummaryEntriesFromText(streamState.currentReasoningSummary.String())
   422						}
   423						if len(summary) > 0 {
   424							payload[turns.PayloadKeySummary] = summary
   425						}
   426						rb.Payload = payload
   427						setOpenAIResponsesBlockMetadata(&rb, streamState.currentResponseID, streamState.currentReasoningOutputIndex, "reasoning", streamState.currentReasoningStatus)
   428						turns.AppendBlock(t, rb)
   429						finalReasoningText := strings.TrimSpace(streamState.currentReasoningText.String())
   430						finalReasoningStatus := streamState.currentReasoningStatus
   431						e.publishEvent(ctx, events.NewReasoningSegmentFinishedEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), reasoningSourceForSummaryIndex(streamState.currentReasoningSummaryIndex), finalReasoningText, finalReasoningStatus))
```

## pkg/steps/ai/openai_responses/stream_events.go:480-523
```go
   480						// finalize function_call and publish ToolCall event
   481						name := ""
   482						if v, ok := it["name"].(string); ok {
   483							name = v
   484						}
   485						callID := ""
   486						if v, ok := it["call_id"].(string); ok {
   487							callID = v
   488						}
   489						itemID := ""
   490						if v, ok := it["id"].(string); ok {
   491							itemID = v
   492						}
   493						args := ""
   494						if v, ok := it["arguments"].(string); ok && v != "" {
   495							args = v
   496						}
   497						status := ""
   498						if v, ok := it["status"].(string); ok && v != "" {
   499							status = v
   500						}
   501						var outputIndex *int
   502						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
   503							outputIndex = &idx
   504						}
   505						if pc := streamState.callsByItem[itemID]; pc != nil {
   506							if callID == "" {
   507								callID = pc.callID
   508							}
   509							if name == "" {
   510								name = pc.name
   511							}
   512							if outputIndex == nil {
   513								outputIndex = pc.outputIndex
   514							}
   515							if status == "" {
   516								status = pc.status
   517							}
   518							if args == "" {
   519								args = pc.args.String()
   520							}
   521						}
   522						if callID != "" && name != "" {
   523							e.publishEvent(ctx, events.NewToolCallRequestedEvent(metadata, toolCorr(itemID, callID, outputIndex), callID, name, args))
```

## pkg/steps/ai/openai_responses/stream_events.go:641-669
```go
   641					e.publishEvent(ctx, events.NewToolCallArgumentsDeltaEvent(metadata, toolCorr(itemID, pc.callID, pc.outputIndex), toolCorr(itemID, pc.callID, pc.outputIndex).ToolCallID, d, pc.args.String(), 0))
   642				}
   643			}
   644		case "response.function_call_arguments.done":
   645			itemID := ""
   646			if v, ok := m["item_id"].(string); ok {
   647				itemID = v
   648			}
   649			if d, ok := m["arguments"].(string); ok && d != "" {
   650				if pc := streamState.callsByItem[itemID]; pc != nil {
   651					pc.args.Reset()
   652					pc.args.WriteString(d)
   653				}
   654			}
   655		// No assistant text in this event; only arguments aggregation
   656		case "response.completed":
   657			streamState.responseCompleted = true
   658			if totals, ok := parseUsageTotalsFromEnvelope(m); ok {
   659				streamState.inputTokens = totals.inputTokens
   660				streamState.outputTokens = totals.outputTokens
   661				streamState.cachedTokens = totals.cachedTokens
   662				streamState.reasoningTokens = totals.reasoningTokens
   663			}
   664			// optional stop reason, sometimes nested
   665			if sr, ok := m["stop_reason"].(string); ok && sr != "" {
   666				streamState.stopReason = &sr
   667			} else if respObj, ok := m["response"].(map[string]any); ok {
   668				if sr, ok := respObj["stop_reason"].(string); ok && sr != "" {
   669					streamState.stopReason = &sr
```

## pkg/steps/ai/openai_responses/stream_state.go:106-135
```go
   106	func finalizeResponsesStreamMetadata(metadata events.EventMetadata, state *responsesStreamState, startTime time.Time, terminal responsesStreamTerminal) events.EventMetadata {
   107		state.applyTerminalStopReason(terminal)
   108		if state.inputTokens > 0 || state.outputTokens > 0 || state.cachedTokens > 0 {
   109			if metadata.Usage == nil {
   110				metadata.Usage = &events.Usage{}
   111			}
   112			metadata.Usage.InputTokens = state.inputTokens
   113			metadata.Usage.OutputTokens = state.outputTokens
   114			metadata.Usage.CachedTokens = state.cachedTokens
   115		}
   116		if metadata.Extra == nil {
   117			metadata.Extra = map[string]any{}
   118		}
   119		if state.reasoningTokens > 0 {
   120			metadata.Extra["reasoning_tokens"] = state.reasoningTokens
   121		}
   122		metadata.Extra["thinking_text"] = state.thinkBuf.String()
   123		metadata.Extra["saying_text"] = state.sayBuf.String()
   124		if state.summaryBuf.Len() > 0 {
   125			metadata.Extra["reasoning_summary_text"] = state.summaryBuf.String()
   126		}
   127		if state.stopReason != nil {
   128			metadata.StopReason = state.stopReason
   129		}
   130		d := time.Since(startTime).Milliseconds()
   131		dm := int64(d)
   132		metadata.DurationMs = &dm
   133		return metadata
   134	}
   135	
```

## pkg/steps/ai/openai_responses/stream_state.go:156-186
```go
   156	func appendResponsesFinalTurnBlocks(t *turns.Turn, state *responsesStreamState, includeToolCalls bool) int {
   157		if strings.TrimSpace(state.message) != "" {
   158			ab := turns.NewAssistantTextBlock(state.message)
   159			if state.latestMessageItemID != "" {
   160				if ab.Payload == nil {
   161					ab.Payload = map[string]any{}
   162				}
   163				ab.Payload[turns.PayloadKeyItemID] = state.latestMessageItemID
   164			}
   165			setOpenAIResponsesBlockMetadata(&ab, state.currentResponseID, state.latestMessageOutputIndex, "message", state.latestMessageStatus)
   166			turns.AppendBlock(t, ab)
   167		}
   168		if !includeToolCalls {
   169			return 0
   170		}
   171		for _, pc := range state.finalCalls {
   172			var args any
   173			if err := json.Unmarshal([]byte(pc.args.String()), &args); err != nil {
   174				args = map[string]any{}
   175			}
   176			b := toolblocks.NewToolCallBlockWithCorrelation(pc.callID, pc.name, args, state.toolCorrelation(pc.itemID, pc.callID, pc.outputIndex))
   177			if b.Payload == nil {
   178				b.Payload = map[string]any{}
   179			}
   180			if pc.itemID != "" {
   181				b.Payload[turns.PayloadKeyItemID] = pc.itemID
   182			}
   183			setOpenAIResponsesBlockMetadata(&b, state.currentResponseID, pc.outputIndex, "function_call", pc.status)
   184			turns.AppendBlock(t, b)
   185		}
   186		return len(state.finalCalls)
```

## pkg/steps/ai/openai_responses/streaming.go:129-141
```go
   129			e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary", data))
   130		}
   131	
   132		includeToolCalls := terminal.Kind == responsesStreamTerminalEOF
   133		toolCallCount := appendResponsesFinalTurnBlocks(t, state, includeToolCalls)
   134		persistResponsesInferenceResult(t, metadata, responsesInferenceProvider(e.settings), includeToolCalls && toolCallCount > 0, e.settings.ModelInfo)
   135		finishClass := responsesFinishClass(state, terminal, toolCallCount)
   136		stopReasonValue := ""
   137		if metadata.StopReason != nil {
   138			stopReasonValue = *metadata.StopReason
   139		}
   140		e.publishEvent(ctx, events.NewProviderCallFinishedEvent(metadata, state.providerCallCorrelation(), stopReasonValue, finishClass, metadata.Usage, metadata.DurationMs, includeToolCalls && toolCallCount > 0))
   141		if terminal.Err != nil {
```

## pkg/steps/ai/claude/helpers.go:95-145
```go
    95	
    96		// Apply provider-native structured output schema when configured.
    97		if chatSettings.IsStructuredOutputEnabled() {
    98			cfg, err := chatSettings.StructuredOutputConfig()
    99			if err != nil {
   100				if chatSettings.StructuredOutputRequireValid {
   101					return nil, err
   102				}
   103				log.Warn().Err(err).Msg("Claude request: ignoring invalid structured output configuration")
   104			} else if cfg != nil {
   105				req.OutputFormat = &api.OutputFormat{
   106					Type:   "json_schema",
   107					Name:   cfg.Name,
   108					Schema: cfg.Schema,
   109				}
   110			}
   111		}
   112	
   113		// Apply per-turn InferenceConfig overrides (Turn.Data > InferenceSettings.Inference).
   114		infCfg := infengine.ResolveInferenceConfig(t, s.Inference)
   115		if infCfg != nil {
   116			if infCfg.ThinkingBudget != nil && *infCfg.ThinkingBudget > 0 {
   117				req.Thinking = &api.ThinkingParam{
   118					Type:         "enabled",
   119					BudgetTokens: *infCfg.ThinkingBudget,
   120				}
   121			}
   122			if infCfg.Temperature != nil {
   123				v := *infCfg.Temperature
   124				req.Temperature = &v
   125			}
   126			if infCfg.TopP != nil {
   127				v := *infCfg.TopP
   128				req.TopP = &v
   129			}
   130			if infCfg.MaxResponseTokens != nil && *infCfg.MaxResponseTokens > 0 {
   131				req.MaxTokens = *infCfg.MaxResponseTokens
   132			}
   133			if infCfg.Stop != nil {
   134				req.StopSequences = infCfg.Stop
   135			}
   136		}
   137	
   138		// Re-validate Claude sampling constraints after overrides.
   139		// Claude requires at most one of temperature/top_p to be non-default.
   140		if req.Temperature != nil && req.TopP != nil {
   141			return nil, errors.New("both temperature and top_p are set (after inference overrides); Claude models require only one to be specified")
   142		}
   143		// When thinking is enabled, Claude requires temperature to be 1.0 or unset.
   144		if req.Thinking != nil && req.Temperature != nil && *req.Temperature != 1.0 {
   145			return nil, fmt.Errorf("thinking is enabled but temperature is %.2f; Claude requires temperature=1.0 (or unset) when thinking is active", *req.Temperature)
```

## pkg/steps/ai/claude/helpers.go:177-338
```go
   177		toolPhaseActive := false
   178		flushDelayed := func() {
   179			if len(delayedMsgs) > 0 {
   180				msgs = append(msgs, delayedMsgs...)
   181				delayedMsgs = nil
   182			}
   183		}
   184		systemPrompt := ""
   185		hasSystemPrompt := false
   186		if t != nil {
   187			for _, b := range t.Blocks {
   188				switch b.Kind {
   189				case turns.BlockKindSystem:
   190					text := ""
   191					if v, ok := b.Payload[turns.PayloadKeyText]; ok {
   192						if s, ok2 := v.(string); ok2 {
   193							text = s
   194						} else if bb, err := json.Marshal(v); err == nil {
   195							text = string(bb)
   196						}
   197					}
   198					if !hasSystemPrompt {
   199						systemPrompt = text
   200						hasSystemPrompt = true
   201					} else if text != "" {
   202						msg := api.Message{Role: RoleUser, Content: []api.Content{api.NewTextContent(text)}}
   203						if toolPhaseActive {
   204							delayedMsgs = append(delayedMsgs, msg)
   205						} else {
   206							msgs = append(msgs, msg)
   207						}
   208					}
   209				case turns.BlockKindUser:
   210					if orig, ok, err := turns.KeyBlockMetaClaudeOriginalContent.Get(b.Metadata); err != nil {
   211						return nil, errors.Wrap(err, "get claude original content (user block)")
   212					} else if ok && orig != nil {
   213						if arr, ok2 := orig.([]api.Content); ok2 && len(arr) > 0 {
   214							msg := api.Message{Role: RoleUser, Content: arr}
   215							if toolPhaseActive {
   216								delayedMsgs = append(delayedMsgs, msg)
   217							} else {
   218								msgs = append(msgs, msg)
   219							}
   220							break
   221						}
   222					}
   223					text := ""
   224					if v, ok := b.Payload[turns.PayloadKeyText]; ok {
   225						if s, ok2 := v.(string); ok2 {
   226							text = s
   227						} else if bb, err := json.Marshal(v); err == nil {
   228							text = string(bb)
   229						}
   230					}
   231					parts := []api.Content{}
   232					if text != "" {
   233						parts = append(parts, api.NewTextContent(text))
   234					}
   235					if imgs, ok := b.Payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
   236						for _, img := range imgs {
   237							mediaType, _ := img["media_type"].(string)
   238							if raw, ok := img["content"]; ok && raw != nil {
   239								var base64Content string
   240								switch rv := raw.(type) {
   241								case []byte:
   242									base64Content = base64.StdEncoding.EncodeToString(rv)
   243								case string:
   244									base64Content = rv
   245								}
   246								if base64Content != "" {
   247									parts = append(parts, api.NewImageContent(mediaType, base64Content))
   248								}
   249							}
   250						}
   251					}
   252					if len(parts) > 0 {
   253						msg := api.Message{Role: RoleUser, Content: parts}
   254						if toolPhaseActive {
   255							delayedMsgs = append(delayedMsgs, msg)
   256						} else {
   257							msgs = append(msgs, msg)
   258						}
   259					}
   260				case turns.BlockKindLLMText:
   261					if orig, ok, err := turns.KeyBlockMetaClaudeOriginalContent.Get(b.Metadata); err != nil {
   262						return nil, errors.Wrap(err, "get claude original content (assistant block)")
   263					} else if ok && orig != nil {
   264						if arr, ok2 := orig.([]api.Content); ok2 && len(arr) > 0 {
   265							msg := api.Message{Role: RoleAssistant, Content: arr}
   266							if toolPhaseActive {
   267								delayedMsgs = append(delayedMsgs, msg)
   268							} else {
   269								msgs = append(msgs, msg)
   270							}
   271							break
   272						}
   273					}
   274					text := ""
   275					if v, ok := b.Payload[turns.PayloadKeyText]; ok {
   276						if s, ok2 := v.(string); ok2 {
   277							text = s
   278						} else if bb, err := json.Marshal(v); err == nil {
   279							text = string(bb)
   280						}
   281					}
   282					if text != "" {
   283						msg := api.Message{Role: RoleAssistant, Content: []api.Content{api.NewTextContent(text)}}
   284						if toolPhaseActive {
   285							delayedMsgs = append(delayedMsgs, msg)
   286						} else {
   287							msgs = append(msgs, msg)
   288						}
   289					}
   290				case turns.BlockKindReasoning:
   291					text := ""
   292					if v, ok := b.Payload[turns.PayloadKeyText]; ok {
   293						_ = assignString(&text, v)
   294					}
   295					signature := ""
   296					if v, ok := b.Payload["signature"]; ok {
   297						_ = assignString(&signature, v)
   298					}
   299					msg := api.Message{Role: RoleAssistant, Content: []api.Content{api.NewThinkingContent(text, signature)}}
   300					if toolPhaseActive {
   301						delayedMsgs = append(delayedMsgs, msg)
   302					} else {
   303						msgs = append(msgs, msg)
   304					}
   305				case turns.BlockKindToolCall:
   306					name := ""
   307					if v, ok := b.Payload[turns.PayloadKeyName]; ok {
   308						_ = assignString(&name, v)
   309					}
   310					toolID := ""
   311					if v, ok := b.Payload[turns.PayloadKeyID]; ok {
   312						_ = assignString(&toolID, v)
   313					}
   314					argsStr := "{}"
   315					if v, ok := b.Payload[turns.PayloadKeyArgs]; ok && v != nil {
   316						switch tv := v.(type) {
   317						case string:
   318							argsStr = tv
   319						case json.RawMessage:
   320							argsStr = string(tv)
   321						default:
   322							if bb, err := json.Marshal(v); err == nil {
   323								argsStr = string(bb)
   324							}
   325						}
   326					}
   327					msgs = append(msgs, api.Message{Role: RoleAssistant, Content: []api.Content{api.NewToolUseContent(toolID, name, argsStr)}})
   328					toolPhaseActive = true
   329				case turns.BlockKindToolUse:
   330					toolID := ""
   331					_ = assignString(&toolID, b.Payload[turns.PayloadKeyID])
   332					result := toolUsePayloadToJSONString(b.Payload)
   333					msgs = append(msgs, api.Message{Role: RoleUser, Content: []api.Content{api.NewToolResultContent(toolID, result)}})
   334					flushDelayed()
   335					toolPhaseActive = false
   336				case turns.BlockKindOther:
   337					if v, ok := b.Payload[turns.PayloadKeyText]; ok {
   338						if s, ok2 := v.(string); ok2 && s != "" {
```

## pkg/steps/ai/claude/content-block-merger.go:217-252
```go
   217			cbm.updateUsage(event)
   218	
   219			cbm.providerCallCorr = events.BuildClaudeProviderCallCorrelation("claude", event.Message.ID, cbm.providerCallIndex)
   220			cbm.providerCallCorr.SessionID = cbm.metadata.SessionID
   221			cbm.providerCallCorr.RunID = cbm.metadata.InferenceID
   222			cbm.providerCallCorr.TurnID = cbm.metadata.TurnID
   223			return []events.Event{events.NewProviderCallStartedEvent(cbm.metadata, cbm.providerCallCorr)}, nil
   224	
   225		case api.MessageDeltaType:
   226			if event.Delta == nil {
   227				return nil, errors.New("MessageDeltaType event must have a delta")
   228			}
   229			if event.Delta.StopReason != "" {
   230				if cbm.metadata.Extra == nil {
   231					cbm.metadata.Extra = map[string]interface{}{}
   232				}
   233				cbm.metadata.Extra[StopReasonMetadataSlug] = event.Delta.StopReason
   234				cbm.metadata.StopReason = &event.Delta.StopReason
   235				if cbm.response != nil {
   236					cbm.response.StopReason = event.Delta.StopReason
   237				}
   238			}
   239			if event.Delta.StopSequence != "" {
   240				if cbm.metadata.Extra == nil {
   241					cbm.metadata.Extra = map[string]interface{}{}
   242				}
   243				cbm.metadata.Extra[StopSequenceMetadataSlug] = event.Delta.StopSequence
   244				if cbm.response != nil {
   245					cbm.response.StopSequence = event.Delta.StopSequence
   246				}
   247			}
   248	
   249			cbm.updateUsage(event)
   250	
   251			return []events.Event{events.NewProviderCallMetadataUpdatedEvent(cbm.metadata, cbm.providerCallCorrelation(), event.Delta.StopReason, event.Delta.StopSequence, cbm.metadata.Usage)}, nil
   252	
```

## pkg/steps/ai/claude/content-block-merger.go:290-393
```go
   290				return nil, errors.New("ContentBlockStartType event must have a message to store the finished content block")
   291			}
   292			if event.ContentBlock == nil {
   293				return nil, errors.New("ContentBlockStartType event must have a content block")
   294			}
   295			if event.Index < 0 {
   296				return nil, errors.New("ContentBlockStartType event must have a positive index")
   297			}
   298			if _, exists := cbm.contentBlocks[event.Index]; exists {
   299				return nil, errors.Errorf("ContentBlockStartType event with index %d already exists", event.Index)
   300			}
   301			cbm.contentBlocks[event.Index] = event.ContentBlock
   302	
   303			switch event.ContentBlock.Type {
   304			case api.ContentTypeText:
   305				return []events.Event{events.NewTextSegmentStartedEvent(cbm.metadata, cbm.contentBlockCorrelation(event.Index, events.SegmentTypeText), cbm.response.Role)}, nil
   306			case api.ContentTypeToolUse:
   307				corr := cbm.contentBlockCorrelation(event.Index, events.SegmentTypeTool)
   308				corr.ToolCallID = event.ContentBlock.ID
   309				return []events.Event{events.NewToolCallStartedEvent(cbm.metadata, corr, event.ContentBlock.ID, event.ContentBlock.Name)}, nil
   310			case api.ContentTypeThinking:
   311				return []events.Event{events.NewReasoningSegmentStartedEvent(cbm.metadata, cbm.contentBlockCorrelation(event.Index, events.SegmentTypeReasoning), "thinking")}, nil
   312			case api.ContentTypeImage, api.ContentTypeToolResult:
   313				return []events.Event{}, nil
   314			default:
   315				return []events.Event{}, nil
   316			}
   317	
   318		case api.ContentBlockDeltaType:
   319			if cbm.response == nil {
   320				return nil, errors.New("ContentBlockDeltaType event must have a message to store the finished content block")
   321			}
   322			if event.Delta == nil {
   323				return nil, errors.New("ContentBlockDeltaType event must have a delta")
   324			}
   325			cb, exists := cbm.contentBlocks[event.Index]
   326			if !exists {
   327				return nil, errors.Errorf("ContentBlockDeltaType event with index %d does not exist", event.Index)
   328			}
   329	
   330			cbm.updateUsage(event)
   331	
   332			delta := ""
   333			switch event.Delta.Type {
   334			case api.TextDeltaType:
   335				delta = event.Delta.Text
   336				cb.Text += event.Delta.Text
   337				return []events.Event{events.NewTextDeltaEvent(cbm.metadata, cbm.contentBlockCorrelation(event.Index, events.SegmentTypeText), delta, cb.Text, 0)}, nil
   338			case api.InputJSONDeltaType:
   339				delta = event.Delta.PartialJSON
   340				// Append to existing input string for tool use
   341				if currentInput, ok := cb.Input.(string); ok {
   342					cb.Input = currentInput + event.Delta.PartialJSON
   343				} else {
   344					cb.Input = event.Delta.PartialJSON
   345				}
   346				corr := cbm.contentBlockCorrelation(event.Index, events.SegmentTypeTool)
   347				corr.ToolCallID = cb.ID
   348				return []events.Event{events.NewToolCallArgumentsDeltaEvent(cbm.metadata, corr, cb.ID, delta, inputString(cb.Input), 0)}, nil
   349			case api.ThinkingDeltaType:
   350				delta = event.Delta.Thinking
   351				cb.Thinking += event.Delta.Thinking
   352				return []events.Event{events.NewReasoningDeltaEventWithSource(cbm.metadata, cbm.contentBlockCorrelation(event.Index, events.SegmentTypeReasoning), "thinking", delta, cb.Thinking, 0)}, nil
   353			case api.SignatureDeltaType:
   354				cb.Signature += event.Delta.Signature
   355				return []events.Event{}, nil
   356			}
   357			return []events.Event{}, nil
   358	
   359		case api.ContentBlockStopType:
   360			if cbm.response == nil {
   361				return nil, errors.New("ContentBlockStopType event must have a message to store the finished content block")
   362			}
   363			cb, exists := cbm.contentBlocks[event.Index]
   364			if !exists {
   365				return nil, errors.Errorf("ContentBlockStopType event with index %d does not exist", event.Index)
   366			}
   367			switch cb.Type {
   368			case api.ContentTypeText:
   369				cbm.response.Content = append(cbm.response.Content, api.NewTextContent(cb.Text))
   370				return []events.Event{events.NewTextSegmentFinishedEvent(cbm.metadata, cbm.contentBlockCorrelation(event.Index, events.SegmentTypeText), cb.Text, "content_block_stop")}, nil
   371	
   372			case api.ContentTypeToolUse:
   373				// Convert Input to string for API compatibility
   374				inputStr := ""
   375				if cb.Input != nil {
   376					if str, ok := cb.Input.(string); ok {
   377						inputStr = str
   378					} else {
   379						// For non-string inputs, marshal to JSON
   380						if inputBytes, err := json.Marshal(cb.Input); err == nil {
   381							inputStr = string(inputBytes)
   382						}
   383					}
   384				}
   385				cbm.response.Content = append(cbm.response.Content, api.NewToolUseContent(cb.ID, cb.Name, inputStr))
   386				corr := cbm.contentBlockCorrelation(event.Index, events.SegmentTypeTool)
   387				corr.ToolCallID = cb.ID
   388				return []events.Event{events.NewToolCallRequestedEvent(cbm.metadata, corr, cb.ID, cb.Name, inputStr)}, nil
   389	
   390			case api.ContentTypeThinking:
   391				cbm.response.Content = append(cbm.response.Content, api.NewThinkingContent(cb.Thinking, cb.Signature))
   392				return []events.Event{events.NewReasoningSegmentFinishedEventWithSource(cbm.metadata, cbm.contentBlockCorrelation(event.Index, events.SegmentTypeReasoning), "thinking", cb.Thinking, "content_block_stop")}, nil
   393	
```

## pkg/steps/ai/claude/engine_claude.go:95-120
```go
    95		// RunInference always consumes Anthropic's streaming Messages API path.
    96		// Force the request into SSE mode even when a profile was authored with
    97		// chat.stream=false; otherwise Anthropic returns a non-streaming JSON
    98		// response and the streaming parser observes a closed stream with no
    99		// message_start event.
   100		req.Stream = true
   101	
   102		// Add tools from context if present (no Turn.Data registry).
   103		if reg, ok := tools.RegistryFrom(ctx); ok && reg != nil {
   104			var claudeTools []api.Tool
   105			for _, tool := range reg.ListTools() {
   106				claudeTool := api.Tool{
   107					Name:        tool.Name,
   108					Description: tool.Description,
   109					InputSchema: tool.Parameters,
   110				}
   111				claudeTools = append(claudeTools, claudeTool)
   112				log.Trace().
   113					Str("tool_name", claudeTool.Name).
   114					Str("tool_description", claudeTool.Description).
   115					Interface("tool_input_schema", claudeTool.InputSchema).
   116					Msg("Converted tool to Claude format")
   117			}
   118			req.Tools = claudeTools
   119			log.Debug().
   120				Int("claude_tool_count", len(claudeTools)).
```

## pkg/steps/ai/claude/engine_claude.go:223-274
```go
   223		syncClaudeEventMetadata(&metadata, completionMerger.Metadata())
   224		if usage := claudeResponseUsageToEventUsage(response.Usage); usage != nil {
   225			metadata.Usage = usage
   226		}
   227		stopReason := ""
   228		if strings.TrimSpace(response.StopReason) != "" {
   229			sr := strings.TrimSpace(response.StopReason)
   230			metadata.StopReason = &sr
   231		}
   232		if metadata.StopReason != nil {
   233			stopReason = *metadata.StopReason
   234		}
   235		log.Trace().
   236			Interface("usage", metadata.Usage).
   237			Str("stop_reason", stopReason).
   238			Msg("Claude metadata finalized")
   239	
   240		// Create blocks from content blocks: text -> llm_text, tool_use -> tool_call
   241		hasToolCalls := false
   242		for i, c := range response.Content {
   243			switch v := c.(type) {
   244			case api.TextContent:
   245				if s := v.Text; s != "" {
   246					turns.AppendBlock(t, turns.NewAssistantTextBlock(s))
   247				}
   248			case api.ToolUseContent:
   249				hasToolCalls = true
   250				var args any
   251				_ = json.Unmarshal(v.Input, &args)
   252				corr := completionMerger.contentBlockCorrelation(i, events.SegmentTypeTool)
   253				corr.ToolCallID = v.ID
   254				turns.AppendBlock(t, toolblocks.NewToolCallBlockWithCorrelation(v.ID, v.Name, args, corr))
   255			case api.ThinkingContent:
   256				payload := map[string]any{}
   257				if v.Thinking != "" {
   258					payload[turns.PayloadKeyText] = v.Thinking
   259				}
   260				if v.Signature != "" {
   261					payload["signature"] = v.Signature
   262				}
   263				turns.AppendBlock(t, turns.Block{
   264					ID:      uuid.NewString(),
   265					Kind:    turns.BlockKindReasoning,
   266					Role:    turns.RoleAssistant,
   267					Payload: payload,
   268				})
   269			}
   270		}
   271	
   272		result := engine.BuildInferenceResultFromEventMetadata(metadata, "claude", hasToolCalls)
   273		settings.ApplyModelInfoCost(&result, e.settings.ModelInfo)
   274		if err := engine.PersistInferenceResult(t, result); err != nil {
```

## pkg/steps/ai/gemini/engine_gemini.go:188-238
```go
   188		if e.settings.Chat.Temperature != nil || e.settings.Chat.TopP != nil || e.settings.Chat.MaxResponseTokens != nil {
   189			cfg := genai.GenerationConfig{}
   190			if e.settings.Chat.Temperature != nil {
   191				v := float32(*e.settings.Chat.Temperature)
   192				cfg.Temperature = &v
   193			}
   194			if e.settings.Chat.TopP != nil {
   195				v := float32(*e.settings.Chat.TopP)
   196				cfg.TopP = &v
   197			}
   198			if e.settings.Chat.MaxResponseTokens != nil {
   199				// Clamp to [0, math.MaxInt32] and convert safely to int32
   200				mt := *e.settings.Chat.MaxResponseTokens
   201				var v int32
   202				if mt < 0 {
   203					log.Warn().Int("requested_max_tokens", mt).Msg("Negative MaxResponseTokens provided; clamping to 0")
   204					v = 0
   205				} else if mt > int(math.MaxInt32) {
   206					log.Warn().Int("requested_max_tokens", mt).Int("clamped_to", int(math.MaxInt32)).Msg("MaxResponseTokens exceeds int32; clamping")
   207					v = math.MaxInt32
   208				} else {
   209					// mt is within int32 range; convert via int64 to avoid int->int32 cast warning linters
   210					mt64 := int64(mt)
   211					v = int32(mt64) // #nosec G115
   212				}
   213				cfg.MaxOutputTokens = &v
   214			}
   215			model.GenerationConfig = cfg
   216		}
   217	
   218		// Apply per-turn InferenceConfig overrides (Turn.Data > InferenceSettings.Inference).
   219		if infCfg := engine.ResolveInferenceConfig(t, e.settings.Inference); infCfg != nil {
   220			if infCfg.Temperature != nil {
   221				v := float32(*infCfg.Temperature)
   222				model.Temperature = &v
   223			}
   224			if infCfg.TopP != nil {
   225				v := float32(*infCfg.TopP)
   226				model.TopP = &v
   227			}
   228			if infCfg.MaxResponseTokens != nil {
   229				mt := *infCfg.MaxResponseTokens
   230				var v int32
   231				if mt < 0 {
   232					v = 0
   233				} else if mt > int(math.MaxInt32) {
   234					v = math.MaxInt32
   235				} else {
   236					v = int32(int64(mt)) // #nosec G115
   237				}
   238				model.MaxOutputTokens = &v
```

## pkg/steps/ai/gemini/engine_gemini.go:242-296
```go
   242		// Attach tools from context if present (tools + minimal parameters when safe).
   243		registry, _ := tools.RegistryFrom(ctx)
   244		if registry != nil {
   245			var toolDecls []*genai.FunctionDeclaration
   246			for _, td := range registry.ListTools() {
   247				fd := &genai.FunctionDeclaration{
   248					Name: td.Name,
   249				}
   250				// Enrich description with parameter names to guide the model
   251				desc := td.Description
   252				var paramNames []string
   253				if td.Parameters != nil && td.Parameters.Properties != nil {
   254					propsVal := reflect.ValueOf(td.Parameters.Properties)
   255					keysMethod := propsVal.MethodByName("Keys")
   256					if keysMethod.IsValid() {
   257						keys := keysMethod.Call(nil)
   258						if len(keys) == 1 {
   259							if ks, ok := keys[0].Interface().([]string); ok {
   260								paramNames = ks
   261							}
   262						}
   263					}
   264				}
   265				if len(paramNames) > 0 {
   266					desc = strings.TrimSpace(desc + " Parameters: " + strings.Join(paramNames, ", "))
   267				}
   268				fd.Description = desc
   269				// Minimal parameters to avoid 400s
   270				if ps := convertJSONSchemaToGenAI(td.Parameters); ps != nil {
   271					fd.Parameters = ps
   272				}
   273				toolDecls = append(toolDecls, fd)
   274			}
   275			if len(toolDecls) > 0 {
   276				model.Tools = []*genai.Tool{{FunctionDeclarations: toolDecls}}
   277				log.Debug().Int("gemini_tool_count", len(toolDecls)).Msg("Added tools to Gemini model")
   278			}
   279		}
   280		// Configure function calling mode if tools are present
   281		// (Removed explicit FunctionCallingConfig to maintain compatibility with SDK version)
   282		// if registry != nil { ... }
   283	
   284		// Build parts from Turn blocks (includes tool results)
   285		parts := e.buildPartsFromTurn(t)
   286	
   287		// Prepend a short, explicit tool signature hint to guide argument filling
   288		if registry != nil {
   289			if hint := buildToolSignatureHint(registry); hint != "" {
   290				parts = append([]genai.Part{genai.Text(hint)}, parts...)
   291			}
   292		}
   293	
   294		// Prepare metadata for events
   295		startTime := time.Now()
   296		metadata := events.EventMetadata{
```

## pkg/steps/ai/gemini/engine_gemini.go:433-469
```go
   433	func extractGeminiUsage(resp *genai.GenerateContentResponse) (*events.Usage, bool) {
   434		if resp == nil {
   435			return nil, false
   436		}
   437		v := reflect.ValueOf(resp)
   438		if v.Kind() == reflect.Ptr {
   439			if v.IsNil() {
   440				return nil, false
   441			}
   442			v = v.Elem()
   443		}
   444		um := v.FieldByName("UsageMetadata")
   445		if !um.IsValid() {
   446			return nil, false
   447		}
   448		if um.Kind() == reflect.Ptr {
   449			if um.IsNil() {
   450				return nil, false
   451			}
   452			um = um.Elem()
   453		}
   454		if !um.IsValid() {
   455			return nil, false
   456		}
   457	
   458		prompt := int(extractIntField(um, "PromptTokenCount"))
   459		candidates := int(extractIntField(um, "CandidatesTokenCount"))
   460		total := int(extractIntField(um, "TotalTokenCount"))
   461	
   462		// We map prompt->input and candidates->output. If candidates is missing but total exists,
   463		// keep output at 0 rather than guessing.
   464		if prompt == 0 && candidates == 0 && total == 0 {
   465			return nil, false
   466		}
   467		return &events.Usage{
   468			InputTokens:  prompt,
   469			OutputTokens: candidates,
```

## pkg/steps/ai/gemini/engine_gemini.go:498-565
```go
   498	// buildPartsFromTurn converts Turn blocks into a flat slice of genai.Part, including tool results.
   499	func (e *GeminiEngine) buildPartsFromTurn(t *turns.Turn) []genai.Part {
   500		if t == nil || len(t.Blocks) == 0 {
   501			return []genai.Part{}
   502		}
   503		// Build lookup from tool_call id to name (for FunctionResponse name)
   504		idToName := map[string]string{}
   505		for _, b := range t.Blocks {
   506			if b.Kind == turns.BlockKindToolCall {
   507				id, _ := b.Payload[turns.PayloadKeyID].(string)
   508				name, _ := b.Payload[turns.PayloadKeyName].(string)
   509				if id != "" && name != "" {
   510					idToName[id] = name
   511				}
   512			}
   513		}
   514	
   515		var parts []genai.Part
   516		for _, b := range t.Blocks {
   517			switch b.Kind {
   518			case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindLLMText, turns.BlockKindOther, turns.BlockKindReasoning:
   519				if txt, ok := b.Payload[turns.PayloadKeyText]; ok && txt != nil {
   520					switch sv := txt.(type) {
   521					case string:
   522						parts = append(parts, genai.Text(sv))
   523					case []byte:
   524						parts = append(parts, genai.Text(string(sv)))
   525					}
   526				}
   527	
   528			case turns.BlockKindToolCall:
   529				parts = append(parts, genai.FunctionCall{
   530					Name: b.Payload[turns.PayloadKeyName].(string),
   531					Args: b.Payload[turns.PayloadKeyArgs].(map[string]any),
   532				})
   533	
   534			case turns.BlockKindToolUse:
   535				// Add FunctionResponse for tool result
   536				id, _ := b.Payload[turns.PayloadKeyID].(string)
   537				res := b.Payload[turns.PayloadKeyResult]
   538				errStr, _ := b.Payload[turns.PayloadKeyError].(string)
   539				name := idToName[id]
   540				var response map[string]any
   541				switch rv := res.(type) {
   542				case string:
   543					// Attempt to parse JSON string into object; if fail, wrap
   544					var obj map[string]any
   545					if json.Unmarshal([]byte(rv), &obj) == nil {
   546						response = obj
   547					} else {
   548						response = map[string]any{"result": rv}
   549					}
   550				case map[string]any:
   551					response = rv
   552				default:
   553					bts, _ := json.Marshal(rv)
   554					var obj map[string]any
   555					if json.Unmarshal(bts, &obj) == nil {
   556						response = obj
   557					} else {
   558						response = map[string]any{"result": rv}
   559					}
   560				}
   561				if errStr != "" {
   562					response = map[string]any{"error": errStr, "result": response}
   563				}
   564				if name == "" {
   565					name = "result"
```

## pkg/steps/ai/gemini/stream_reducer.go:35-99
```go
    35	func reduceGeminiStreamResponse(metadata events.EventMetadata, state *geminiStreamState, resp *genai.GenerateContentResponse) []events.Event {
    36		if state == nil {
    37			return nil
    38		}
    39		state.chunkCount++
    40	
    41		var out []events.Event
    42		var chunkUsage *events.Usage
    43		if resp != nil {
    44			if u, ok := extractGeminiUsage(resp); ok {
    45				state.finalUsage = u
    46				chunkUsage = u
    47			}
    48		}
    49	
    50		delta := ""
    51		chunkStopReason := ""
    52		if resp != nil && len(resp.Candidates) > 0 {
    53			for _, cand := range resp.Candidates {
    54				if fr, ok := extractGeminiFinishReason(cand); ok {
    55					state.finalStopReason = fr
    56					chunkStopReason = fr
    57				}
    58				if cand.Content == nil {
    59					continue
    60				}
    61				for _, p := range cand.Content.Parts {
    62					switch v := p.(type) {
    63					case genai.Text:
    64						delta += string(v)
    65					case genai.FunctionCall:
    66						args := v.Args
    67						if args == nil {
    68							args = map[string]any{}
    69						}
    70						id := uuid.NewString()
    71						state.pendingCalls = append(state.pendingCalls, geminiPendingCall{id: id, name: v.Name, args: args})
    72						inputBytes, _ := json.Marshal(args)
    73						toolCorr := geminiToolCorrelation(state.providerCallCorr, id, state.toolCallIndex)
    74						state.toolCallIndex++
    75						out = append(out, events.NewToolCallStartedEvent(metadata, toolCorr, id, v.Name))
    76						out = append(out, events.NewToolCallRequestedEvent(metadata, toolCorr, id, v.Name, string(inputBytes)))
    77					}
    78				}
    79			}
    80		}
    81	
    82		if chunkUsage != nil || chunkStopReason != "" {
    83			out = append(out, events.NewProviderCallMetadataUpdatedEvent(metadata, state.providerCallCorr, state.finalStopReason, "", state.finalUsage))
    84		}
    85		if delta != "" {
    86			if !state.textSegmentStarted {
    87				state.textSegmentStarted = true
    88				state.textCorr = geminiSegmentCorrelation(state.providerCallCorr, "", 0, events.SegmentTypeText)
    89				out = append(out, events.NewTextSegmentStartedEvent(metadata, state.textCorr, "assistant"))
    90			}
    91			state.message += delta
    92			state.textSequence++
    93			out = append(out, events.NewTextDeltaEvent(metadata, state.textCorr, delta, state.message, state.textSequence))
    94		}
    95	
    96		return out
    97	}
```

## pkg/steps/ai/gemini/stream_helpers.go:47-94
```go
    47	func completeGeminiStream(
    48		t *turns.Turn,
    49		metadata *events.EventMetadata,
    50		state *geminiStreamState,
    51		startedAt time.Time,
    52		terminalErr error,
    53	) (engine.InferenceResult, []events.Event) {
    54		if metadata == nil {
    55			return engine.InferenceResult{}, nil
    56		}
    57		if state == nil {
    58			state = newGeminiStreamState(events.Correlation{})
    59		}
    60		if terminalErr != nil && strings.TrimSpace(state.finalStopReason) == "" {
    61			state.finalStopReason = "error"
    62		}
    63	
    64		out := make([]events.Event, 0, 3)
    65		if state.message != "" && state.textSegmentStarted {
    66			out = append(out, events.NewTextSegmentFinishedEvent(*metadata, state.textCorr, state.message, state.finalStopReason))
    67		}
    68	
    69		appendGeminiFinalTurnBlocks(t, state)
    70	
    71		durationMs := time.Since(startedAt).Milliseconds()
    72		metadata.DurationMs = &durationMs
    73		if strings.TrimSpace(state.finalStopReason) != "" {
    74			metadata.StopReason = &state.finalStopReason
    75		}
    76		if state.finalUsage != nil {
    77			metadata.Usage = state.finalUsage
    78		}
    79	
    80		hasToolCalls := len(state.pendingCalls) > 0
    81		result := engine.BuildInferenceResultFromEventMetadata(*metadata, "gemini", hasToolCalls)
    82		if terminalErr != nil {
    83			result.FinishClass = engine.InferenceFinishClassError
    84		}
    85	
    86		if terminalErr != nil {
    87			out = append(out, events.NewErrorEvent(*metadata, terminalErr))
    88		}
    89		out = append(out, events.NewProviderCallFinishedEvent(
    90			*metadata,
    91			state.providerCallCorr,
    92			state.finalStopReason,
    93			string(result.FinishClass),
    94			metadata.Usage,
```

## pkg/steps/ai/claude/token_count.go:18-126
```go
    18	}
    19	
    20	func NewTokenCounter(s *settings.InferenceSettings) *TokenCounter {
    21		return &TokenCounter{settings: s}
    22	}
    23	
    24	func (tc *TokenCounter) CountTurn(ctx context.Context, t *turns.Turn) (*tokencount.Result, error) {
    25		if tc == nil || tc.settings == nil {
    26			return nil, errors.New("claude token count: settings are required")
    27		}
    28		engine_ := &ClaudeEngine{settings: tc.settings}
    29		req, err := engine_.MakeCountTokensRequestFromTurn(ctx, t)
    30		if err != nil {
    31			return nil, err
    32		}
    33	
    34		apiType := "claude"
    35		if tc.settings.Chat != nil && tc.settings.Chat.ApiType != nil && strings.TrimSpace(string(*tc.settings.Chat.ApiType)) != "" {
    36			apiType = string(*tc.settings.Chat.ApiType)
    37		}
    38	
    39		if tc.settings.API == nil {
    40			return nil, errors.New("claude token count: api settings are required")
    41		}
    42		apiKey, ok := tc.settings.API.APIKeys["claude-api-key"]
    43		if !ok || strings.TrimSpace(apiKey) == "" {
    44			return nil, errors.New("claude token count: no claude api key configured")
    45		}
    46		baseURL, ok := tc.settings.API.BaseUrls["claude-base-url"]
    47		if !ok || strings.TrimSpace(baseURL) == "" {
    48			baseURL = "https://api.anthropic.com"
    49		}
    50	
    51		client := api.NewClient(apiKey, baseURL)
    52		if tc.settings.Client != nil && tc.settings.Client.HTTPClient != nil {
    53			client.SetHTTPClient(tc.settings.Client.HTTPClient)
    54		}
    55		res, err := client.CountTokens(ctx, req)
    56		if err != nil {
    57			return nil, err
    58		}
    59	
    60		return &tokencount.Result{
    61			Provider:    apiType,
    62			Model:       req.Model,
    63			InputTokens: res.InputTokens,
    64			Source:      tokencount.SourceProviderAPI,
    65			Endpoint:    strings.TrimRight(baseURL, "/") + "/v1/messages/count_tokens",
    66			RequestKind: "messages_count_tokens",
    67		}, nil
    68	}
    69	
    70	func (e *ClaudeEngine) MakeCountTokensRequestFromTurn(ctx context.Context, t *turns.Turn) (*api.MessageCountTokensRequest, error) {
    71		if e == nil || e.settings == nil {
    72			return nil, errors.New("claude token count: settings are required")
    73		}
    74		projection, err := e.buildMessageProjectionFromTurn(t)
    75		if err != nil {
    76			return nil, err
    77		}
    78	
    79		chatSettings := e.settings.Chat
    80		model := ""
    81		if chatSettings != nil && chatSettings.Engine != nil {
    82			model = *chatSettings.Engine
    83		}
    84		if strings.TrimSpace(model) == "" {
    85			return nil, errors.New("no engine specified")
    86		}
    87	
    88		req := &api.MessageCountTokensRequest{
    89			Model:    model,
    90			System:   projection.System,
    91			Messages: projection.Messages,
    92		}
    93	
    94		infCfg := engine.ResolveInferenceConfig(t, e.settings.Inference)
    95		if infCfg != nil && infCfg.ThinkingBudget != nil && *infCfg.ThinkingBudget > 0 {
    96			req.Thinking = &api.ThinkingParam{
    97				Type:         "enabled",
    98				BudgetTokens: *infCfg.ThinkingBudget,
    99			}
   100		}
   101	
   102		if claudeCfg := engine.ResolveClaudeInferenceConfig(t); claudeCfg != nil {
   103			if claudeCfg.UserID != nil {
   104				req.Metadata = &api.Metadata{UserID: *claudeCfg.UserID}
   105			}
   106		}
   107	
   108		if reg, ok := tools.RegistryFrom(ctx); ok && reg != nil {
   109			var claudeTools []api.Tool
   110			for _, tool := range reg.ListTools() {
   111				claudeTools = append(claudeTools, api.Tool{
   112					Name:        tool.Name,
   113					Description: tool.Description,
   114					InputSchema: tool.Parameters,
   115				})
   116			}
   117			req.Tools = claudeTools
   118		}
   119	
   120		return req, nil
   121	}
```

## pkg/steps/ai/openai_responses/token_count.go:34-111
```go
    34		return &TokenCounter{settings: s}
    35	}
    36	
    37	func (tc *TokenCounter) CountTurn(ctx context.Context, t *turns.Turn) (*tokencount.Result, error) {
    38		if tc == nil || tc.settings == nil {
    39			return nil, errors.New("openai token count: settings are required")
    40		}
    41	
    42		engine := &Engine{settings: tc.settings}
    43		reqBody, err := engine.buildResponsesRequest(t)
    44		if err != nil {
    45			return nil, err
    46		}
    47		if err := engine.attachToolsToResponsesRequest(ctx, t, &reqBody); err != nil {
    48			return nil, err
    49		}
    50	
    51		countReq := inputTokensRequest{
    52			Model:             reqBody.Model,
    53			Input:             reqBody.Input,
    54			Text:              reqBody.Text,
    55			Reasoning:         reqBody.Reasoning,
    56			Tools:             reqBody.Tools,
    57			ToolChoice:        reqBody.ToolChoice,
    58			ParallelToolCalls: reqBody.ParallelToolCalls,
    59		}
    60	
    61		payload, err := json.Marshal(countReq)
    62		if err != nil {
    63			return nil, err
    64		}
    65	
    66		apiKey := responsesAPIKey(tc.settings.API)
    67		if strings.TrimSpace(apiKey) == "" {
    68			return nil, errors.New("responses token count: no responses api key configured")
    69		}
    70	
    71		url := responsesEndpoint(tc.settings.API, "/responses/input_tokens")
    72		if err := security.ValidateOutboundURL(url, security.OutboundURLOptions{AllowHTTP: false}); err != nil {
    73			return nil, errors.Wrap(err, "invalid responses token count URL")
    74		}
    75	
    76		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
    77		if err != nil {
    78			return nil, err
    79		}
    80		req.Header.Set("Content-Type", "application/json")
    81		req.Header.Set("Authorization", "Bearer "+apiKey)
    82	
    83		httpClient := http.DefaultClient
    84		if tc.settings.Client != nil && tc.settings.Client.HTTPClient != nil {
    85			httpClient = tc.settings.Client.HTTPClient
    86		}
    87	
    88		resp, err := httpClient.Do(req)
    89		if err != nil {
    90			return nil, err
    91		}
    92		defer resp.Body.Close()
    93	
    94		body, err := io.ReadAll(resp.Body)
    95		if err != nil {
    96			return nil, err
    97		}
    98		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    99			var m map[string]any
   100			_ = json.Unmarshal(body, &m)
   101			return nil, fmt.Errorf("openai token count error: status=%d body=%v", resp.StatusCode, m)
   102		}
   103	
   104		inputTokens, err := parseOpenAIInputTokens(body)
   105		if err != nil {
   106			return nil, err
   107		}
   108	
   109		return &tokencount.Result{
   110			Provider:    string(responsesAPIType(tc.settings)),
   111			Model:       countReq.Model,
```

