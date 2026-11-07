// Package main implements decision-pi-smoke: it builds a decision provider
// from Pi's agent directory (pi.AgentDir.Decider) and asks a query-routing
// question set, or one relevance question per --passage, emitting one row per
// answer.
//
// Usage:
//
//	go run ./cmd/examples/decision-pi-smoke run --query "how does the rebalance law preserve placements?"
//	go run ./cmd/examples/decision-pi-smoke run --query "..." --passage "first text" --passage "second text"
//	go run ./cmd/examples/decision-pi-smoke run --model lunaroute/djev --pi-agent-dir ~/.pi/agent --query "..." --format json
package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/pi"
)

type decideCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*decideCommand)(nil)

type decideSettings struct {
	Model          string   `glazed:"model"`
	PiAgentDir     string   `glazed:"pi-agent-dir"`
	Query          string   `glazed:"query"`
	Corpus         string   `glazed:"corpus"`
	Passage        []string `glazed:"passage"`
	TimeoutSeconds int      `glazed:"timeout-seconds"`
}

func newDecideCommand() *decideCommand {
	return &decideCommand{CommandDescription: cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Ask a decision model (System One) a question set using Pi's credentials"),
		cmds.WithLong(`Ask the routing question set (route: choice, needs_retrieval: noul,
complexity: score) about --query, or, with --passage (repeatable), one
relevance question per passage. Credentials come from Pi's agent directory.`),
		cmds.WithFlags(
			fields.New("model", fields.TypeString, fields.WithDefault("lunaroute/djev"), fields.WithHelp("Pi model reference of the decision model")),
			fields.New("pi-agent-dir", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Pi agent directory (default ~/.pi/agent)")),
			fields.New("query", fields.TypeString, fields.WithDefault("how does the rebalance law preserve product placements?"), fields.WithHelp("The query to decide about")),
			fields.New("corpus", fields.TypeString, fields.WithDefault("engineering design notes and project reports"), fields.WithHelp("One-line corpus description, given to the model in the state")),
			fields.New("passage", fields.TypeStringList, fields.WithHelp("A passage to grade for relevance (repeatable)")),
			fields.New("timeout-seconds", fields.TypeInteger, fields.WithDefault(30), fields.WithHelp("Request timeout in seconds")),
		),
	)}
}

func (c *decideCommand) RunIntoGlazeProcessor(ctx context.Context, parsed *values.Values, gp middlewares.Processor) error {
	s := &decideSettings{}
	if err := parsed.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return errors.Wrap(err, "decode settings")
	}
	a, err := pi.LoadAgentDir(pi.DefaultDir(s.PiAgentDir))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(max(s.TimeoutSeconds, 1))*time.Second)
	defer cancel()
	d, err := a.Decider(ctx, s.Model, pi.DecisionOptions{})
	if err != nil {
		return err
	}
	resp, err := d.Decide(ctx, request(s.Query, s.Corpus, s.Passage))
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(resp.Answers))
	for id := range resp.Answers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		a := resp.Answers[id]
		row := types.NewRow(types.MRP("model", resp.Model), types.MRP("question", id), types.MRP("type", string(a.Type)))
		switch a.Type {
		case decision.Noul:
			row.Set("value", a.Probability)
		case decision.Choice:
			row.Set("value", a.Choice)
			row.Set("confidence", a.Confidence)
		case decision.Score:
			row.Set("value", a.Score)
			row.Set("confidence", a.Confidence)
		}
		if resp.DurationMs != nil {
			row.Set("duration_ms", *resp.DurationMs)
		}
		row.Set("request_id", resp.RequestID)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func request(query, corpus string, passages []string) decision.Request {
	state := map[string]any{"query": query, "corpus": corpus}
	if len(passages) > 0 {
		qs := map[string]decision.Question{}
		for i, p := range passages {
			qs[fmt.Sprintf("p%d", i)] = decision.Question{
				Type:         decision.Noul,
				Instructions: map[string]any{"passage": p, "question": "Does this passage help answer the query?"},
				True:         "The passage contains facts needed to answer the query",
				False:        "The passage is off-topic or only loosely related",
			}
		}
		return decision.Request{State: state, Questions: qs}
	}
	return decision.Request{State: state, Questions: map[string]decision.Question{
		"route": {Type: decision.Choice, Instructions: "Which retrieval strategy fits this query best?", Options: []decision.Option{
			{Name: "keyword", Description: "Exact identifiers, names, error strings"},
			{Name: "semantic", Description: "Conceptual question needing paraphrase matching"},
			{Name: "none", Description: "Chit-chat or no retrieval needed"},
		}},
		"needs_retrieval": {Type: decision.Noul, Instructions: "Does answering require looking up documents in the corpus?",
			True: "The answer depends on facts in the corpus", False: "Answerable without documents"},
		"complexity": {Type: decision.Score, Instructions: "How complex is the question?",
			Levels: []string{"trivial lookup", "single fact", "explanation of one mechanism", "multi-document synthesis"}},
	}}
}

func main() {
	root := examplecmd.NewRoot("decision-pi-smoke", "Decision-model smoke test over Pi credentials")
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", newDecideCommand()))
}
