package openai_responses

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepsession "github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func (e *Engine) runStreamingInference(ctx context.Context, t *turns.Turn, httpClient *http.Client, url string, body []byte, apiKey string, metadata events.EventMetadata, tap engine.DebugTap, startTime time.Time, reqBody responsesRequest) (*turns.Turn, error) {
	resp, err := openResponsesStream(ctx, httpClient, url, body, apiKey, tap)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	var eventName string
	var dataBuf strings.Builder
	providerCallIndex := 0
	if idx, ok := gepsession.ProviderCallIndexFromContext(ctx); ok {
		providerCallIndex = idx
	}
	providerCallCorr := newResponsesProviderCallCorrelation(metadata, reqBody, providerCallIndex)
	e.publishEvent(ctx, events.NewProviderCallStartedEvent(metadata, providerCallCorr))
	streamState := newResponsesStreamState(reqBody, providerCallCorr, tap)
	log.Trace().Msg("Responses: starting SSE read loop")
	flush := func() error {
		if dataBuf.Len() == 0 {
			return nil
		}
		raw := dataBuf.String()
		dataBuf.Reset()
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return nil
		}
		if respObj, ok := m["response"].(map[string]any); ok {
			if id, ok := respObj["id"].(string); ok && id != "" {
				streamState.currentResponseID = id
			}
		}
		if id, ok := m["response_id"].(string); ok && id != "" {
			streamState.currentResponseID = id
		}
		providerEventType := normalizeResponsesEventName(eventName)
		if providerEventType == "" {
			if typ, ok := m["type"].(string); ok {
				providerEventType = normalizeResponsesEventName(typ)
			}
		}
		e.observeProviderEvent(ctx, metadata, reqBody.Model, streamState.currentResponseID, providerEventType, m)
		e.handleResponsesProviderEvent(ctx, t, metadata, reqBody, tap, streamState, eventName, providerEventType, raw, m)
		return nil
	}
	terminal := consumeResponsesSSE(ctx, reader, tap, &eventName, &dataBuf, flush)
	if terminal.Kind == responsesStreamTerminalError && streamState.streamErr == nil {
		streamState.streamErr = terminal.Err
	}
	if streamState.streamErr != nil {
		terminal = responsesStreamTerminal{Kind: responsesStreamTerminalError, Err: streamState.streamErr}
		log.Debug().Err(streamState.streamErr).Msg("Responses: stream ended with provider error")
	}
	return e.completeResponsesStream(ctx, t, metadata, startTime, terminal, streamState)
}

func missingProviderSuffix(current, full string) string {
	if full == "" || strings.HasSuffix(current, full) {
		return ""
	}
	overlap := 0
	maxOverlap := len(full)
	if len(current) < maxOverlap {
		maxOverlap = len(current)
	}
	for i := maxOverlap; i > 0; i-- {
		if strings.HasSuffix(current, full[:i]) {
			overlap = i
			break
		}
	}
	return full[overlap:]
}

func responsesChunkFromValue(v any) string {
	switch tv := v.(type) {
	case string:
		return tv
	default:
		if tv == nil {
			return ""
		}
		b, err := json.Marshal(tv)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func newResponsesProviderCallCorrelation(metadata events.EventMetadata, _ responsesRequest, providerCallIndex int) events.Correlation {
	inferenceScopeID := metadata.InferenceID
	if inferenceScopeID == "" {
		inferenceScopeID = metadata.ID.String()
	}
	corr := events.BuildProviderCallCorrelation("openai_responses", inferenceScopeID, "", providerCallIndex, "")
	corr.SessionID = metadata.SessionID
	corr.TurnID = metadata.TurnID
	return corr
}

func (e *Engine) completeResponsesStream(ctx context.Context, t *turns.Turn, metadata events.EventMetadata, startTime time.Time, terminal responsesStreamTerminal, state *responsesStreamState) (*turns.Turn, error) {
	metadata = finalizeResponsesStreamMetadata(metadata, state, startTime, terminal)
	if state.summaryBuf.Len() > 0 {
		// Publish a friendly info event with the complete summary and provider identity when available.
		data := providerData("openai_responses", state.currentResponseID, state.lastReasoningItemID, state.lastReasoningOutputIndex, state.lastReasoningSummaryIndex)
		if data == nil {
			data = map[string]any{}
		}
		data["text"] = state.summaryBuf.String()
		e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary", data))
	}

	includeToolCalls := terminal.Kind == responsesStreamTerminalEOF
	toolCallCount := appendResponsesFinalTurnBlocks(t, state, includeToolCalls)
	persistResponsesInferenceResult(t, metadata, responsesInferenceProvider(e.settings), includeToolCalls && toolCallCount > 0, e.settings.ModelInfo)
	finishClass := responsesFinishClass(state, terminal, toolCallCount)
	stopReasonValue := ""
	if metadata.StopReason != nil {
		stopReasonValue = *metadata.StopReason
	}
	e.publishEvent(ctx, events.NewProviderCallFinishedEvent(metadata, state.providerCallCorrelation(), stopReasonValue, finishClass, metadata.Usage, metadata.DurationMs, includeToolCalls && toolCallCount > 0))
	if terminal.Err != nil {
		return t, terminal.Err
	}
	return t, nil
}

func openResponsesStream(ctx context.Context, httpClient *http.Client, url string, body []byte, apiKey string, tap engine.DebugTap) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	log.Trace().Msg("Responses: initiating HTTP request (streaming)")
	if tap != nil {
		tap.OnHTTP(req, body)
	}
	// #nosec G704 -- URL is validated above with ValidateOutboundURL.
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Responses: HTTP request failed")
		if tap != nil {
			tap.OnProviderObject("http.error", map[string]any{"error": err.Error()})
		}
		return nil, err
	}
	log.Debug().Int("status", resp.StatusCode).Str("content_type", resp.Header.Get("Content-Type")).Msg("Responses: HTTP response received")
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	defer resp.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&m)
	log.Debug().Interface("error_body", m).Int("status", resp.StatusCode).Msg("Responses: HTTP error")
	if tap != nil {
		tap.OnHTTPResponse(resp, mustMarshalJSON(m))
	}
	return nil, fmt.Errorf("responses api error: status=%d body=%v", resp.StatusCode, m)
}

func consumeResponsesSSE(
	ctx context.Context,
	reader *bufio.Reader,
	tap engine.DebugTap,
	eventName *string,
	dataBuf *strings.Builder,
	flush func() error,
) responsesStreamTerminal {
	for {
		select {
		case <-ctx.Done():
			return responsesStreamTerminal{Kind: responsesStreamTerminalCancelled, Err: ctx.Err()}
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			log.Debug().Err(err).Msg("Responses: error reading SSE line")
			return responsesStreamTerminal{Kind: responsesStreamTerminalError, Err: err}
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			_ = flush()
			*eventName = ""
			if err == io.EOF {
				return responsesStreamTerminal{Kind: responsesStreamTerminalEOF}
			}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			*eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(data)
			if tap != nil {
				tap.OnSSE(*eventName, []byte(data))
			}
			if err == io.EOF {
				_ = flush()
				return responsesStreamTerminal{Kind: responsesStreamTerminalEOF}
			}
			continue
		}
		if err == io.EOF {
			_ = flush()
			return responsesStreamTerminal{Kind: responsesStreamTerminalEOF}
		}
	}
}
