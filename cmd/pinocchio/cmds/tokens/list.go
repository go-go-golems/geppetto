package tokens

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/tiktoken-go/tokenizer"
)

type ListModelsCommand struct {
	*cmds.CommandDescription
}

func (c *ListModelsCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	models := []tokenizer.Model{
		tokenizer.GPT4,
		tokenizer.GPT35Turbo,
		tokenizer.TextEmbeddingAda002,
		tokenizer.TextDavinci003,
		tokenizer.TextDavinci002,
		tokenizer.CodeDavinci002,
		tokenizer.CodeDavinci001,
		tokenizer.CodeCushman002,
		tokenizer.CodeCushman001,
		tokenizer.DavinciCodex,
		tokenizer.CushmanCodex,
		tokenizer.TextDavinci001,
		tokenizer.TextCurie001,
		tokenizer.TextBabbage001,
		tokenizer.TextAda001,
		tokenizer.Davinci,
		tokenizer.Curie,
		tokenizer.Babbage,
		tokenizer.Ada,
		tokenizer.TextSimilarityDavinci001,
		tokenizer.TextSimilarityCurie001,
		tokenizer.TextSimilarityBabbage001,
		tokenizer.TextSimilarityAda001,
		tokenizer.TextSearchDavinciDoc001,
		tokenizer.TextSearchCurieDoc001,
		tokenizer.TextSearchAdaDoc001,
		tokenizer.TextSearchBabbageDoc001,
		tokenizer.CodeSearchBabbageCode001,
		tokenizer.CodeSearchAdaCode001,
		tokenizer.TextDavinciEdit001,
		tokenizer.CodeDavinciEdit001,
	}

	for _, m := range models {
		row := types.NewRow(
			types.MRP("model_name", m),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func NewListModelsCommand() (*ListModelsCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	return &ListModelsCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list-models",
			cmds.WithShort("List available models"),
			cmds.WithLayers(glazedLayer),
		),
	}, nil
}

var _ cmds.GlazeCommand = (*ListModelsCommand)(nil)

type ListCodecsCommand struct {
	*cmds.CommandDescription
}

func (l *ListCodecsCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	encodings := []tokenizer.Encoding{
		tokenizer.R50kBase,
		tokenizer.P50kBase,
		tokenizer.P50kEdit,
		tokenizer.Cl100kBase,
	}

	for _, e := range encodings {
		row := types.NewRow(
			types.MRP("codec_name", e),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

var _ cmds.GlazeCommand = (*ListCodecsCommand)(nil)

func NewListCodecsCommand() (*ListCodecsCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	return &ListCodecsCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list-codecs",
			cmds.WithShort("List available codecs"),
			cmds.WithLayers(glazedLayer),
		),
	}, nil
}
