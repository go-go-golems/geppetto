package openai_responses

func normalizeResponsesEventName(eventName string) string {
	switch eventName {
	case "response.reasoning.delta":
		return "response.reasoning_text.delta"
	case "response.reasoning.done":
		return "response.reasoning_text.done"
	default:
		return eventName
	}
}

func toInt(v any) (int, bool) {
	const maxInt = int(^uint(0) >> 1)
	const minInt = -maxInt - 1

	switch x := v.(type) {
	case float64:
		if x > float64(maxInt) || x < float64(minInt) {
			return 0, false
		}
		return int(x), true
	case float32:
		f := float64(x)
		if f > float64(maxInt) || f < float64(minInt) {
			return 0, false
		}
		return int(x), true
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		if x > int64(maxInt) || x < int64(minInt) {
			return 0, false
		}
		return int(x), true
	case uint:
		if uint64(x) > uint64(maxInt) {
			return 0, false
		}
		return int(x), true
	case uint32:
		if uint64(x) > uint64(maxInt) {
			return 0, false
		}
		return int(x), true
	case uint64:
		if x > uint64(maxInt) {
			return 0, false
		}
		return int(x), true
	default:
		return 0, false
	}
}
