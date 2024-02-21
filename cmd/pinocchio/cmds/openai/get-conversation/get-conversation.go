package get_conversation

import (
	"context"
	_ "embed"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/markdown"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"strings"
)

type GetConversationCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = &GetConversationCommand{}

//go:embed "doc.md"
var doc string

func NewGetConversationCommand() (*GetConversationCommand, error) {
	return &GetConversationCommand{
		CommandDescription: cmds.NewCommandDescription(
			"get-conversation",
			cmds.WithShort("Converts GPT HTML to markdown"),
			cmds.WithLong(doc),
			cmds.WithArguments(
				parameters.NewParameterDefinition(
					"urls",
					parameters.ParameterTypeStringList,
					parameters.WithHelp("Path to HTML files or URLs"),
					parameters.WithRequired(true),
				),
			),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"concise",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Concise output"),
					parameters.WithDefault(true),
				),
				parameters.NewParameterDefinition(
					"with-metadata",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Include metadata"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"rename-roles",
					parameters.ParameterTypeKeyValue,
					parameters.WithHelp("Rename roles"),
					parameters.WithDefault(map[string]string{
						"user":      "john",
						"assistant": "claire",
						"system":    "george",
					}),
				),
				parameters.NewParameterDefinition(
					"output-json",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Output JSON"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"output-as-array",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Output as array"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"full-json",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Output full JSON"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"only-conversations",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Only conversations in JSON output"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"only-assistant",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Only assistant responses in markdown output"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"only-source-blocks",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Only source blocks in markdown output"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"merge-source-blocks",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Merge source blocks in markdown output"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"inline-conversations",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Inline conversation as comments in source blocks"),
					parameters.WithDefault(true),
				),
			),
		),
	}, nil
}

type GetConversationSettings struct {
	URLs                []string          `glazed.parameter:"urls"`
	Concise             bool              `glazed.parameter:"concise"`
	WithMetadata        bool              `glazed.parameter:"with-metadata"`
	RenameRoles         map[string]string `glazed.parameter:"rename-roles"`
	OutputJson          bool              `glazed.parameter:"output-json"`
	OutputAsArray       bool              `glazed.parameter:"output-as-array"`
	FullJson            bool              `glazed.parameter:"full-json"`
	OnlyConversations   bool              `glazed.parameter:"only-conversations"`
	OnlyAssistant       bool              `glazed.parameter:"only-assistant"`
	OnlySourceBlocks    bool              `glazed.parameter:"only-source-blocks"`
	MergeSourceBlocks   bool              `glazed.parameter:"merge-source-blocks"`
	InlineConversations bool              `glazed.parameter:"inline-conversations"`
}

func (cmd *GetConversationCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	w io.Writer,
) error {
	s := &GetConversationSettings{}
	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
	if err != nil {
		return err
	}

	// Extracting arguments and flags

	if len(s.URLs) == 0 {
		return errors.New("No URLs provided")
	}

	outputJsons := make([]interface{}, len(s.URLs))

	for _, url := range s.URLs {
		var htmlContent []byte
		var err error

		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			htmlContent_, err := getContent(url)
			if err != nil {
				return err
			}
			htmlContent = htmlContent_
		} else {
			htmlContent, err = os.ReadFile(url)
			if err != nil {
				return err
			}
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
		if err != nil {
			return err
		}

		scriptContent := doc.Find("#__NEXT_DATA__").Text()

		if s.OutputJson {
			if s.FullJson {
				data := map[string]interface{}{}
				err = json.Unmarshal([]byte(scriptContent), &data)
				if err != nil {
					return err
				}

				outputJsons = append(outputJsons, data)
				continue
			}

			var data NextData
			err = json.Unmarshal([]byte(scriptContent), &data)
			if err != nil {
				return err
			}

			if s.OnlyConversations {
				outputJsons = append(outputJsons, data.Props.PageProps.ServerResponse.ServerResponseData.LinearConversation)
				continue
			}

			outputJsons = append(outputJsons, data.Props.PageProps.ServerResponse.ServerResponseData)
		}

		var data NextData
		err = json.Unmarshal([]byte(scriptContent), &data)
		if err != nil {
			return err
		}

		onlySourceBlocks := s.OnlySourceBlocks || s.MergeSourceBlocks

		linearConversation := data.Props.PageProps.ServerResponse.LinearConversation

		if onlySourceBlocks {
			merged := ""
			for _, conversation := range linearConversation {
				if len(conversation.Message.Content.Parts) == 0 {
					continue
				}
				content := strings.Join(conversation.Message.Content.Parts, "\n")
				if s.MergeSourceBlocks {
					var blocks []string
					if s.InlineConversations {
						blocks = markdown.ExtractCodeBlocksWithComments(content, false)
					} else {
						blocks = markdown.ExtractQuotedBlocks(content, false)
					}
					if len(blocks) == 0 {
						continue
					}
					merged += strings.Join(blocks, "\n\n") + "\n\n"
					continue
				}

				var blocks []string
				if s.InlineConversations {
					blocks = markdown.ExtractCodeBlocksWithComments(content, true)
				} else {
					blocks = markdown.ExtractQuotedBlocks(content, true)
				}
				if len(blocks) == 0 {
					continue
				}
				foo := strings.Join(blocks, "\n\n") + "\n\n---\n\n"
				_, err := w.Write([]byte(foo))
				if err != nil {
					return err
				}
			}

			if s.MergeSourceBlocks {
				_, err := w.Write([]byte(merged))
				if err != nil {
					return err
				}
			}

			continue
		}

		if s.OnlyAssistant {
			conversations := []Conversation{}
			for _, conversation := range linearConversation {
				if conversation.Message.Author.Role != "assistant" {
					continue
				}
				// skip author role in the output, now that it's all just the ai
				conversation.Message.Author.Role = ""
				conversations = append(conversations, conversation)
			}
			linearConversation = conversations
		}

		renderer := &Renderer{
			RenameRoles:  s.RenameRoles,
			Concise:      s.Concise,
			WithMetadata: s.WithMetadata,
		}

		renderer.PrintConversation(url, data.Props.PageProps.ServerResponse.ServerResponseData, linearConversation)
	}

	if s.OutputJson {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if len(outputJsons) == 1 && !s.OutputAsArray {
			err := encoder.Encode(outputJsons[0])
			if err != nil {
				return err
			}
		} else {
			err := encoder.Encode(outputJsons)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getContent(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	htmlContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return htmlContent, nil
}
