package helpers

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/spf13/cobra"
)

// TODO(manuel, 2023-11-28) Move to glazed package
func ParseLayersFromCobraCommand(cmd *cobra.Command, layers_ []cli.CobraParameterLayer) (
	map[string]*layers.ParsedParameterLayer,
	error,
) {
	ret := map[string]*layers.ParsedParameterLayer{}

	for _, layer := range layers_ {
		ps, err := layer.ParseFlagsFromCobraCommand(cmd)
		if err != nil {
			return nil, err
		}
		ret[layer.GetSlug()] = &layers.ParsedParameterLayer{
			Layer:      layer,
			Parameters: ps,
		}
	}

	return ret, nil
}
