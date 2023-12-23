package kagi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"io"
	"net/http"
)

type SummarizeCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = &SummarizeCommand{}

type SummarizationResponse struct {
	Meta struct {
		ID   string `json:"id"`
		Node string `json:"node"`
		MS   int    `json:"ms"`
	} `json:"meta"`
	Data struct {
		Output string `json:"output"`
		Tokens int    `json:"tokens"`
	} `json:"data"`
}

type SummarizationRequest struct {
	URL            string `json:"url,omitempty"`
	Text           string `json:"text,omitempty"`
	Engine         string `json:"engine,omitempty"`
	SummaryType    string `json:"summary_type,omitempty"`
	TargetLanguage string `json:"target_language,omitempty"`
	Cache          bool   `json:"cache"`
}

type SummarizeSettings struct {
	URL            string `glazed.parameter:"url"`
	Text           string `glazed.parameter:"text"`
	Engine         string `glazed.parameter:"engine"`
	SummaryType    string `glazed.parameter:"summary_type"`
	TargetLanguage string `glazed.parameter:"target_language"`
}

func NewSummarizeCommand() (*SummarizeCommand, error) {
	return &SummarizeCommand{
		CommandDescription: cmds.NewCommandDescription(
			"summarize",
			cmds.WithShort("Summarize content"),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"url",
					parameters.ParameterTypeString,
					parameters.WithHelp("URL to a document to summarize"),
				),
				parameters.NewParameterDefinition(
					"text",
					parameters.ParameterTypeStringFromFile,
					parameters.WithHelp("Text to summarize"),
					// NOTE(manuel, 2023-09-27) This exclusive with is pretty cool as an idea
					//parameters.WithExclusiveWith("url"),
				),
				parameters.NewParameterDefinition(
					"engine",
					parameters.ParameterTypeChoice,
					parameters.WithHelp("Summarization engine"),
					parameters.WithChoices([]string{"agnes", "cecil", "daphne", "muriel"}),
					parameters.WithDefault("cecil"),
				),
				parameters.NewParameterDefinition(
					"summary_type",
					parameters.ParameterTypeChoice,
					parameters.WithHelp("Type of summary to generate"),
					parameters.WithChoices([]string{"summary", "takeaway"}),
					parameters.WithDefault("summary"),
				),
				parameters.NewParameterDefinition(
					"target_language",
					parameters.ParameterTypeString,
					parameters.WithHelp("Target language for the summary"),
					parameters.WithDefault("en"),
				),
			),
		),
	}, nil
}

func (c *SummarizeCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	w io.Writer,
) error {
	token := viper.GetString("kagi-api-key")
	if token == "" {
		return errors.New("no API token provided")
	}

	s := &SummarizeSettings{}
	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
	if err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}

	// Construct the request
	reqData := SummarizationRequest{
		URL:            s.URL,
		Text:           s.Text,
		Engine:         s.Engine,
		SummaryType:    s.SummaryType,
		TargetLanguage: s.TargetLanguage,
		Cache:          false,
	}

	bodyData, err := json.Marshal(reqData)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request body")
	}

	req, err := http.NewRequest("POST", "https://kagi.com/api/v0/summarize", bytes.NewBuffer(bodyData))
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

	var response SummarizationResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return errors.Wrap(err, "failed to parse response body")
	}

	// Print tokens
	fmt.Printf("Tokens: %d\n", response.Data.Tokens)
	// Print the summarization result
	fmt.Println(response.Data.Output)

	return nil
}
