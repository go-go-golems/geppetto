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

var _ cmds.WriterCommand = &EncodeCommand{}

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

type EncodeSettings struct {
	Model string `glazed.parameter:"model"`
	Codec string `glazed.parameter:"codec"`
	Input string `glazed.parameter:"input"`
}

func (cmd *EncodeCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	w io.Writer,
) error {
	s := &EncodeSettings{}
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

	// Use tokenizer to encode
	codec := getCodec(s.Model, codecStr)
	ids, _, err := codec.Encode(s.Input)
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
