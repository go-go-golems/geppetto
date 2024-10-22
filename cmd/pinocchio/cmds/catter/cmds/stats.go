package cmds

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/clay/pkg/filefilter"
	"github.com/go-go-golems/geppetto/cmd/pinocchio/cmds/catter/pkg"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
)

type CatterStatsSettings struct {
	Stats         []string `glazed.parameter:"stats"`
	PrintFilters  bool     `glazed.parameter:"print-filters"`
	FilterYAML    string   `glazed.parameter:"filter-yaml"`
	FilterProfile string   `glazed.parameter:"filter-profile"`
	Glazed        bool     `glazed.parameter:"glazed"`
	Paths         []string `glazed.parameter:"paths"`
}

type CatterStatsCommand struct {
	*cmds.CommandDescription
}

func NewCatterStatsCommand() (*CatterStatsCommand, error) {
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, fmt.Errorf("could not create Glazed parameter layer: %w", err)
	}

	fileFilterLayer, err := filefilter.NewFileFilterParameterLayer()
	if err != nil {
		return nil, fmt.Errorf("could not create file filter parameter layer: %w", err)
	}

	return &CatterStatsCommand{
		CommandDescription: cmds.NewCommandDescription(
			"stats",
			cmds.WithShort("Print statistics for files and directories"),
			cmds.WithLong("A CLI tool to print statistics for files and directories, including token counts and sizes."),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"stats",
					parameters.ParameterTypeStringList,
					parameters.WithHelp("Types of statistics to show: overview, dir, full"),
					parameters.WithShortFlag("s"),
					parameters.WithDefault([]string{"overview"}),
				),
				parameters.NewParameterDefinition(
					"print-filters",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Print configured filters"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"filter-yaml",
					parameters.ParameterTypeString,
					parameters.WithHelp("Path to YAML file containing filter configuration"),
				),
				parameters.NewParameterDefinition(
					"filter-profile",
					parameters.ParameterTypeString,
					parameters.WithHelp("Name of the filter profile to use"),
				),
				parameters.NewParameterDefinition(
					"glazed",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Enable Glazed structured output"),
					parameters.WithDefault(true),
				),
			),
			cmds.WithArguments(
				parameters.NewParameterDefinition(
					"paths",
					parameters.ParameterTypeStringList,
					parameters.WithHelp("Paths to process"),
					parameters.WithDefault([]string{"."}),
				),
			),
			cmds.WithLayersList(
				glazedParameterLayer,
				fileFilterLayer,
			),
		),
	}, nil
}

func (c *CatterStatsCommand) RunIntoGlazeProcessor(ctx context.Context, parsedLayers *layers.ParsedLayers, gp middlewares.Processor) error {
	s := &CatterStatsSettings{}
	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
	if err != nil {
		return fmt.Errorf("error initializing settings: %w", err)
	}

	ff, err := createFileFilter(parsedLayers, s.FilterYAML, s.FilterProfile)
	if err != nil {
		return err
	}

	if len(s.Paths) < 1 {
		s.Paths = append(s.Paths, ".")
	}

	stats := pkg.NewStats()
	err = stats.ComputeStats(s.Paths, ff)
	if err != nil {
		return fmt.Errorf("error computing stats: %w", err)
	}

	config := pkg.Config{}
	for _, statType := range s.Stats {
		switch strings.ToLower(statType) {
		case "overview":
			config.OutputFlags |= pkg.OutputOverview
		case "dir":
			config.OutputFlags |= pkg.OutputDirStructure
		case "full":
			config.OutputFlags |= pkg.OutputFullStructure
		default:
			_, _ = fmt.Fprintf(os.Stderr, "Unknown stat type: %s\n", statType)
		}
	}

	if !s.Glazed {
		gp = nil
	}
	err = stats.PrintStats(config, gp)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error printing stats: %v\n", err)
	}

	return nil
}
