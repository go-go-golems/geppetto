package settings

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

//go:embed "flags/inference.yaml"
var inferenceSettingsYAML []byte

const AiInferenceSlug = "ai-inference"

type InferenceValueSection struct {
	*schema.SectionImpl `yaml:",inline"`
}

func NewInferenceValueSection(options ...schema.SectionOption) (*InferenceValueSection, error) {
	ret, err := schema.NewSectionFromYAML(inferenceSettingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &InferenceValueSection{
		SectionImpl: ret,
	}, nil
}
