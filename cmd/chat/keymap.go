package main

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	SelectPrevMessage           key.Binding
	SelectNextMessage           key.Binding
	UnfocusMessage              key.Binding
	FocusMessage                key.Binding
	SubmitMessage               key.Binding
	ScrollUp                    key.Binding
	ScrollDown                  key.Binding
	Quit                        key.Binding
	CancelCompletion            key.Binding
	SaveToFile                  key.Binding
	LoadFromFile                key.Binding
	CopyToClipboard             key.Binding
	CopyLastResponseToClipboard key.Binding
	SaveSourceBlocksToFile      key.Binding
	CopySourceBlocksToClipboard key.Binding
}

var DefaultKeyMap = KeyMap{
	SelectPrevMessage: key.NewBinding(key.WithKeys("up")),
	SelectNextMessage: key.NewBinding(key.WithKeys("down")),
	UnfocusMessage:    key.NewBinding(key.WithKeys("esc", "ctrl+g")),
	FocusMessage:      key.NewBinding(key.WithKeys("enter")),
	SubmitMessage:     key.NewBinding(key.WithKeys("tab")),
	ScrollUp:          key.NewBinding(key.WithKeys("shift+pgup")),
	ScrollDown:        key.NewBinding(key.WithKeys("shift+pgdown")),
	Quit:              key.NewBinding(key.WithKeys("ctrl+c")),
}
