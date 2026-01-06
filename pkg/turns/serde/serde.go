package serde

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// Options controls serialization behavior.
type Options struct {
	// OmitData omits Turn.Data on write
	OmitData bool
}

// NormalizeTurn applies serde defaults (best-effort) without mutating order.
func NormalizeTurn(t *turns.Turn) {
	if t == nil {
		return
	}
	for i := range t.Blocks {
		b := &t.Blocks[i]
		// Ensure payload and metadata are non-nil for stability
		if b.Payload == nil {
			b.Payload = map[string]any{}
		}
		// Synthesize assistant role for llm_text if missing
		if b.Kind == turns.BlockKindLLMText {
			if strings.TrimSpace(b.Role) == "" {
				b.Role = turns.RoleAssistant
			}
		}
	}
}

// ToYAML marshals a Turn to YAML using snake_case tags and BlockKind string enums.
func ToYAML(t *turns.Turn, opt Options) ([]byte, error) {
	if t == nil {
		return []byte("{}"), nil
	}
	// Optionally omit Data
	snapshot := *t
	if opt.OmitData {
		snapshot.Data = turns.Data{}
	}
	NormalizeTurn(&snapshot)
	return yaml.Marshal(snapshot)
}

// FromYAML unmarshals a Turn from YAML.
func FromYAML(b []byte) (*turns.Turn, error) {
	var t turns.Turn
	if err := yaml.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	NormalizeTurn(&t)
	return &t, nil
}

// SaveTurnYAML writes a Turn to a YAML file.
func SaveTurnYAML(path string, t *turns.Turn, opt Options) error {
	data, err := ToYAML(t, opt)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadTurnYAML reads a Turn from a YAML file.
func LoadTurnYAML(path string) (*turns.Turn, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return FromYAML(b)
}
