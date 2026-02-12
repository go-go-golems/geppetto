package ollama

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/huandu/go-clone"
)

type Settings struct {
	Mirostat      *int     `yaml:"mirostat,omitempty" glazed:"ollama-mirostat"`
	MirostatEta   *float64 `yaml:"mirostat-eta,omitempty" glazed:"ollama-mirostat-eta"`
	MirostatTau   *float64 `yaml:"mirostat-tau,omitempty" glazed:"ollama-mirostat-tau"`
	NumCtx        *int     `yaml:"num-ctx,omitempty" glazed:"ollama-num-ctx"`
	NumGqa        *int     `yaml:"num-gqa,omitempty" glazed:"ollama-num-gqa"`
	NumGpu        *int     `yaml:"num-gpu,omitempty" glazed:"ollama-num-gpu"`
	NumThread     *int     `yaml:"num-thread,omitempty" glazed:"ollama-num-thread"`
	RepeatLastN   *int     `yaml:"repeat-last-n,omitempty" glazed:"ollama-repeat-last-n"`
	RepeatPenalty *float64 `yaml:"repeat-penalty,omitempty" glazed:"ollama-repeat-penalty"`
	// TODO(manuel, 2024-07-03) I think this needs to be removed
	Temperature *float64 `yaml:"temperature,omitempty" glazed:"ollama-temperature"`
	Seed        *int     `yaml:"seed,omitempty" glazed:"ollama-seed"`
	// TODO(manuel, 2024-07-03) I think this needs to be removed
	Stop       []string `yaml:"stop,omitempty" glazed:"ollama-stop"`
	TfsZ       *float64 `yaml:"tfs-z,omitempty" glazed:"ollama-tfs-z"`
	NumPredict *int     `yaml:"num-predict,omitempty" glazed:"ollama-num-predict"`
	TopK       *int     `yaml:"top-k,omitempty" glazed:"ollama-top-k"`
	// TODO(manuel, 2024-07-03) I think this needs to be removed
	TopP *float64 `yaml:"top-p,omitempty" glazed:"ollama-top-p"`
}

func NewSettings() (*Settings, error) {
	s := &Settings{}

	p, err := NewParameterLayer()
	if err != nil {
		return nil, err
	}

	err = p.InitializeStructFromFieldDefaults(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Settings) Clone() *Settings {
	return clone.Clone(s).(*Settings)
}

const OllamaChatSlug = "ollama-chat"

//go:embed "chat.yaml"
var settingsYAML []byte

type ParameterLayer struct {
	*schema.SectionImpl `yaml:",inline"`
}

func NewParameterLayer(options ...schema.SectionOption) (*ParameterLayer, error) {
	ret, err := schema.NewSectionFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ParameterLayer{
		SectionImpl: ret,
	}, nil
}
