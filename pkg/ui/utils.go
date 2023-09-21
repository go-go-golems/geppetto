package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"golang.org/x/term"
	"os"
)

func drawBorderedMessage(msg string) string {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	width = width / 2

	style := lipgloss.NewStyle().
		Padding(1, 1).
		Border(lipgloss.RoundedBorder()).
		Width(width - 4)

	w := wordwrap.NewWriter(width - 4)
	_, err = fmt.Fprintf(w, msg)
	if err != nil {
		panic(err)
	}
	return style.Render(w.String())
}
