package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/openai/chat"
	"github.com/muesli/reflow/wordwrap"
)

type errMsg error

type model struct {
	contextManager *context.Manager

	viewport viewport.Model

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
	step   *chat.Step
	// if not nil, streaming is going on
	stepResult *steps.StepResult[string]
}

func (m *model) updateKeyBindings() {
	if m.focused {
		m.keyMap.SelectNextMessage.SetEnabled(false)
		m.keyMap.SelectPrevMessage.SetEnabled(false)
		m.keyMap.FocusMessage.SetEnabled(false)
		m.keyMap.UnfocusMessage.SetEnabled(true)
		m.keyMap.SubmitMessage.SetEnabled(true)
	} else {
		m.keyMap.SelectNextMessage.SetEnabled(true)
		m.keyMap.SelectPrevMessage.SetEnabled(true)
		m.keyMap.FocusMessage.SetEnabled(true)
		m.keyMap.UnfocusMessage.SetEnabled(false)
		m.keyMap.SubmitMessage.SetEnabled(false)
	}
}

func initialModel(manager *context.Manager, step *chat.Step) model {
	ret := model{
		contextManager: manager,
		style:          DefaultStyles(),
		keyMap:         DefaultKeyMap,
		step:           step,
		viewport:       viewport.New(0, 0),
	}

	ret.textArea = textarea.New()
	ret.textArea.Placeholder = "Once upon a time..."
	ret.textArea.Focus()
	ret.focused = true

	ret.selectedIdx = len(ret.contextManager.GetMessages()) - 1

	messages := ret.messageView()
	ret.viewport.SetContent(messages)
	ret.viewport.YPosition = 0

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
				//cmd := m.submit()
				//cmds = append(cmds, cmd)
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

		headerHeight := lipgloss.Height(m.headerView())
		textAreaHeight := lipgloss.Height(m.textAreaView())
		newHeight := msg.Height - textAreaHeight - headerHeight
		m.viewport.Width = m.width
		m.viewport.Height = newHeight - 2
		m.viewport.YPosition = headerHeight

		m.viewport.SetContent(m.messageView())

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil

	default:
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) headerView() string {
	return "PINOCCHIO AT YOUR SERVICE:"
}

func (m model) messageView() string {
	ret := ""

	for idx := range m.contextManager.GetMessagesWithSystemPrompt() {
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

	return ret
}

func (m model) textAreaView() string {
	v := m.textArea.View()
	if m.focused {
		v = m.style.FocusedMessage.Render(v)
	} else {
		v = m.style.UnselectedMessage.Render(v)
	}

	return v
}

func (m model) View() string {
	ret := m.headerView() + "\n" + m.viewport.View() + "\n" + m.textAreaView()
	//ret := "foofoo\n" + m.textAreaView()

	return ret
}

//func (m *model) submit() tea.Cmd {
//	m.focused = false
//	m.updateKeyBindings()
//
//	//var err error
//	m.stepResult, err = m.step.Start(context2.Background(), m.contextManager.GetMessagesWithSystemPrompt())
//
//	return
//}
