package main

import (
	tea "github.com/charmbracelet/bubbletea"
	chat2 "github.com/go-go-golems/bobatea/pkg/chat"
	ui2 "github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/ui"
	"time"
)

func main() {
	manager := context.NewManager(context.WithMessages([]*ui2.Message{
		ui2.NewMessage("hahahahaha", ui2.RoleSystem),
	}))

	step := &chat.EchoStep{
		TimePerCharacter: 150 * time.Millisecond,
	}

	options := []tea.ProgramOption{
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	}
	options = append(options, tea.WithAltScreen())

	backend := ui.NewStepBackend(step)
	p := tea.NewProgram(
		chat2.InitialModel(manager, backend),
		options...,
	)

	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
