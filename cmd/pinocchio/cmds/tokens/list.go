package tokens

import "github.com/go-go-golems/glazed/pkg/cmds"

type ListModelsCommand struct {
	*cmds.CommandDescription
}

func NewListModelsCommand() (*ListModelsCommand, error) {
	return &ListModelsCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list-models",
			cmds.WithShort("List available models"),
		),
	}, nil
}

type ListCodecsCommand struct {
	*cmds.CommandDescription
}

func NewListCodecsCommand() (*ListCodecsCommand, error) {
	return &ListCodecsCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list-codecs",
			cmds.WithShort("List available codecs"),
		),
	}, nil
}
