package cmds

import (
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/openai/chat"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
)

type GeppettoCommandLoader struct {
}

func (g *GeppettoCommandLoader) LoadCommandFromYAML(
	s io.Reader,
	options ...cmds.CommandDescriptionOption,
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
	step, err := chat.NewStepFromYAML(buf)
	if err != nil {
		return nil, err
	}

	// check if the openai-api-key is set in viper
	openaiAPIKey := viper.GetString("openai-api-key")
	if openaiAPIKey != "" {
		step.ClientSettings.APIKey = &openaiAPIKey
	}

	chatParameterLayer, err := chat.NewParameterLayer(
		layers.WithDefaults(step.StepSettings),
	)
	if err != nil {
		return nil, err
	}

	clientParameterLayer, err := openai.NewClientParameterLayer(
		layers.WithDefaults(step.ClientSettings),
	)
	if err != nil {
		return nil, err
	}

	ls := append(scd.Layers, chatParameterLayer, clientParameterLayer)

	factories := map[string]interface{}{}
	if step != nil {
		factories["openai-chat"] = step
	}

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithShort(scd.Short),
		cmds.WithLong(scd.Long),
		cmds.WithFlags(scd.Flags...),
		cmds.WithArguments(scd.Arguments...),
		cmds.WithLayers(ls...),
	}

	description := cmds.NewCommandDescription(
		scd.Name,
		options_...,
	)
	if scd.Prompt != "" && len(scd.Messages) != 0 {
		return nil, errors.Errorf("Prompt and messages are mutually exclusive")
	}

	sq, err := NewGeppettoCommand(description, factories,
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

func (g *GeppettoCommandLoader) LoadCommandAliasFromYAML(s io.Reader, options ...alias.Option) ([]*alias.CommandAlias, error) {
	return loaders.LoadCommandAliasFromYAML(s, options...)
}