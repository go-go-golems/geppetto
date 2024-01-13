package codegen

import (
	"github.com/dave/jennifer/jen"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/codegen"
	"github.com/iancoleman/strcase"
	"strconv"
)

const TemplatingPath = "github.com/go-go-golems/glazed/pkg/helpers/templating"
const ChatPath = "github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
const AiPath = "github.com/go-go-golems/geppetto/pkg/steps/ai"
const SettingsPath = "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
const StepsPath = "github.com/go-go-golems/geppetto/pkg/steps"
const CommandPath = "github.com/go-go-golems/geppetto/pkg/cmds"
const ContextPath = "github.com/go-go-golems/geppetto/pkg/context"
const ConversationPath = "github.com/go-go-golems/bobatea/pkg/chat/conversation"
const LayerPath = "github.com/go-go-golems/glazed/pkg/cmds/layers"

type GeppettoCommandCodeGenerator struct {
	PackageName string
}

func (g *GeppettoCommandCodeGenerator) defineConstants(f *jen.File, cmdName string, cmd *cmds.GeppettoCommand) {
	// Define the constants for prompts and messages.
	promptConstName := strcase.ToLowerCamel(cmdName) + "CommandPrompt"
	f.Const().Id(promptConstName).Op("=").Lit(cmd.Prompt)

	systemPromptConstName := strcase.ToLowerCamel(cmdName) + "CommandSystemPrompt"
	f.Const().Id(systemPromptConstName).Op("=").Lit(cmd.SystemPrompt)

	for i, message := range cmd.Messages {
		messageConstName := strcase.ToLowerCamel(cmdName) + "CommandMessage" + strcase.ToCamel(strconv.Itoa(i))
		// TODO(manuel, 2024-01-13) Handle other message types, this is a shortcut
		f.Const().Id(messageConstName).Op("=").Lit(message.Content.String())
	}
}

func (g *GeppettoCommandCodeGenerator) defineStruct(f *jen.File, cmdName string) {
	structName := strcase.ToCamel(cmdName) + "Command"
	f.Type().Id(structName).Struct(
		jen.Op("*").Qual(codegen.GlazedCommandsPath, "CommandDescription"),
		jen.Id("StepSettings").Qual(SettingsPath, "StepSettings").Tag(map[string]string{"yaml": "-"}),
		jen.Id("Prompt").String().Tag(map[string]string{"yaml": "prompt"}),
		jen.Id("Messages").Qual(ConversationPath, "Conversation").
			Tag(map[string]string{"yaml": "messages,omitempty"}),
		jen.Id("SystemPrompt").String().Tag(map[string]string{"yaml": "system-prompt"}),
	)
}

func (g *GeppettoCommandCodeGenerator) defineParametersStruct(
	f *jen.File,
	cmdName string,
	cmd *cmds.GeppettoCommand,
) {
	structName := strcase.ToCamel(cmdName) + "CommandParameters"
	f.Type().Id(structName).StructFunc(func(g *jen.Group) {
		cmd.GetDefaultFlags().ForEach(func(flag *parameters.ParameterDefinition) {
			s := g.Id(strcase.ToCamel(flag.Name))
			s = codegen.FlagTypeToGoType(s, flag.Type)
			s.Tag(map[string]string{"glazed.parameter": strcase.ToSnake(flag.Name)})
		})
		cmd.GetDefaultArguments().ForEach(func(arg *parameters.ParameterDefinition) {
			s := g.Id(strcase.ToCamel(arg.Name))
			s = codegen.FlagTypeToGoType(s, arg.Type)
			s.Tag(map[string]string{"glazed.argument": strcase.ToSnake(arg.Name)})
		})
	})
}

func (g *GeppettoCommandCodeGenerator) defineNewFunction(
	f *jen.File,
	cmdName string,
	cmd *cmds.GeppettoCommand,
) error {
	commandName := strcase.ToCamel(cmdName)
	lowerCommandName := strcase.ToLowerCamel(cmdName)

	funcName := "New" + commandName + "Command"
	commandStruct := commandName + "Command"
	promptConstName := lowerCommandName + "CommandPrompt"
	systemPromptConstName := lowerCommandName + "CommandSystemPrompt"

	description := cmd.Description()

	f.Var().Id("_").
		Qual(ContextPath, "GeppettoRunnable").
		Op("=").
		Parens(jen.Op("*").Id(commandStruct)).Parens(jen.Nil())

	var err_ error
	f.Func().Id(funcName).Params().
		Params(jen.Op("*").Id(commandStruct), jen.Error()).
		Block(
			// TODO(manuel, 2023-12-07) Can be refactored since this is duplicated in geppetto/codegen.go
			jen.Var().Id("flagDefs").Op("=").
				Index().Op("*").
				Qual(codegen.GlazedParametersPath, "ParameterDefinition").
				ValuesFunc(func(g *jen.Group) {
					err_ = cmd.GetDefaultFlags().ForEachE(func(flag *parameters.ParameterDefinition) error {
						dict, err := codegen.ParameterDefinitionToDict(flag)
						if err != nil {
							return err
						}
						g.Values(dict)
						return nil
					})
				}),
			jen.Line(),
			jen.Var().Id("argDefs").Op("=").
				Index().Op("*").
				Qual(codegen.GlazedParametersPath, "ParameterDefinition").
				ValuesFunc(func(g *jen.Group) {
					err_ = cmd.GetDefaultArguments().ForEachE(func(arg *parameters.ParameterDefinition) error {
						dict, err := codegen.ParameterDefinitionToDict(arg)
						if err != nil {
							return err
						}
						g.Values(dict)
						return nil
					})
				}),
			jen.Id("cmdDescription").
				Op(":=").
				Qual(codegen.GlazedCommandsPath, "NewCommandDescription").
				Call(
					jen.Line().Lit(description.Name),
					jen.Line().Qual(codegen.GlazedCommandsPath, "WithShort").
						Call(jen.Lit(description.Short)),
					jen.Line().Qual(codegen.GlazedCommandsPath, "WithLong").
						Call(jen.Lit(description.Long)),
					jen.Line().Qual(codegen.GlazedCommandsPath, "WithFlags").
						Call(jen.Id("flagDefs").Op("...")),
					jen.Line().Qual(codegen.GlazedCommandsPath, "WithArguments").
						Call(jen.Id("argDefs").Op("...")),
				),
			jen.Line(),
			jen.Return(jen.Op("&").Id(commandStruct).Values(jen.Dict{
				jen.Id("CommandDescription"): jen.Id("cmdDescription"),
				jen.Id("Prompt"):             jen.Id(promptConstName),
				jen.Id("SystemPrompt"):       jen.Id(systemPromptConstName),
			}), jen.Nil()),
		)

	return err_
}

func (g *GeppettoCommandCodeGenerator) defineRunMethods(f *jen.File, cmdName string) {
	cmdName = strcase.ToCamel(cmdName) + "Command"
	// CreateStep method
	f.Func().
		Params(jen.Id("c").Op("*").Id(cmdName)).
		Id("CreateStep").
		Params(jen.Id("options").Op("...").Qual(ChatPath, "StepOption")).
		Parens(jen.List(jen.Qual(ChatPath, "Step"), jen.Error())).
		Block(
			jen.Id("stepFactory").Op(":=").Op("&").Qual(AiPath, "StandardStepFactory").Values(jen.Dict{
				jen.Id("Settings"): jen.Op("&").Id("c").Dot("StepSettings"),
			}),
			jen.Return(jen.Id("stepFactory").Dot("NewStep").Call(jen.Id("options").Op("..."))),
		).Line()

	f.Func().
		Params(jen.Id("c").Op("*").Id(cmdName)).
		Id("CreateManager").
		Params(
			jen.Id("params").Op("*").Id(cmdName+"Parameters"),
		).
		Params(jen.Qual(ConversationPath, "Manager"), jen.Error()).
		Block(
			jen.Return(
				jen.Qual(ConversationPath, "CreateManager").Call(
					jen.Id("c").Dot("SystemPrompt"),
					jen.Id("c").Dot("Prompt"),
					jen.Id("c").Dot("Messages"),
					jen.Id("params"),
				),
			),
		).Line()

	// RunWithManager method
	f.Func().
		Params(jen.Id("c").Op("*").Id(cmdName)).Id("RunWithManager").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("manager").Qual(ConversationPath, "Manager"),
		).
		Params(jen.Qual(StepsPath, "StepResult").Index(jen.String()), jen.Error()).
		Block(
			jen.Comment("instantiate step from factory"),
			jen.List(jen.Id("step"), jen.Err()).
				Op(":=").
				Id("c").Dot("CreateStep").Call(),
			jen.If().Err().Op("!=").Nil().Block(
				jen.Return(jen.Nil(), jen.Err()),
			),
			jen.List(jen.Id("stepResult"), jen.Err()).Op(":=").Id("step").Dot("Start").
				Call(jen.Id("ctx"), jen.Id("manager").Dot("GetConversation").Call()),
			jen.If().Err().Op("!=").Nil().Block(
				jen.Return(jen.Nil(), jen.Err()),
			),
			jen.Return(jen.Id("stepResult"), jen.Nil()),
		).Line()

	// RunIntoWriter method
	f.Func().Params(jen.Id("c").Op("*").Id(cmdName)).Id("RunIntoWriter").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("params").Op("*").Id(cmdName+"Parameters"),
			jen.Id("w").Qual("io", "Writer"),
		).
		Error().
		Block(
			jen.List(
				jen.Id("manager"), jen.Err()).
				Op(":=").
				Id("c").Dot("CreateManager").Call(jen.Id("params")),
			jen.If().Err().Op("!=").Nil().Block(
				jen.Return(jen.Err()),
			),
			jen.Return(jen.Qual(ContextPath, "RunIntoWriter").
				Call(
					jen.Id("ctx"),
					jen.Id("c"),
					jen.Id("manager"),
					jen.Id("w"),
				)),
		).Line()

	// RunToString method
	f.Func().Params(jen.Id("c").Op("*").Id(cmdName)).Id("RunToString").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("params").Op("*").Id(cmdName+"Parameters"),
		).
		Params(jen.String(), jen.Error()).
		Block(
			jen.List(
				jen.Id("manager"), jen.Err()).
				Op(":=").
				Id("c").Dot("CreateManager").Call(jen.Id("params")),
			jen.If().Err().Op("!=").Nil().Block(
				jen.Return(jen.Lit(""), jen.Err()),
			),
			jen.Return(
				jen.Qual(ContextPath, "RunToString").
					Call(jen.Id("ctx"), jen.Id("c"), jen.Id("manager"))),
		).Line()

	// RunToContextManager method
	f.Func().Params(jen.Id("c").Op("*").Id(cmdName)).Id("RunToContextManager").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("params").Op("*").Id(cmdName+"Parameters"),
		).
		Params(
			jen.Qual(ConversationPath, "Manager"),
			jen.Error()).
		Block(
			jen.List(
				jen.Id("manager"), jen.Err()).
				Op(":=").
				Id("c").Dot("CreateManager").Call(jen.Id("params")),
			jen.If().Err().Op("!=").Nil().Block(
				jen.Return(jen.Nil(), jen.Err()),
			),
			jen.Return(
				jen.Qual(ContextPath, "RunToContextManager").
					Call(jen.Id("ctx"), jen.Id("c"), jen.Id("manager"))),
		)
}

func (g *GeppettoCommandCodeGenerator) GenerateCommandCode(cmd *cmds.GeppettoCommand) (*jen.File, error) {
	f := jen.NewFile(g.PackageName)
	cmdName := strcase.ToLowerCamel(cmd.Name)

	// Define constants, struct, and methods using helper functions.
	g.defineConstants(f, cmdName, cmd)
	g.defineStruct(f, cmdName)

	f.Line()
	g.defineParametersStruct(f, cmdName, cmd)
	g.defineRunMethods(f, cmdName)
	f.Line()
	err := g.defineNewFunction(f, cmdName, cmd)
	if err != nil {
		return nil, err
	}

	return f, nil
}
