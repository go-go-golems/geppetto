package codegen

import (
	context1 "context"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	context "github.com/go-go-golems/geppetto/pkg/context"
	steps "github.com/go-go-golems/geppetto/pkg/steps"
	ai "github.com/go-go-golems/geppetto/pkg/steps/ai"
	chat "github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	settings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	cmds "github.com/go-go-golems/glazed/pkg/cmds"
	parameters "github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"io"
)

const unitTestsCommandPrompt = "{{ define \"context\" -}}\n{{ .query | join \" \" }}\n{{ if .additional }}Additional instructions:\n{{ .additional | join \"\\n\" }}{{ end }}\n{{ if .concise }}\nGive a concise answer, answer in a single sentence if possible, skip unnecessary explanations.\n{{- end }}{{ if .use_bullets }}\nUse bullet points in the answer.\n{{- end }}{{ if .use_keywords }}\nUse keywords in the answer, not full sentences.\n{{- end }}\n{{- end }}\n\n{{ template \"context\" . }}\n\nCreate unit tests to test the given code.\n\n{{ if not .only_code }}\nAs an advanced AI assistant, you are here to guide me through the process of writing effective unit tests for my program.\nLet's begin by understanding the workings of my program, identifying potential edge cases, and considering important factors that could affect the functionality of my program.\n\nFirstly, could you provide a brief overview of your program's functionality? This will help us identify the key areas that need to be tested.\n\nSecondly, let's think about potential edge cases. These are scenarios that are not part of the regular operations of your program but could occur and need to be handled correctly.\n\nLastly, let's consider any important factors that could affect the functionality of your program. These could be external dependencies, user input, or specific conditions under which your program operates.\n\nRemember, the goal of unit testing is not just to find bugs, but to validate that each component of your program is working as expected under various conditions.\n\nBe exhaustive, think of all the edge cases.\nReturn a list of bullet points describing each test.\n{{- end }}\nHere is the code:\n\n```\n{{ .code }}\n```\n\n{{ if .framework }}Use {{ .framework }} framework for generating unit tests.{{end}}\n{{ if .table_driven }}Use table driven tests.{{end}}\n{{ if .with_signature }}\nAfter listing the unit tests, write the signature (not the test itself) of the function that would implement the tests.\n{{- end }}\n{{ if (or .with_implementation .only_code)  -}}\nPlease provide the implementation for each test.\n{{ if .with_comments }}\nWrite a short comment before each test describing the purpose of the test, if not obvious from the test name.\nDon't write obvious comments that just repeat the test name.\n{{- end }}\n{{- end }}\n\n{{ if .context}}Additional Context:\n{{ range .context }}\nPath: {{ .Path }}\n---\n{{ .Content }}\n---\n{{- end }}\n{{ end }}\n\n{{ if .bracket }}\n{{ template \"context\" . }}\n{{ end }}\n"
const unitTestsCommandSystemPrompt = "You are a meticulous and experienced software engineer with a deep understanding of testing and unit tests.\n{{ if .language }} You are an expert in {{ .language }} programming language.{{end}}\nYou are known for your ability to think of all possible edge cases and your attention to detail. You write clear and concise code.\n{{ .additional_system }}\n"

type UnitTestsCommand struct {
	*cmds.CommandDescription
	StepSettings settings.StepSettings   `yaml:"-"`
	Prompt       string                  `yaml:"prompt"`
	Messages     []*conversation.Message `yaml:"messages,omitempty"`
	SystemPrompt string                  `yaml:"system-prompt"`
}

type UnitTestsCommandParameters struct {
	Code               string                `glazed.parameter:"code"`
	Language           string                `glazed.parameter:"language"`
	WithSignature      bool                  `glazed.parameter:"with_signature"`
	WithImplementation bool                  `glazed.parameter:"with_implementation"`
	WithComments       bool                  `glazed.parameter:"with_comments"`
	OnlyCode           bool                  `glazed.parameter:"only_code"`
	Framework          string                `glazed.parameter:"framework"`
	TableDriven        bool                  `glazed.parameter:"table_driven"`
	AdditionalSystem   string                `glazed.parameter:"additional_system"`
	Additional         []string              `glazed.parameter:"additional"`
	Context            []parameters.FileData `glazed.parameter:"context"`
	Bracket            bool                  `glazed.parameter:"bracket"`
}

func (c *UnitTestsCommand) CreateStep(options ...chat.StepOption) (chat.Step, error) {
	stepFactory := &ai.StandardStepFactory{Settings: &c.StepSettings}
	return stepFactory.NewStep(options...)
}

func (c *UnitTestsCommand) CreateManager(params *UnitTestsCommandParameters) (*context.Manager, error) {
	return context.CreateManager(c.SystemPrompt, c.Prompt, c.Messages, params)
}

func (c *UnitTestsCommand) RunWithManager(ctx context1.Context, manager *context.Manager) (steps.StepResult[string], error) {
	// instantiate step from factory
	step, err := c.CreateStep()
	if err != nil {
		return nil, err
	}
	stepResult, err := step.Start(ctx, manager.GetMessages())
	if err != nil {
		return nil, err
	}
	return stepResult, nil
}

func (c *UnitTestsCommand) RunIntoWriter(ctx context1.Context, params *UnitTestsCommandParameters, w io.Writer) error {
	manager, err := c.CreateManager(params)
	if err != nil {
		return err
	}
	return context.RunIntoWriter(ctx, c, manager, w)
}

func (c *UnitTestsCommand) RunToString(ctx context1.Context, params *UnitTestsCommandParameters) (string, error) {
	manager, err := c.CreateManager(params)
	if err != nil {
		return "", err
	}
	return context.RunToString(ctx, c, manager)
}

func (c *UnitTestsCommand) RunToContextManager(ctx context1.Context, params *UnitTestsCommandParameters) (*context.Manager, error) {
	manager, err := c.CreateManager(params)
	if err != nil {
		return nil, err
	}
	return context.RunToContextManager(ctx, c, manager)
}

var _ context.GeppettoRunnable = (*UnitTestsCommand)(nil)

func NewUnitTestsCommand() (*UnitTestsCommand, error) {
	var flagDefs = []*parameters.ParameterDefinition{{
		Help: "Code to generate unit tests for",
		Name: "code",
		Type: "stringFromFiles",
	}, {
		Help: "Programming language of the code",
		Name: "language",
		Type: "string",
	}, {
		Help: "Whether to include signature in the output",
		Name: "with_signature",
		Type: "bool",
	}}

	var argDefs = []*parameters.ParameterDefinition{}
	cmdDescription := cmds.NewCommandDescription(
		"unit-tests",
		cmds.WithShort("Generate a list of unit tests for given code."),
		cmds.WithLong(""),
		cmds.WithFlags(flagDefs...),
		cmds.WithArguments(argDefs...))

	return &UnitTestsCommand{
		CommandDescription: cmdDescription,
		Prompt:             unitTestsCommandPrompt,
		SystemPrompt:       unitTestsCommandSystemPrompt,
	}, nil
}
