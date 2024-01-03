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
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"net/url"
	"text/template"
)

type EnrichWebCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &EnrichWebCommand{}

type SearchObject struct {
	T         int    `json:"t"`
	Rank      int    `json:"rank"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	Published string `json:"published"`
}

type EnrichWebResponse struct {
	Meta struct {
		ID   string `json:"id"`
		Node string `json:"node"`
		MS   int    `json:"ms"`
	} `json:"meta"`
	Data []SearchObject `json:"data"`
}

func RenderMarkdown(searchObjects []SearchObject) (string, error) {
	// Define a Go template for the markdown representation
	const mdTemplate = `
{{- range . }}
## {{.Rank}}. {{ .Title }}

- **URL:** [{{ .URL }}]({{ .URL }})  
- **Published:** {{ .Published }}  

{{ .Snippet }}

{{ end }}
`

	// Parse and execute the template
	tmpl, err := template.New("markdown").Parse(mdTemplate)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, searchObjects)
	if err != nil {
		return "", err
	}

	// Convert the generated markdown into a styled string using glamour
	styled, err := glamour.Render(buffer.String(), "dark")
	if err != nil {
		return "", err
	}

	return styled, nil
}

type EnrichWebSettings struct {
	Query    string `glazed.parameter:"query"`
	Token    string `glazed.parameter:"token"`
	Markdown bool   `glazed.parameter:"markdown"`
	Limit    int    `glazed.parameter:"limit"`
	News     bool   `glazed.parameter:"news"`
}

func NewEnrichWebCommand() (*EnrichWebCommand, error) {
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, errors.Wrap(err, "could not create Glazed parameter layer")
	}

	return &EnrichWebCommand{
		CommandDescription: cmds.NewCommandDescription(
			"enrich",
			cmds.WithShort("Fetch enrichment results for web content"),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"query",
					parameters.ParameterTypeString,
					parameters.WithHelp("Search query"),
					parameters.WithRequired(true),
				),
				parameters.NewParameterDefinition(
					"token",
					parameters.ParameterTypeString,
					parameters.WithHelp("API Token"),
				),
				parameters.NewParameterDefinition(
					"markdown",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Render output as markdown"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"limit",
					parameters.ParameterTypeInteger,
					parameters.WithHelp("Limit number of results"),
					parameters.WithDefault(10),
				),
				parameters.NewParameterDefinition(
					"news",
					parameters.ParameterTypeBool,
					parameters.WithHelp("Search news"),
					parameters.WithDefault(false),
				),
			),
			cmds.WithLayersList(
				glazedParameterLayer,
			),
		),
	}, nil
}

func (c *EnrichWebCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	s := &EnrichWebSettings{}
	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
	if err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}

	query := s.Query
	token := viper.GetString("kagi-api-key")
	if s.Token != "" {
		token = s.Token
	}
	if token == "" {
		return errors.New("no API token provided")
	}

	news := s.News

	url_ := fmt.Sprintf("https://kagi.com/api/v0/enrich/web?q=%s", url.QueryEscape(query))
	if news {
		url_ = fmt.Sprintf("https://kagi.com/api/v0/enrich/news?q=%s", url.QueryEscape(query))
	}
	req, err := http.NewRequest("GET", url_, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", "Bot "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("non-200 response received: " + string(body))
	}

	var response EnrichWebResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return errors.Wrap(err, "failed to parse response body")
	}

	limit := s.Limit
	if limit > len(response.Data) {
		limit = len(response.Data)
	}
	response.Data = response.Data[:limit]

	markdown := s.Markdown
	if markdown {
		styled, err := RenderMarkdown(response.Data)
		if err != nil {
			return err
		}

		fmt.Println(styled)
		return &cmds.ExitWithoutGlazeError{}
	}

	for _, obj := range response.Data {
		row := types.NewRow(
			types.MRP("rank", obj.Rank),
			types.MRP("url", obj.URL),
			types.MRP("title", obj.Title),
			types.MRP("snippet", obj.Snippet),
			types.MRP("published", obj.Published),
		)

		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}

	return nil
}
