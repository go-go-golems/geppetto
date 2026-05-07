package openai_responses

import "encoding/json"

type usageTotals struct {
	inputTokens     int
	outputTokens    int
	cachedTokens    int
	reasoningTokens int
}

func mustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func parseUsageTotalsFromEnvelope(envelope map[string]any) (usageTotals, bool) {
	if envelope == nil {
		return usageTotals{}, false
	}
	usage, ok := envelope["usage"].(map[string]any)
	if !ok {
		if respObj, hasResponse := envelope["response"].(map[string]any); hasResponse {
			usage, ok = respObj["usage"].(map[string]any)
		}
	}
	if !ok || usage == nil {
		return usageTotals{}, false
	}
	return parseUsageTotals(usage), true
}

func parseUsageTotalsFromResponse(resp responsesResponse) (usageTotals, bool) {
	if totals, ok := parseUsageTotalsFromRawUsage(resp.Usage); ok {
		return totals, true
	}
	if resp.Response != nil {
		if totals, ok := parseUsageTotalsFromRawUsage(resp.Response.Usage); ok {
			return totals, true
		}
	}
	return usageTotals{}, false
}

func parseUsageTotalsFromRawUsage(raw json.RawMessage) (usageTotals, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return usageTotals{}, false
	}
	var usage map[string]any
	if err := json.Unmarshal(raw, &usage); err != nil {
		return usageTotals{}, false
	}
	return parseUsageTotals(usage), true
}

func parseUsageTotals(usage map[string]any) usageTotals {
	ret := usageTotals{}
	if v, ok := toInt(usage["input_tokens"]); ok {
		ret.inputTokens = v
	}
	if v, ok := toInt(usage["output_tokens"]); ok {
		ret.outputTokens = v
	}
	if inputDetails, ok := usage["input_tokens_details"].(map[string]any); ok {
		if v, ok := toInt(inputDetails["cached_tokens"]); ok {
			ret.cachedTokens = v
		}
	} else if v, ok := toInt(usage["cached_tokens"]); ok {
		ret.cachedTokens = v
	}
	if outputDetails, ok := usage["output_tokens_details"].(map[string]any); ok {
		if v, ok := toInt(outputDetails["reasoning_tokens"]); ok {
			ret.reasoningTokens = v
		}
	} else if v, ok := toInt(usage["reasoning_tokens"]); ok {
		ret.reasoningTokens = v
	}
	return ret
}
