package openai

import (
    "context"
    "encoding/json"
    "fmt"
    "bufio"
    "net/http"
    "strings"

    "github.com/go-go-golems/geppetto/pkg/turns"
)

func (e *OpenAIEngine) runResponses(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    // Build HTTP request to /v1/responses
    reqBody, err := buildResponsesRequest(e.settings, t)
    if err != nil {
        return nil, err
    }
    b, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }

    baseURL := "https://api.openai.com/v1"
    apiKey := ""
    if e.settings != nil && e.settings.API != nil {
        if v, ok := e.settings.API.BaseUrls["openai-base-url"]; ok && v != "" {
            baseURL = v
        }
        if v, ok := e.settings.API.APIKeys["openai-api-key"]; ok {
            apiKey = v
        }
    }
    url := strings.TrimRight(baseURL, "/") + "/responses"

    // Construct HTTP client and request
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(b)))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    if apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+apiKey)
    }

    // Streaming when configured
    if e.settings != nil && e.settings.Chat != nil && e.settings.Chat.Stream {
        req.Header.Set("Accept", "text/event-stream")
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()
        if resp.StatusCode < 200 || resp.StatusCode >= 300 {
            var m map[string]any
            _ = json.NewDecoder(resp.Body).Decode(&m)
            return nil, fmt.Errorf("responses api error: status=%d body=%v", resp.StatusCode, m)
        }
        // Parse SSE events and accumulate output_text
        reader := bufio.NewReader(resp.Body)
        var eventName string
        var message string
        var dataBuf strings.Builder
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
            switch eventName {
            case "response.output_text.delta":
                // Handle {"delta":"..."} or {"text":{"delta":"..."}}
                if v, ok := m["delta"].(string); ok && v != "" {
                    message += v
                } else if tv, ok := m["text"].(map[string]any); ok {
                    if d, ok := tv["delta"].(string); ok {
                        message += d
                    }
                }
            case "response.completed":
                // no-op; will break after loop ends
            }
            return nil
        }
        for {
            line, err := reader.ReadString('\n')
            if err != nil {
                break
            }
            line = strings.TrimRight(line, "\r\n")
            if line == "" {
                _ = flush()
                eventName = ""
                continue
            }
            if strings.HasPrefix(line, "event:") {
                eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
                continue
            }
            if strings.HasPrefix(line, "data:") {
                if dataBuf.Len() > 0 {
                    dataBuf.WriteByte('\n')
                }
                dataBuf.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
                continue
            }
        }
        // finalize
        if strings.TrimSpace(message) != "" {
            turns.AppendBlock(t, turns.NewAssistantTextBlock(message))
        }
        return t, nil
    }

    // Non-streaming
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        var m map[string]any
        _ = json.NewDecoder(resp.Body).Decode(&m)
        return nil, fmt.Errorf("responses api error: status=%d body=%v", resp.StatusCode, m)
    }
    var rr responsesResponse
    if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
        return nil, err
    }
    var message string
    for _, oi := range rr.Output {
        for _, c := range oi.Content {
            if c.Type == "output_text" || c.Type == "text" {
                message += c.Text
            }
        }
    }
    if strings.TrimSpace(message) != "" {
        turns.AppendBlock(t, turns.NewAssistantTextBlock(message))
    }
    return t, nil
}


