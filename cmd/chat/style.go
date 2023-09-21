package main

import "github.com/charmbracelet/lipgloss"

type Style struct {
	UnselectedMessage lipgloss.Style
	SelectedMessage   lipgloss.Style
	FocusedMessage    lipgloss.Style
}

type BorderColors struct {
	Unselected string
	Selected   string
	Focused    string
}

func DefaultStyles() *Style {
	lightModeColors := BorderColors{
		Unselected: "#CCCCCC",
		Selected:   "#FFB6C1", // Light pink
		Focused:    "#FFFF99", // Light yellow
	}

	darkModeColors := BorderColors{
		Unselected: "#444444",
		Selected:   "#DD7090", // Desaturated pink for dark mode
		Focused:    "#DDDD77", // Desaturated yellow for dark mode
	}

	return &Style{
		UnselectedMessage: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).
			Padding(1, 1).
			BorderForeground(lipgloss.AdaptiveColor{
				Light: lightModeColors.Unselected,
				Dark:  darkModeColors.Unselected,
			}),
		SelectedMessage: lipgloss.NewStyle().Border(lipgloss.ThickBorder()).
			Padding(1, 1).
			BorderForeground(lipgloss.AdaptiveColor{
				Light: lightModeColors.Selected,
				Dark:  darkModeColors.Selected,
			}),
		FocusedMessage: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).
			Padding(1, 1).
			BorderForeground(lipgloss.AdaptiveColor{
				Light: lightModeColors.Focused,
				Dark:  darkModeColors.Focused,
			}),
	}
}
