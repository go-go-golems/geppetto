package cmds

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/ollama"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"strings"
)

type GeppettoCommandLoader struct {
}

func (g *GeppettoCommandLoader) IsFileSupported(f fs.FS, fileName string) bool {
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
}

var _ loaders.CommandLoader = (*GeppettoCommandLoader)(nil)

func (g *GeppettoCommandLoader) loadGeppettoCommandFromReader(
	s io.Reader,
	options []cmds.CommandDescriptionOption,
	_ []alias.Option,
) ([]cmds.Command, error) {
	yamlContent, err := io.ReadAll(s)
	if err != nil {
		return nil, err
	}

	buf := strings.NewReader(string(yamlContent))
	scd := &GeppettoCommandDescription{
		Flags:     []*parameters.ParameterDefinition{},
		Arguments: []*parameters.ParameterDefinition{},
	}
	err = yaml.NewDecoder(buf).Decode(scd)
	if err != nil {
		return nil, err
	}

	// TODO(manuel, 2023-01-27): There has to be a better way to parse YAML factories
	// maybe the easiest is just going to be to make them a separate file in the bundle format, really
	// rewind to read the factories...
	buf = strings.NewReader(string(yamlContent))
	stepSettings, err := settings.NewStepSettingsFromYAML(buf)
	if err != nil {
		return nil, err
	}

	ls, err := CreateGeppettoLayers(stepSettings)
	if err != nil {
		return nil, err
	}

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithShort(scd.Short),
		cmds.WithLong(scd.Long),
		cmds.WithFlags(scd.Flags...),
		cmds.WithArguments(scd.Arguments...),
		cmds.WithLayersList(ls...),
	}

	description := cmds.NewCommandDescription(
		scd.Name,
		options_...,
	)
	if scd.Prompt != "" && len(scd.Messages) != 0 {
		return nil, errors.Errorf("Prompt and messages are mutually exclusive")
	}

	sq, err := NewGeppettoCommand(
		description,
		stepSettings,
		WithPrompt(scd.Prompt),
		WithMessages(scd.Messages),
		WithSystemPrompt(scd.SystemPrompt),
	)
	if err != nil {
		return nil, err
	}

	for _, option := range options {
		option(sq.Description())
	}

	return []cmds.Command{sq}, nil
}

func CreateGeppettoLayers(stepSettings *settings.StepSettings) ([]layers.ParameterLayer, error) {
	chatParameterLayer, err := settings.NewChatParameterLayer(
		layers.WithDefaults(stepSettings.Chat),
	)
	if err != nil {
		return nil, err
	}

	clientParameterLayer, err := settings.NewClientParameterLayer(
		layers.WithDefaults(stepSettings.Client),
	)
	if err != nil {
		return nil, err
	}

	claudeParameterLayer, err := claude.NewParameterLayer(
		layers.WithDefaults(stepSettings.Claude),
	)
	if err != nil {
		return nil, err
	}
	openaiParameterLayer, err := openai.NewParameterLayer(
		layers.WithDefaults(stepSettings.OpenAI),
	)
	if err != nil {
		return nil, err
	}

	ollamaParameterLayer, err := ollama.NewParameterLayer(
		layers.WithDefaults(stepSettings.Ollama),
	)
	if err != nil {
		return nil, err
	}

	helpersLayer, err := NewHelpersParameterLayer()
	if err != nil {
		return nil, err
	}

	return []layers.ParameterLayer{
		helpersLayer,
		chatParameterLayer, clientParameterLayer,
		claudeParameterLayer,
		openaiParameterLayer,
		ollamaParameterLayer,
	}, nil
}

func (scl *GeppettoCommandLoader) LoadCommands(
	f fs.FS, entryName string,
	options []cmds.CommandDescriptionOption,
	aliasOptions []alias.Option,
) ([]cmds.Command, error) {
	r, err := f.Open(entryName)
	if err != nil {
		return nil, err
	}
	defer func(r fs.File) {
		_ = r.Close()
	}(r)
	return loaders.LoadCommandOrAliasFromReader(
		r,
		scl.loadGeppettoCommandFromReader,
		options,
		aliasOptions)
}
