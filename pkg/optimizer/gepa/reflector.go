package gepa

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// Reflector runs the natural-language reflection step that proposes prompt mutations.
type Reflector struct {
	Engine    engine.Engine
	System    string
	Template  string
	Objective string
}

func (r *Reflector) Propose(ctx context.Context, currentInstruction string, sideInfo string) (string, string, error) {
	if r == nil || r.Engine == nil {
		return "", "", fmt.Errorf("reflector: engine is nil")
	}
	sys := strings.TrimSpace(r.System)
	if sys == "" {
		sys = "You are an expert prompt engineer."
	}
	tmpl := r.Template
	if strings.TrimSpace(tmpl) == "" {
		tmpl = DefaultReflectionPromptTemplate
	}
	if !strings.Contains(tmpl, "<curr_param>") || !strings.Contains(tmpl, "<side_info>") {
		return "", "", fmt.Errorf("reflector: template must include <curr_param> and <side_info>")
	}

	user := strings.ReplaceAll(tmpl, "<curr_param>", currentInstruction)
	user = strings.ReplaceAll(user, "<side_info>", sideInfo)
	if strings.TrimSpace(r.Objective) != "" {
		user = fmt.Sprintf("Objective:\n%s\n\n%s", strings.TrimSpace(r.Objective), user)
	}

	turn := turns.NewTurnBuilder().
		WithSystemPrompt(sys).
		WithUserPrompt(user).
		Build()

	out, err := r.Engine.RunInference(ctx, turn)
	if err != nil {
		return "", "", fmt.Errorf("reflector: inference failed: %w", err)
	}

	raw := ExtractAssistantText(out)
	proposed := extractTripleBacktickBlock(raw)
	if strings.TrimSpace(proposed) == "" {
		proposed = strings.TrimSpace(raw)
	}
	return proposed, raw, nil
}

func ExtractAssistantText(t *turns.Turn) string {
	if t == nil {
		return ""
	}
	var parts []string
	for _, b := range t.Blocks {
		if b.Kind == turns.BlockKindLLMText || b.Role == turns.RoleAssistant {
			if b.Payload != nil {
				if s, ok := b.Payload[turns.PayloadKeyText].(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						parts = append(parts, s)
					}
				}
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func extractTripleBacktickBlock(s string) string {
	start := strings.Index(s, "```")
	if start < 0 {
		return ""
	}
	rest := s[start+3:]
	end := strings.Index(rest, "```")
	if end < 0 {
		return ""
	}

	block := strings.TrimSpace(rest[:end])
	if block == "" {
		return ""
	}

	// Strip common fenced language tags only when they are on a dedicated first line.
	if i := strings.Index(block, "\n"); i > 0 {
		firstLine := strings.ToLower(strings.TrimSpace(block[:i]))
		switch firstLine {
		case "text", "txt", "markdown", "md", "yaml", "yml", "json":
			block = strings.TrimSpace(block[i+1:])
		}
	}

	return block
}
