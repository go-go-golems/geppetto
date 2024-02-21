package tokens

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"io"
	"strconv"
	"strings"
)

type DecodeCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = &DecodeCommand{}

func NewDecodeCommand() (*DecodeCommand, error) {
	return &DecodeCommand{
		CommandDescription: cmds.NewCommandDescription(
			"decode",
			cmds.WithShort("Decode data using a specific model and codec"),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"model",
					parameters.ParameterTypeString,
					parameters.WithHelp("Model used for encoding"),
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

type DecodeSettings struct {
	Model string `glazed.parameter:"model"`
	Codec string `glazed.parameter:"codec"`
	Input string `glazed.parameter:"input"`
}

func (d *DecodeCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	w io.Writer,
) error {
	s := &DecodeSettings{}
	err := parsedLayers.InitializeStructFromLayer(layers.DefaultSlug, s)
	if err != nil {
		return err
	}
	// Retrieve parsed parameters from the layers.
	codecStr := s.Codec
	if codecStr == "" {
		codecStr, err = getDefaultEncoding(s.Model)
		if err != nil {
			return fmt.Errorf("error getting default encoding: %v", err)
		}
	}

	// Get codec based on model and codec string.
	codec := getCodec(s.Model, codecStr)

	// Decode input
	var ids []uint
	for _, t := range strings.Split(s.Input, " ") {
		id, err := strconv.Atoi(t)
		if err != nil {
			return fmt.Errorf("invalid token id: %s", t)
		}
		ids = append(ids, uint(id))
	}

	text, err := codec.Decode(ids)
	if err != nil {
		return fmt.Errorf("error decoding: %v", err)
	}

	// Write the result to the provided writer
	_, err = w.Write([]byte(text))
	return err
}
