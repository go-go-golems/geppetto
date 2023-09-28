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

func (d *DecodeCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	w io.Writer,
) error {
	// Retrieve parsed parameters from the layers.
	model, ok := ps["model"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid model parameter")
	}

	var err error
	codecStr, ok := ps["codec"].(string)
	if !ok {
		codecStr, err = getDefaultEncoding(model)
		if err != nil {
			return fmt.Errorf("error getting default encoding: %v", err)
		}
	}

	input, ok := ps["input"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid input parameter")
	}

	// Get codec based on model and codec string.
	codec := getCodec(model, codecStr)

	// Decode input
	var ids []uint
	for _, t := range strings.Split(input, " ") {
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
