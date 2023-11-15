package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/ui"
	"time"
)

func main() {
	manager := context.NewManager(context.WithMessages([]*context.Message{
		&context.Message{
			Text: "hahahahaha",
			Time: time.Time{},
			Role: "system",
		},
	}))

	step := chat.EchoStep{}

	options := []tea.ProgramOption{
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	}
	options = append(options, tea.WithAltScreen())
	p := tea.NewProgram(
		ui.InitialModel(manager, step),
		options...,
	)

	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
