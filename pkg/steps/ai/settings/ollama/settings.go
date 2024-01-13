package ollama

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
)

type Settings struct {
	Mirostat      *int     `yaml:"mirostat,omitempty" glazed.parameter:"ollama-mirostat"`
	MirostatEta   *float64 `yaml:"mirostat-eta,omitempty" glazed.parameter:"ollama-mirostat-eta"`
	MirostatTau   *float64 `yaml:"mirostat-tau,omitempty" glazed.parameter:"ollama-mirostat-tau"`
	NumCtx        *int     `yaml:"num-ctx,omitempty" glazed.parameter:"ollama-num-ctx"`
	NumGqa        *int     `yaml:"num-gqa,omitempty" glazed.parameter:"ollama-num-gqa"`
	NumGpu        *int     `yaml:"num-gpu,omitempty" glazed.parameter:"ollama-num-gpu"`
	NumThread     *int     `yaml:"num-thread,omitempty" glazed.parameter:"ollama-num-thread"`
	RepeatLastN   *int     `yaml:"repeat-last-n,omitempty" glazed.parameter:"ollama-repeat-last-n"`
	RepeatPenalty *float64 `yaml:"repeat-penalty,omitempty" glazed.parameter:"ollama-repeat-penalty"`
	Temperature   *float64 `yaml:"temperature,omitempty" glazed.parameter:"ollama-temperature"`
	Seed          *int     `yaml:"seed,omitempty" glazed.parameter:"ollama-seed"`
	Stop          *string  `yaml:"stop,omitempty" glazed.parameter:"ollama-stop"`
	TfsZ          *float64 `yaml:"tfs-z,omitempty" glazed.parameter:"ollama-tfs-z"`
	NumPredict    *int     `yaml:"num-predict,omitempty" glazed.parameter:"ollama-num-predict"`
	TopK          *int     `yaml:"top-k,omitempty" glazed.parameter:"ollama-top-k"`
	TopP          *float64 `yaml:"top-p,omitempty" glazed.parameter:"ollama-top-p"`
}

func NewSettings() *Settings {
	return &Settings{}
}

func (s *Settings) Clone() *Settings {
	return clone.Clone(s).(*Settings)
}

const OllamaChatSlug = "ollama-chat"

//go:embed "chat.yaml"
var settingsYAML []byte

type ParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

func NewParameterLayer(options ...layers.ParameterLayerOptions) (*ParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ParameterLayer{
		ParameterLayerImpl: ret,
	}, nil
}
