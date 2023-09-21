package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/muesli/reflow/wordwrap"
	"github.com/rs/zerolog/log"
)

type KeyMap struct {
	SelectPrevMessage key.Binding
	SelectNextMessage key.Binding
	UnfocusMessage    key.Binding
	FocusMessage      key.Binding
	SubmitMessage     key.Binding
	ScrollUp          key.Binding
	ScrollDown        key.Binding
	Quit              key.Binding
}

var DefaultKeyMap = KeyMap{
	SelectPrevMessage: key.NewBinding(key.WithKeys("up")),
	SelectNextMessage: key.NewBinding(key.WithKeys("down")),
	UnfocusMessage:    key.NewBinding(key.WithKeys("esc", "ctrl+g")),
	FocusMessage:      key.NewBinding(key.WithKeys("enter")),
	SubmitMessage:     key.NewBinding(key.WithKeys("shift+enter")),
	ScrollUp:          key.NewBinding(key.WithKeys("shift+pgup")),
	ScrollDown:        key.NewBinding(key.WithKeys("shift+pgdown")),
	Quit:              key.NewBinding(key.WithKeys("ctrl+c")),
}

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

type errMsg error

type model struct {
	contextManager *context.Manager
	// not really what we want, but use this for now, we'll have to either find a normal text box,
	// or implement wrapping ourselves.
	textArea textarea.Model
	// is the textarea currently focused
	focused bool
	// currently selected message, always valid
	selectedIdx int
	err         error
	keyMap      KeyMap

	style  *Style
	width  int
	height int
}

func (m *model) updateKeyBindings() {
	if m.focused {
		m.keyMap.SelectNextMessage.SetEnabled(false)
		m.keyMap.SelectPrevMessage.SetEnabled(false)
		m.keyMap.FocusMessage.SetEnabled(false)
		m.keyMap.UnfocusMessage.SetEnabled(true)
	} else {
		m.keyMap.SelectNextMessage.SetEnabled(true)
		m.keyMap.SelectPrevMessage.SetEnabled(true)
		m.keyMap.FocusMessage.SetEnabled(true)
		m.keyMap.UnfocusMessage.SetEnabled(false)
	}
}

func initialModel(manager *context.Manager) model {
	ret := model{
		contextManager: manager,
		style:          DefaultStyles(),
		keyMap:         DefaultKeyMap,
	}

	ret.textArea = textarea.New()
	ret.textArea.Placeholder = "Once upon a time..."
	ret.textArea.Focus()
	ret.focused = true

	ret.selectedIdx = len(ret.contextManager.GetMessages()) - 1

	ret.updateKeyBindings()

	return ret
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.UnfocusMessage):
			// we need to access with the pointer
			if m.focused {
				m.textArea.Blur()
				m.focused = false
				m.updateKeyBindings()
			}
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.FocusMessage):
			if !m.focused {
				cmd = m.textArea.Focus()

				m.focused = true
				m.updateKeyBindings()
			}

		case key.Matches(msg, m.keyMap.SelectNextMessage):
			if m.selectedIdx < len(m.contextManager.GetMessages())-1 {
				m.selectedIdx++
			}

		case key.Matches(msg, m.keyMap.SelectPrevMessage):
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}

		case key.Matches(msg, m.keyMap.SubmitMessage):
			if m.focused {
				// XXX actually send the whole context to the LLM
			}

		default:
			if m.focused {
				m.textArea, cmd = m.textArea.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		h, _ := m.style.SelectedMessage.GetFrameSize()
		newWidth := msg.Width - h
		m.textArea.SetWidth(newWidth)
		m.width = msg.Width
		m.height = msg.Height

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil

	default:
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	ret := ""

	for idx := range m.contextManager.GetMessages() {
		v := m.contextManager.GetMessages()[idx].Text

		w, _ := m.style.SelectedMessage.GetFrameSize()

		w_ := wordwrap.NewWriter(m.width - w)
		_, err := fmt.Fprintf(w_, v)
		if err != nil {
			panic(err)
		}
		v = w_.String()
		if idx == m.selectedIdx && !m.focused {
			v = m.style.SelectedMessage.Render(v)
		} else {
			v = m.style.UnselectedMessage.Render(v)
		}
		ret += v
		ret += "\n"
	}

	v := m.textArea.View()
	if m.focused {
		v = m.style.FocusedMessage.Render(v)
	} else {
		v = m.style.UnselectedMessage.Render(v)
	}

	ret += v
	ret += "\n"

	return ret
}

const veryLongLoremIpsum = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec a diam lectus. " +
	"Sed sit amet ipsum mauris. Maecenas congue ligula ac quam viverra nec consectetur ante hendrerit. " +
	"Donec et mollis dolor. Praesent et diam eget libero egestas mattis sit amet vitae augue. " +
	"Nam tincidunt congue enim, ut porta lorem lacinia consectetur. " +
	"Donec ut libero sed arcu vehicula ultricies a non tortor. Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
	"Aenean ut gravida lorem. Ut turpis felis, pulvinar a semper sed, adipiscing id dolor. " +
	"Pellentesque auctor nisi id magna consequat sagittis. " +
	"Curabitur dapibus enim sit amet elit pharetra tincidunt feugiat nisl imperdiet. " +
	"Ut convallis libero in urna ultrices accumsan. Donec sed odio eros."

func main() {
	messages := []*context.Message{
		// different substrings of veryLongLoremIpsum
		{
			Text: veryLongLoremIpsum[:100],
			Role: "system",
		},
		{
			Text: veryLongLoremIpsum[100:300],
			Role: "assistant",
		},
		{
			Text: veryLongLoremIpsum[200:400],
			Role: "user",
		},
	}
	ctxtManager := context.NewManager(context.WithMessages(messages))

	p := tea.NewProgram(initialModel(ctxtManager))

	if _, err := p.Run(); err != nil {
		log.Fatal().Err(err).Msg("Error while handling bubbletea")
	}
}
