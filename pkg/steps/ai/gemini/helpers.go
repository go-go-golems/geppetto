package gemini

import "strings"

func IsGeminiEngine(engine string) bool {
	return strings.HasPrefix(engine, "gemini")
}
