package main

import (
	"bytes"
	context2 "context"
	_ "embed"
	tea "github.com/charmbracelet/bubbletea"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/openai/chat"
	"github.com/go-go-golems/geppetto/pkg/ui"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	Step *chat.Step
}

func NewChatCommand() (*ChatCommand, error) {
	buf := bytes.NewReader(defaultChatSettings)
	step, err := chat.NewStepFromYAML(buf)
	if err != nil {
		return nil, err
	}

	openaiAPIKey := viper.GetString("openai-api-key")
	if openaiAPIKey != "" {
		step.ClientSettings.APIKey = &openaiAPIKey
	}
	chatParameterLayer, err := chat.NewParameterLayer(
		layers.WithDefaults(step.StepSettings),
	)
	if err != nil {
		return nil, err
	}

	openaiParameterLayer, err := openai.NewClientParameterLayer(
		layers.WithDefaults(step.ClientSettings),
	)
	if err != nil {
		return nil, err
	}

	return &ChatCommand{
		CommandDescription: cmds.NewCommandDescription(
			"chat",
			cmds.WithShort("chat with the mechanical god in the clouds"),
			cmds.WithLayers(
				openaiParameterLayer,
				chatParameterLayer,
			),
		),
		Step: step,
	}, nil
}

//go:embed default-chat-settings.yaml
var defaultChatSettings []byte

func (c *ChatCommand) Run(
	ctx context2.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
) error {
	messages := []*context.Message{
		// different substrings of veryLongLoremIpsum
		{
			Text: "You are a chatbot.",
			Role: "system",
		},
		//{
		//	Text: veryLongLoremIpsum[:200],
		//	Role: "assistant",
		//},
		//{
		//	Text: veryLongLoremIpsum[:400],
		//	Role: "user",
		//},
	}
	ctxtManager := context.NewManager(context.WithMessages(messages))

	err := c.Step.UpdateFromParameters(ps)
	if err != nil {
		return err
	}

	p := tea.NewProgram(
		ui.InitialModel(ctxtManager, c.Step),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

var rootCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat with the mechanical god in the clouds",
}

func main() {
	// stick with pinocchio to load the api key
	err := clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	chatCmd, err := NewChatCommand()
	cobra.CheckErr(err)

	chatCobraCommand, err := cli.BuildCobraCommandFromBareCommand(chatCmd)
	cobra.CheckErr(err)

	err = chatCobraCommand.Execute()
	cobra.CheckErr(err)

}
