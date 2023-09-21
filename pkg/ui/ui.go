package ui

import (
	context2 "context"
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
	"github.com/pkg/errors"
	"time"
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

	step *chat.Step
	// if not nil, streaming is going on
	stepResult             *steps.StepResult[string]
	stepCancel             func()
	currentResponse        string
	previousResponseHeight int
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

func InitialModel(manager *context.Manager, step *chat.Step) model {
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
	ret.viewport.GotoBottom()

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
				cmd := m.submit()
				cmds = append(cmds, cmd)
			}

		case key.Matches(msg, m.keyMap.CancelCompletion):
			if m.stepCancel == nil {
				// shouldn't happen
			}
			m.stepCancel()
			return m, tea.Batch(cmds...)

		default:
			if m.focused {
				m.textArea, cmd = m.textArea.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.recomputeSize()

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil

	case streamCompletionMsg:
		m.currentResponse += msg.Completion
		newTextAreaView := m.textAreaView()
		newHeight := lipgloss.Height(newTextAreaView)
		if newHeight != m.previousResponseHeight {
			m.recomputeSize()
			m.previousResponseHeight = newHeight
		}
		cmds = append(cmds, func() tea.Msg {
			return refreshMessageMsg{}
		})
		cmd = m.getNextCompletion()
		cmds = append(cmds, cmd)

	case streamDoneMsg:
		cmd = m.finishCompletion()
		cmds = append(cmds, cmd)

	case streamCompletionError:
		cmd = m.finishCompletion()
		m.err = msg.Err
		cmds = append(cmds, cmd)

	case refreshMessageMsg:
		m.viewport.SetContent(m.messageView())
		m.recomputeSize()
		if msg.GoToBottom {
			m.viewport.GotoBottom()
		}

	default:
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) recomputeSize() {
	headerView := m.headerView()
	headerHeight := lipgloss.Height(headerView)
	textAreaView := m.textAreaView()
	textAreaHeight := lipgloss.Height(textAreaView)
	m.previousResponseHeight = textAreaHeight
	newHeight := m.height - textAreaHeight - headerHeight
	if newHeight < 0 {
		newHeight = 0
	}
	m.viewport.Width = m.width
	m.viewport.Height = newHeight
	m.viewport.YPosition = headerHeight + 1

	h, _ := m.style.SelectedMessage.GetFrameSize()

	m.textArea.SetWidth(m.width - h)

	messageView := m.messageView()
	m.viewport.SetContent(messageView)

	// TODO(manuel, 2023-09-21) Keep the current position by trying to match it to some message
	// This is probably going to be tricky
	m.viewport.GotoBottom()
}

func (m model) headerView() string {
	return "PINOCCHIO AT YOUR SERVICE:"
}

func (m model) messageView() string {
	ret := ""

	for idx := range m.contextManager.GetMessagesWithSystemPrompt() {
		message := m.contextManager.GetMessages()[idx]
		v := fmt.Sprintf("[%s]: %s", message.Role, message.Text)

		w, _ := m.style.SelectedMessage.GetFrameSize()

		v_ := wrapWords(v, m.width-w-m.style.SelectedMessage.GetHorizontalPadding())
		v_ = m.style.UnselectedMessage.Width(m.width - m.style.SelectedMessage.GetHorizontalPadding()).Render(v_)
		ret += v_
		ret += "\n"
	}

	return ret
}

func (m model) textAreaView() string {
	if m.err != nil {
		// TODO(manuel, 2023-09-21) Use a proper error style
		w, _ := m.style.SelectedMessage.GetFrameSize()
		v := wrapWords(m.err.Error(), m.width-w)
		return m.style.SelectedMessage.Render(v)
	}

	// we are currently streaming
	if m.stepResult != nil {
		w, _ := m.style.SelectedMessage.GetFrameSize()
		v := wrapWords(m.currentResponse, m.width-w-m.style.SelectedMessage.GetHorizontalPadding())
		// TODO(manuel, 2023-09-21) this is where we'd add the spinner
		return m.style.SelectedMessage.Width(m.width - m.style.SelectedMessage.GetHorizontalPadding()).Render(v)
	}

	v := m.textArea.View()
	if m.focused {
		v = m.style.FocusedMessage.Render(v)
	} else {
		v = m.style.UnselectedMessage.Render(v)
	}

	return v
}

func wrapWords(text string, w int) string {
	w_ := wordwrap.NewWriter(w)
	_, err := fmt.Fprintf(w_, text)
	if err != nil {
		panic(err)
	}
	_ = w_.Close()
	v := w_.String()
	return v
}

func (m model) View() string {
	headerView := m.headerView()
	viewportView := m.viewport.View()
	textAreaView := m.textAreaView()

	viewportHeight := lipgloss.Height(viewportView)
	_ = viewportHeight
	textAreaHeight := lipgloss.Height(textAreaView)
	_ = textAreaHeight
	headerHeight := lipgloss.Height(headerView)
	_ = headerHeight
	ret := headerView + "\n" + viewportView + "\n" + textAreaView

	return ret
}

type streamDoneMsg struct {
}
type streamCompletionMsg struct {
	Completion string
}

func (m *model) submit() tea.Cmd {
	if m.stepResult != nil {
		return func() tea.Msg {
			return errMsg(errors.New("already streaming"))
		}
	}

	m.keyMap.SubmitMessage.SetEnabled(false)
	m.keyMap.CancelCompletion.SetEnabled(true)

	m.contextManager.AddMessages(&context.Message{
		Role: "user",
		Text: m.textArea.Value(),
		Time: time.Now(),
	})

	m.focused = false
	m.updateKeyBindings()
	m.currentResponse = ""
	m.previousResponseHeight = 0

	m.viewport.GotoBottom()
	ctx, cancel := context2.WithCancel(context2.Background())
	m.stepCancel = cancel
	var err error
	m.stepResult, err = m.step.Start(ctx, m.contextManager.GetMessagesWithSystemPrompt())

	if err != nil {
		return func() tea.Msg {
			return errMsg(err)
		}
	}

	return tea.Batch(func() tea.Msg {
		return refreshMessageMsg{
			GoToBottom: true,
		}
	},
		m.getNextCompletion())
}

type streamCompletionError struct {
	Err error
}

func (m model) getNextCompletion() tea.Cmd {
	return func() tea.Msg {
		c, ok := <-m.stepResult.GetChannel()
		if !ok {
			return streamDoneMsg{}
		}
		v, err := c.Value()
		if err != nil {
			return streamCompletionError{err}
		}

		return streamCompletionMsg{Completion: v}
	}
}

type refreshMessageMsg struct {
	GoToBottom bool
}

func (m *model) finishCompletion() tea.Cmd {
	m.contextManager.AddMessages(&context.Message{
		Role: "assistant",
		Text: m.currentResponse,
		Time: time.Now(),
	})
	m.currentResponse = ""
	m.previousResponseHeight = 0
	m.stepCancel()
	m.stepResult = nil
	m.stepCancel = nil

	m.focused = true
	m.textArea.Focus()
	m.textArea.SetValue("")

	m.recomputeSize()
	m.updateKeyBindings()

	return func() tea.Msg {
		return refreshMessageMsg{
			GoToBottom: true,
		}
	}
}
