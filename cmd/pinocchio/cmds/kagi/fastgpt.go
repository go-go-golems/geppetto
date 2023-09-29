package kagi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/glamour"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"io"
	"net/http"
)

type FastGPTCommand struct {
	*cmds.CommandDescription
}

type FastGPTResponse struct {
	Meta struct {
		ID   string `json:"id"`
		Node string `json:"node"`
		MS   int    `json:"ms"`
	} `json:"meta"`
	Data FastGPTAnswer `json:"data"`
}

type FastGPTAnswer struct {
	Output     string      `json:"output"`
	Tokens     int         `json:"tokens"`
	References []Reference `json:"references"`
}

type Reference struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

type FastGPTRequest struct {
	Query     string `json:"query"`
	Cache     bool   `json:"cache"`
	WebSearch bool   `json:"web_search"`
}

func RenderFastGPTAnswers(answer FastGPTAnswer, query string) (string, error) {
	// Define a Go template for the markdown representation
	const mdTemplate = `
# Query: {{ .Query }}

{{- with .Answer }}
## Answer:

{{ .Output }}
**Tokens Used:** {{ .Tokens }}

### References:
{{- range $idx, $value := .References }}
{{ with $value -}}
[{{ add $idx 1 }}]: {{ .Title }}
- **URL:** [{{ .URL }}]({{ .URL }})
- **Snippet:** {{ .Snippet }}
{{ end }}
{{ end }}
{{- end }}
`

	// Parse and execute the template
	tmpl, err := templating.CreateTemplate("markdown").Parse(mdTemplate)
	if err != nil {
		return "", err
	}

	data := struct {
		Query  string
		Answer FastGPTAnswer
	}{
		Query:  query,
		Answer: answer,
	}

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, data)
	if err != nil {
		return "", err
	}

	// Convert the generated markdown into a styled string using glamour
	// Assuming you have the glamour library imported and set up properly
	styled, err := glamour.Render(buffer.String(), "dark")
	if err != nil {
		return "", err
	}

	return styled, nil
}

func NewFastGPTCommand() (*FastGPTCommand, error) {
	return &FastGPTCommand{
		CommandDescription: cmds.NewCommandDescription(
			"fastgpt",
			cmds.WithShort("Answer a query using FastGPT"),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"query",
					parameters.ParameterTypeString,
					parameters.WithHelp("A query to be answered"),
					parameters.WithRequired(true),
				),
				parameters.NewParameterDefinition(
					"web_search",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Whether to use web search"),
					parameters.WithDefault(true),
				),
			),
			cmds.WithArguments(
				parameters.NewParameterDefinition(
					"cache",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Whether to allow cached requests & responses"),
					parameters.WithDefault(true),
				),
			),
		),
	}, nil
}

func (c *FastGPTCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	w io.Writer,
) error {
	token := viper.GetString("kagi-api-key")
	if token == "" {
		return errors.New("no API token provided")
	}

	var reqData FastGPTRequest
	if query, ok := ps["query"]; ok {
		reqData.Query = query.(string)
	}
	if cache, ok := ps["cache"]; ok {
		reqData.Cache = cache.(bool)
	}
	if webSearch, ok := ps["web_search"]; ok {
		reqData.WebSearch = webSearch.(bool)
	}

	bodyData, err := json.Marshal(reqData)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request body")
	}

	req, err := http.NewRequest("POST", "https://kagi.com/api/v0/fastgpt", bytes.NewBuffer(bodyData))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("non-200 response received: " + string(respBody))
	}

	var response FastGPTResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return errors.Wrap(err, "failed to parse response body")
	}

	styled, err := RenderFastGPTAnswers(response.Data, reqData.Query)
	if err != nil {
		return err
	}

	fmt.Println(styled)
	return &cmds.ExitWithoutGlazeError{}
}
