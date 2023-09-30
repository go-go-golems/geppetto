package tokens

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	_ "github.com/tiktoken-go/tokenizer"
	"io"
	"strconv"
	"strings"
)

type EncodeCommand struct {
	*cmds.CommandDescription
}

func NewEncodeCommand() (*EncodeCommand, error) {
	return &EncodeCommand{
		CommandDescription: cmds.NewCommandDescription(
			"encode",
			cmds.WithShort("Encode data using a specific model and codec"),
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

func (cmd *EncodeCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	w io.Writer,
) error {
	// Parse input parameters and flags
	model, ok := ps["model"].(string)
	if !ok {
		return fmt.Errorf("model flag is missing or invalid")
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
		return fmt.Errorf("input flag is missing or invalid")
	}

	// Use tokenizer to encode
	codec := getCodec(model, codecStr)
	ids, _, err := codec.Encode(input)
	if err != nil {
		return fmt.Errorf("error encoding: %v", err)
	}

	var textIds []string
	for _, id := range ids {
		textIds = append(textIds, strconv.Itoa(int(id)))
	}

	// Write the result into provided io.Writer
	_, err = w.Write([]byte(strings.Join(textIds, " ")))
	if err != nil {
		return err
	}

	return nil
}

func getDefaultEncoding(model string) (string, error) {
	codecStr := ""
	if strings.HasPrefix(model, "gpt-4") || strings.HasPrefix(model, "gpt-3.5-turbo") || strings.HasPrefix(model, "text-embedding-ada-002") {
		codecStr = "cl100k_base"
	} else if strings.HasPrefix(model, "text-davinci-002") || strings.HasPrefix(model, "text-davinci-003") {
		codecStr = "p50k_base"
	} else {
		codecStr = "r50k_base"
	}
	if codecStr == "" {
		return "", fmt.Errorf("invalid model: %s", model)
	}
	return codecStr, nil
}
