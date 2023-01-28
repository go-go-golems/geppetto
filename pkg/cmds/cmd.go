package cmds

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wesen/geppetto/pkg/steps"
	cmds2 "github.com/wesen/glazed/pkg/cmds"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
	"text/template"
)

type GeppettoCommandDescription struct {
	Name      string             `yaml:"name"`
	Short     string             `yaml:"short"`
	Long      string             `yaml:"long,omitempty"`
	Flags     []*cmds2.Parameter `yaml:"flags,omitempty"`
	Arguments []*cmds2.Parameter `yaml:"arguments,omitempty"`

	Prompt string `yaml:"prompt"`
}

type GeppettoCommand struct {
	description *cmds2.CommandDescription
	Prompt      string
}

func (g *GeppettoCommand) RunFromCobra(cmd *cobra.Command, args []string) error {
	parameters, err := cmds2.GatherParameters(cmd, g.Description(), args)
	if err != nil {
		return err
	}

	printPrompt, _ := cmd.Flags().GetBool("print-prompt")
	parameters["print-prompt"] = printPrompt

	return g.Run(parameters)
}

func (g *GeppettoCommand) Run(parameters map[string]interface{}) error {
	apiKey := viper.GetString("openai-api-key")
	s := steps.NewOpenAICompletionStep(apiKey)
	ctx := context.Background()

	promptTemplate, err := template.New("prompt").Parse(g.Prompt)
	if err != nil {
		return err
	}

	var promptBuffer strings.Builder
	err = promptTemplate.Execute(&promptBuffer, parameters)
	if err != nil {
		return err
	}

	if parameters["print-prompt"].(bool) {
		fmt.Println(promptBuffer.String())
		return nil
	}

	eg, ctx2 := errgroup.WithContext(ctx)
	prompt := promptBuffer.String()
	//fmt.Printf("Prompt:\n\n%s\n\n", prompt)

	err = s.Start(ctx2, prompt)
	if err != nil {
		return err
	}

	eg.Go(func() error {
		result := <-s.GetOutput()
		v, err := result.Value()
		if err != nil {
			return err
		}

		fmt.Printf("%s", v)
		return err
	})
	return eg.Wait()
}

func (g *GeppettoCommand) Description() *cmds2.CommandDescription {
	return g.description
}

func (g *GeppettoCommand) BuildCobraCommand() (*cobra.Command, error) {
	cmd, err := cmds2.NewCobraCommand(g)
	if err != nil {
		return nil, err
	}
	cmd.Flags().Bool("print-prompt", false, "Print the prompt that will be executed")
	return cmd, nil
}

type GeppettoCommandLoader struct {
}

func (g *GeppettoCommandLoader) LoadCommandFromYAML(s io.Reader) ([]cmds2.Command, error) {
	scd := &GeppettoCommandDescription{
		Flags:     []*cmds2.Parameter{},
		Arguments: []*cmds2.Parameter{},
	}
	err := yaml.NewDecoder(s).Decode(scd)
	if err != nil {
		return nil, err
	}

	sq := &GeppettoCommand{
		Prompt: scd.Prompt,
		description: &cmds2.CommandDescription{
			Name:      scd.Name,
			Short:     scd.Short,
			Long:      scd.Long,
			Flags:     scd.Flags,
			Arguments: scd.Arguments,
		},
	}

	return []cmds2.Command{sq}, nil
}

func (g *GeppettoCommandLoader) LoadCommandAliasFromYAML(s io.Reader) ([]*cmds2.CommandAlias, error) {
	var alias cmds2.CommandAlias
	err := yaml.NewDecoder(s).Decode(&alias)
	if err != nil {
		return nil, err
	}

	if !alias.IsValid() {
		return nil, errors.New("Invalid command alias")
	}

	return []*cmds2.CommandAlias{&alias}, nil
}
