package geppetto

func cloneNestedStringAnyMap(in map[string]map[string]any) map[string]map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneJSONMap(v)
	}
	return out
}
