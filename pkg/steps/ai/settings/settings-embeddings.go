package settings

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
)

type EmbeddingsSettings struct {
	// Engine is the specific model to use for embeddings (e.g. "text-embedding-3-small" for OpenAI)
	Engine *string `yaml:"engine,omitempty" glazed.parameter:"embeddings-engine"`

	// Type specifies which provider to use (ollama, openai)
	Type *string `yaml:"type,omitempty" glazed.parameter:"embeddings-type"`

	// Dimensions specifies the output dimension of the embeddings
	Dimensions *int `yaml:"dimensions,omitempty" glazed.parameter:"embeddings-dimensions"`
}

func NewEmbeddingsSettings() (*EmbeddingsSettings, error) {
	s := &EmbeddingsSettings{
		Engine:     nil,
		Type:       nil,
		Dimensions: nil,
	}

	p, err := NewEmbeddingsParameterLayer()
	if err != nil {
		return nil, err
	}
	err = p.InitializeStructFromParameterDefaults(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *EmbeddingsSettings) Clone() *EmbeddingsSettings {
	return clone.Clone(s).(*EmbeddingsSettings)
}

//go:embed "flags/embeddings.yaml"
var embeddingsSettingsYAML []byte

type EmbeddingsParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

const EmbeddingsSlug = "embeddings"

func NewEmbeddingsParameterLayer(options ...layers.ParameterLayerOptions) (*EmbeddingsParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(embeddingsSettingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &EmbeddingsParameterLayer{
		ParameterLayerImpl: ret,
	}, nil
}
