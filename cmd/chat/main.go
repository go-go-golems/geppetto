package main

import (
	context2 "context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/spf13/cobra"
)

const veryLongLoremIpsum = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec a diam lectus. " +
	"Sed sit amet ipsum mauris. Maecenas congue ligula ac quam viverra nec consectetur ante hendrerit. " +
	"Donec et mollis dolor. Praesent et diam eget libero egestas mattis sit amet vitae augue. " +
	"Nam tincidunt congue enim, ut porta lorem lacinia consectetur. " +
	"Donec ut libero sed arcu vehicula ultricies a non tortor. Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
	"Aenean ut gravida lorem. Ut turpis felis, pulvinar a semper sed, adipiscing id dolor. " +
	"Pellentesque auctor nisi id magna consequat sagittis. " +
	"Curabitur dapibus enim sit amet elit pharetra tincidunt feugiat nisl imperdiet. " +
	"Ut convallis libero in urna ultrices accumsan. Donec sed odio eros."

type ChatCommand struct {
	*cmds.CommandDescription
}

func NewChatCommand() (*ChatCommand, error) {
	openaiParameterLayer, err := openai.NewClientParameterLayer()
	if err != nil {
		return nil, err
	}

	return &ChatCommand{
		CommandDescription: cmds.NewCommandDescription(
			"chat",
			cmds.WithShort("chat with the mechanical god in the clouds"),
			cmds.WithLayers(
				openaiParameterLayer,
			),
		),
	}, nil
}

func (c *ChatCommand) Run(
	ctx context2.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
) error {
	messages := []*context.Message{
		// different substrings of veryLongLoremIpsum
		{
			Text: veryLongLoremIpsum[:100],
			Role: "system",
		},
		{
			Text: veryLongLoremIpsum[100:300],
			Role: "assistant",
		},
		{
			Text: veryLongLoremIpsum[200:400],
			Role: "user",
		},
	}
	ctxtManager := context.NewManager(context.WithMessages(messages))

	p := tea.NewProgram(initialModel(ctxtManager))

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

var RootCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat with the mechanical god in the clouds",
}

func main() {
	chatCmd, err := NewChatCommand()
	cobra.CheckErr(err)

	chatCobraCommand, err := cli.BuildCobraCommandFromBareCommand(chatCmd)
	cobra.CheckErr(err)

	err = chatCobraCommand.Execute()
	cobra.CheckErr(err)
}
