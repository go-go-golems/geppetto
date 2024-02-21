package tokens

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"io"
)

type CountCommand struct {
	*cmds.CommandDescription
}

func NewCountCommand() (*CountCommand, error) {
	return &CountCommand{
		CommandDescription: cmds.NewCommandDescription(
			"count",
			cmds.WithShort("Count data entries using a specific model and codec"),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"model",
					parameters.ParameterTypeString,
					parameters.WithHelp("Model used for encoding"),
					parameters.WithDefault("gpt-4"),
				),
				parameters.NewParameterDefinition(
					"codec",
					parameters.ParameterTypeString,
					parameters.WithHelp("Codec used for encoding"),
				),
			),
			cmds.WithArguments(
				parameters.NewParameterDefinition(
					"input",
					parameters.ParameterTypeStringFromFiles,
					parameters.WithHelp("Input file"),
				),
			),
		),
	}, nil
}

type CountSettings struct {
	Model string `glazed.parameter:"model"`
	Codec string `glazed.parameter:"codec"`
	Input string `glazed.parameter:"input"`
}

var _ cmds.WriterCommand = (*CountCommand)(nil)

func (cc *CountCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	w io.Writer,
) error {
	s := &CountSettings{}
	err := parsedLayers.InitializeStructFromLayer(layers.DefaultSlug, s)
	if err != nil {
		return err
	}

	codecStr := s.Codec
	if s.Codec == "" {
		codecStr, err = getDefaultEncoding(s.Model)
		if err != nil {
			return fmt.Errorf("error getting default encoding: %v", err)
		}
	}

	// Get codec based on model and codec string.
	codec := getCodec(s.Model, codecStr)

	ids, _, err := codec.Encode(s.Input)
	if err != nil {
		return fmt.Errorf("error encoding input: %v", err)
	}

	count := len(ids)

	// Write the result to the provided writer.
	// print model and encoding
	_, err = w.Write([]byte(fmt.Sprintf("Model: %s\n", s.Model)))
	if err != nil {
		return fmt.Errorf("error writing to output: %v", err)
	}
	_, err = w.Write([]byte(fmt.Sprintf("Codec: %s\n", codecStr)))
	if err != nil {
		return fmt.Errorf("error writing to output: %v", err)
	}
	_, err = w.Write([]byte(fmt.Sprintf("Total tokens: %d\n", count)))
	if err != nil {
		return fmt.Errorf("error writing to output: %v", err)
	}

	return nil
}
