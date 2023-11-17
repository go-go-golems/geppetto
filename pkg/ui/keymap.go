package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	SelectPrevMessage key.Binding
	SelectNextMessage key.Binding
	UnfocusMessage    key.Binding
	FocusMessage      key.Binding
	SubmitMessage     key.Binding
	ScrollUp          key.Binding
	ScrollDown        key.Binding

	CancelCompletion key.Binding
	DismissError     key.Binding

	LoadFromFile key.Binding

	SaveToFile             key.Binding
	SaveSourceBlocksToFile key.Binding

	CopyToClipboard             key.Binding
	CopyLastResponseToClipboard key.Binding
	CopySourceBlocksToClipboard key.Binding

	Help key.Binding
	Quit key.Binding
}

var DefaultKeyMap = KeyMap{
	SelectPrevMessage: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "move up")),
	SelectNextMessage: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "move down"),
	),

	UnfocusMessage: key.NewBinding(
		key.WithKeys("esc", "ctrl+g"),
		key.WithHelp("esc", "unfocus message"),
	),
	FocusMessage: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "focus message"),
	),

	SubmitMessage: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "submit message"),
	),
	CancelCompletion: key.NewBinding(
		key.WithKeys("esc", "ctrl+g"),
		key.WithHelp("esc", "cancel completion"),
	),

	DismissError: key.NewBinding(
		key.WithKeys("esc", "ctrl+g"),
		key.WithHelp("esc", "dismiss error"),
	),

	SaveToFile: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save to file"),
	),
	SaveSourceBlocksToFile: key.NewBinding(
		key.WithKeys("ctrl+shift+s"),
		key.WithHelp("ctrl+shift+s", "save source blocks to file"),
	),

	CopyToClipboard: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "copy to clipboard"),
	),
	CopyLastResponseToClipboard: key.NewBinding(
		key.WithKeys("ctrl+shift+d"),
		key.WithHelp("ctrl+shift+d", "copy last response to clipboard"),
	),
	CopySourceBlocksToClipboard: key.NewBinding(
		key.WithKeys("ctrl+shift+c"),
		key.WithHelp("ctrl+shift+c", "copy source blocks to clipboard"),
	),

	ScrollUp: key.NewBinding(
		key.WithKeys("shift+pgup"),
		key.WithHelp("shift+pgup", "scroll up"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("shift+pgdown"),
		key.WithHelp("shift+pgdown", "scroll down"),
	),

	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help,
		k.SubmitMessage,
		k.DismissError,
		k.CancelCompletion,
		k.SaveToFile,
		k.Quit}
}
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.SelectPrevMessage, k.SelectNextMessage, k.UnfocusMessage, k.FocusMessage},
	}
}
