package cmds

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wesen/geppetto/pkg/steps"
	"github.com/wesen/geppetto/pkg/steps/openai"
	glazedcmds "github.com/wesen/glazed/pkg/cmds"
	"github.com/wesen/glazed/pkg/helpers"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
	"text/template"
)

type GeppettoCommandDescription struct {
	Name      string                  `yaml:"name"`
	Short     string                  `yaml:"short"`
	Long      string                  `yaml:"long,omitempty"`
	Flags     []*glazedcmds.Parameter `yaml:"flags,omitempty"`
	Arguments []*glazedcmds.Parameter `yaml:"arguments,omitempty"`

	Prompt string `yaml:"prompt"`
}

type GeppettoCommand struct {
	description *glazedcmds.CommandDescription
	Factories   map[string]interface{} `yaml:"__factories,omitempty"`
	Prompt      string
}

func (g *GeppettoCommand) RunFromCobra(cmd *cobra.Command, args []string) error {
	parameters, err := glazedcmds.GatherParameters(cmd, g.Description(), args)
	if err != nil {
		return err
	}

	printPrompt, _ := cmd.Flags().GetBool("print-prompt")
	parameters["print-prompt"] = printPrompt
	printDyno, _ := cmd.Flags().GetBool("print-dyno")
	parameters["print-dyno"] = printDyno

	for _, f := range g.Factories {
		factory, ok := f.(steps.GenericStepFactory)
		if !ok {
			continue
		}
		err = factory.UpdateFromCobra(cmd)
		if err != nil {
			return err
		}
	}

	return g.Run(parameters)
}

//go:embed templates/dyno.tmpl.html
var dynoTemplate string

func (g *GeppettoCommand) Run(parameters map[string]interface{}) error {
	openaiCompletionStepFactory_, ok := g.Factories["openai-completion-step"]
	if !ok {
		return errors.Errorf("No openai-completion-step factory defined")
	}
	openaiCompletionStepFactory, ok := openaiCompletionStepFactory_.(steps.StepFactory[string, string])
	if !ok {
		return errors.Errorf("openai-completion-step factory is not a StepFactory[string, string]")
	}

	// TODO(manuel, 2023-01-28) here we would overload the factory settings with stuff passed on the CLI
	// (say, temperature or model). This would probably be part of the API for the factory, in general the
	// factory is the central abstraction of the entire system
	s, err := openaiCompletionStepFactory.NewStep()
	if err != nil {
		return err
	}

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

	printPrompt, ok := parameters["print-prompt"]
	if ok && printPrompt.(bool) {
		fmt.Println(promptBuffer.String())
		return nil
	}

	printDyno, ok := parameters["print-dyno"]
	if ok && printDyno.(bool) {
		openaiCompletionStepFactory__, ok := openaiCompletionStepFactory_.(*openai.CompletionStepFactory)
		if !ok {
			return errors.Errorf("openai-completion-step factory is not a CompletionStepFactory")
		}
		settings := openaiCompletionStepFactory__.StepSettings

		dyno, err := helpers.RenderTemplateString(dynoTemplate, map[string]interface{}{
			"initialPrompt":   promptBuffer.String(),
			"initialResponse": "",
			"maxTokens":       settings.MaxResponseTokens,
			"temperature":     settings.Temperature,
			"topP":            settings.TopP,
		})
		if err != nil {
			return err
		}
		fmt.Println(dyno)
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

func (g *GeppettoCommand) Description() *glazedcmds.CommandDescription {
	return g.description
}

func (g *GeppettoCommand) BuildCobraCommand() (*cobra.Command, error) {
	cmd, err := glazedcmds.NewCobraCommand(g)
	if err != nil {
		return nil, err
	}
	cmd.Flags().Bool("print-prompt", false, "Print the prompt that will be executed.")
	cmd.Flags().Bool("print-dyno", false, "Print a dyno HTML embed with the given prompt. Useful to create documentation examples.")

	for _, f := range g.Factories {
		factory, ok := f.(steps.GenericStepFactory)
		if !ok {
			continue
		}

		err := factory.AddFlags(cmd, "openai-", &openai.CompletionStepFactoryFlagsDefaults{})
		if err != nil {
			return nil, err
		}
	}
	return cmd, nil
}

type GeppettoCommandLoader struct {
}

func (g *GeppettoCommandLoader) LoadCommandFromYAML(s io.Reader) ([]glazedcmds.Command, error) {
	yamlContent, err := io.ReadAll(s)
	if err != nil {
		return nil, err
	}

	buf := strings.NewReader(string(yamlContent))
	scd := &GeppettoCommandDescription{
		Flags:     []*glazedcmds.Parameter{},
		Arguments: []*glazedcmds.Parameter{},
	}
	err = yaml.NewDecoder(buf).Decode(scd)
	if err != nil {
		return nil, err
	}

	// TODO(manuel, 2023-01-27): There has to be a better way to parse YAML factories
	// maybe the easiest is just going to be to make them a separate file in the bundle format, really
	// rewind to read the factories...
	buf = strings.NewReader(string(yamlContent))
	completionStepFactory, err := openai.NewCompletionStepFactoryFromYAML(buf)

	if err != nil {
		return nil, err
	}

	factories := map[string]interface{}{}
	if completionStepFactory != nil {
		factories["openai-completion-step"] = completionStepFactory
	}
	sq := &GeppettoCommand{
		Prompt: scd.Prompt,
		// separate copy because the glazed framework uses this to build the cobra command and mutates it
		description: &glazedcmds.CommandDescription{
			Name:      scd.Name,
			Short:     scd.Short,
			Long:      scd.Long,
			Flags:     scd.Flags,
			Arguments: scd.Arguments,
		},
		Factories: factories,
	}

	return []glazedcmds.Command{sq}, nil
}

func (g *GeppettoCommandLoader) LoadCommandAliasFromYAML(s io.Reader) ([]*glazedcmds.CommandAlias, error) {
	var alias glazedcmds.CommandAlias
	err := yaml.NewDecoder(s).Decode(&alias)
	if err != nil {
		return nil, err
	}

	if !alias.IsValid() {
		return nil, errors.New("Invalid command alias")
	}

	return []*glazedcmds.CommandAlias{&alias}, nil
}
