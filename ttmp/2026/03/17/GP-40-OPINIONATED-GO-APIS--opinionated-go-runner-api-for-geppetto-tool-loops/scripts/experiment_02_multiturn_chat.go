//go:build ignore

// Experiment 02: Multi-Turn Chat Runner (Design B's Runner with Design D's ergonomics)
//
// Shows how the Runner struct supports multi-turn conversations while
// keeping the simple option syntax.
//
// This file is a design sketch and is excluded from normal builds.

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/runner"
)

// --- Domain-specific tools ---

type QueryInput struct {
	SQL string `json:"sql" jsonschema:"description=SQL query to execute against the product database"`
}

type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Count   int             `json:"count"`
}

func sqlQuery(input QueryInput) (QueryResult, error) {
	// Simulated — in reality this would execute against a real DB
	return QueryResult{
		Columns: []string{"product", "revenue"},
		Rows:    [][]interface{}{{"Widget A", 15000}, {"Widget B", 23000}},
		Count:   2,
	}, nil
}

func calculator(input struct {
	Expression string `json:"expression" jsonschema:"description=Math expression to evaluate"`
}) (float64, error) {
	// Simulated
	return 42.0, nil
}

// --- Main: Interactive multi-turn chat ---

func main() {
	ctx := context.Background()

	// Create a reusable runner with tools.
	// The runner holds conversation state internally.
	r := runner.New(
		runner.System("You are a data analyst. Use SQL to answer questions about our product database."),
		runner.Tool("sql_query", "Execute a SQL query against the product database", sqlQuery),
		runner.Tool("calc", "Evaluate mathematical expressions", calculator),
		runner.MaxTools(10),
		runner.MaxTokens(4096),
	)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Data Analyst Assistant (type 'quit' to exit)")
	fmt.Println("---")

	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "quit" || input == "exit" {
			break
		}
		if input == "" {
			continue
		}

		fmt.Print("\nAssistant: ")

		// Chat() appends to conversation history automatically.
		// Previous turns (user messages, assistant responses, tool calls/results)
		// are all preserved and sent to the LLM for context.
		result, err := r.Chat(ctx, input,
			runner.Stream(func(delta string) {
				fmt.Print(delta)
			}),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
			continue
		}

		fmt.Printf("\n  [%d tool calls | %d tokens in | %d tokens out]\n",
			len(result.ToolCalls), result.Usage.InputTokens, result.Usage.OutputTokens)
	}

	fmt.Println("\nGoodbye!")
}
