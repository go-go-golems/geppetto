package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	openairesponses "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openaisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func p[T any](v T) *T { return &v }

func main() {
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	imagePath := os.Getenv("IMAGE_PATH")
	if key == "" {
		panic("OPENAI_API_KEY missing")
	}
	if strings.TrimSpace(imagePath) == "" {
		panic("IMAGE_PATH missing")
	}
	imgBytes, err := os.ReadFile(imagePath)
	if err != nil {
		panic(err)
	}

	eng, err := openairesponses.NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": key},
			BaseUrls: map[string]string{"openai-base-url": "https://api.openai.com/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine:            p("gpt-5-nano"),
			Stream:            false,
			MaxResponseTokens: p(200),
		},
		OpenAI: &openaisettings.Settings{
			ReasoningEffort:  p("low"),
			ReasoningSummary: p("concise"),
		},
	})
	if err != nil {
		panic(err)
	}

	question := "What four-digit passcode is shown in the image, and what shape/color appears on the left? Answer in one short sentence."
	ask := func(label string, blocks []turns.Block) {
		turn := &turns.Turn{Blocks: blocks}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		out, err := eng.RunInference(ctx, turn)
		if err != nil {
			fmt.Printf("%s ERROR: %v\n", label, err)
			return
		}
		fmt.Printf("%s ANSWER:\n%s\n\n", label, assistantText(out))
	}

	ask("WITHOUT_IMAGE", []turns.Block{turns.NewUserTextBlock(question)})
	ask("WITH_IMAGE", []turns.Block{turns.NewUserMultimodalBlock(question, []map[string]any{{
		"media_type": "image/png",
		"content":    imgBytes,
		"detail":     "high",
	}})})
}

func assistantText(t *turns.Turn) string {
	if t == nil {
		return ""
	}
	var parts []string
	for _, b := range t.Blocks {
		if b.Kind == turns.BlockKindLLMText {
			if s, ok := b.Payload[turns.PayloadKeyText].(string); ok && strings.TrimSpace(s) != "" {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, "\n")
}
