package get_conversation

import (
	"context"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"strings"
)

type GetConversationCommand struct {
	*cmds.CommandDescription
}

// TODO add flag for only exporting the assistant responses
// TODO add flag for exporting the source blocks
// TODO add flag for adding the messages as comments in the source blocks (if we can detect their type, for example)

func NewGetConversationCommand() (*GetConversationCommand, error) {
	return &GetConversationCommand{
		CommandDescription: cmds.NewCommandDescription(
			"render",
			cmds.WithShort("Converts GPT HTML to markdown"),
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
			),
		),
	}, nil

}

func (cmd *GetConversationCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	w io.Writer,
) error {
	// Extracting arguments and flags
	urls := ps["urls"].([]string)
	concise := ps["concise"].(bool)
	withMetadata := ps["with-metadata"].(bool)
	outputJson := ps["output-json"].(bool)

	renameRoles, ok := cast.CastInterfaceToStringMap[string, interface{}](ps["rename-roles"])
	if !ok {
		return errors.New("Failed to cast rename-roles to map[string]string")
	}

	if len(urls) == 0 {
		return errors.New("No URLs provided")
	}

	outputAsArray := ps["output-as-array"].(bool)
	onlyConversations := ps["only-conversations"].(bool)
	fullJson := ps["full-json"].(bool)
	outputJsons := make([]interface{}, len(urls))

	for _, url := range urls {
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

		if outputJson {
			if fullJson {
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

			if onlyConversations {
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

		renderer := &Renderer{
			RenameRoles:  renameRoles,
			Concise:      concise,
			WithMetadata: withMetadata,
		}

		linearConversation := data.Props.PageProps.ServerResponse.LinearConversation

		renderer.PrintConversation(url, data.Props.PageProps.ServerResponse.ServerResponseData, linearConversation)
	}

	if outputJson {

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if len(outputJsons) == 1 && !outputAsArray {
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
