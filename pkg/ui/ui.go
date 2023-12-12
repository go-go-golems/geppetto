package ui

import (
	context2 "context"
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/pkg/errors"
	"time"
)

type errMsg error

// states:
// - user input
// - user moving around messages
// - stream completion
// - showing error

type State string

const (
	StateUserInput        State = "user_input"
	StateMovingAround     State = "moving_around"
	StateStreamCompletion State = "stream_completion"
	StateError            State = "error"
)

type model struct {
	contextManager *context.Manager

	viewport viewport.Model

	// not really what we want, but use this for now, we'll have to either find a normal text box,
	// or implement wrapping ourselves.
	textArea textarea.Model

	help help.Model

	// currently selected message, always valid
	selectedIdx int
	err         error
	keyMap      KeyMap

	style  *Style
	width  int
	height int

	step chat.Step
	// if not nil, streaming is going on
	stepResult steps.StepResult[string]

	currentResponse        string
	previousResponseHeight int

	state        State
	quitReceived bool
}

type StreamDoneMsg struct {
}

type StreamCompletionMsg struct {
	Completion string
}

type StreamCompletionError struct {
	Err error
}

func InitialModel(manager *context.Manager, step chat.Step) model {
	ret := model{
		contextManager: manager,
		style:          DefaultStyles(),
		keyMap:         DefaultKeyMap,
		step:           step,
		viewport:       viewport.New(0, 0),
		help:           help.New(),
	}

	ret.textArea = textarea.New()
	ret.textArea.Placeholder = "Once upon a time..."
	ret.textArea.Focus()
	ret.state = StateUserInput

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
			if m.state == StateUserInput {
				m.textArea.Blur()
				m.state = StateMovingAround
				m.updateKeyBindings()
			}
		case key.Matches(msg, m.keyMap.Quit):
			if !m.quitReceived {
				m.quitReceived = true
				// on first quit, try to cancel completion if running
				m.step.Interrupt()
			}

			if m.stepResult != nil {
				// force save completion before quitting
				m.finishCompletion()
			}

			return m, tea.Quit

		case key.Matches(msg, m.keyMap.FocusMessage):
			if m.state == StateMovingAround {
				cmd = m.textArea.Focus()
				cmds = append(cmds, cmd)

				m.state = StateUserInput
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
			if m.state == StateUserInput {
				cmd := m.submit()
				cmds = append(cmds, cmd)
			}

		case key.Matches(msg, m.keyMap.SaveToFile):
			// TODO(manuel, 2023-11-14) Implement file chosing dialog
			err := m.contextManager.SaveToFile("/tmp/output.json")
			if err != nil {
				return m, func() tea.Msg {
					return errMsg(err)
				}
			}

		// same keybinding for both
		case key.Matches(msg, m.keyMap.CancelCompletion):
			if m.state == StateStreamCompletion {
				m.step.Interrupt()
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keyMap.DismissError):
			if m.state == StateError {
				m.err = nil
				m.state = StateUserInput
				m.updateKeyBindings()
			}

			return m, tea.Batch(cmds...)

		default:
			switch m.state {
			case StateUserInput:
				m.textArea, cmd = m.textArea.Update(msg)
				cmds = append(cmds, cmd)
			case StateMovingAround, StateStreamCompletion, StateError:
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

	// handle chat streaming messages
	case StreamCompletionMsg:
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

	case StreamDoneMsg:
		cmd = m.finishCompletion()
		cmds = append(cmds, cmd)

	case StreamCompletionError:
		cmd = m.setError(msg.Err)
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

func (m *model) updateKeyBindings() {
	m.keyMap.SaveToFile.SetEnabled(true)
	m.keyMap.SaveSourceBlocksToFile.SetEnabled(true)
	m.keyMap.CopyToClipboard.SetEnabled(true)
	m.keyMap.CopyLastResponseToClipboard.SetEnabled(true)
	m.keyMap.CopySourceBlocksToClipboard.SetEnabled(true)

	m.keyMap.SelectNextMessage.SetEnabled(m.state == StateMovingAround)
	m.keyMap.SelectPrevMessage.SetEnabled(m.state == StateMovingAround)
	m.keyMap.FocusMessage.SetEnabled(m.state == StateMovingAround)
	m.keyMap.UnfocusMessage.SetEnabled(m.state == StateUserInput)
	m.keyMap.SubmitMessage.SetEnabled(m.state == StateUserInput)

	m.keyMap.DismissError.SetEnabled(m.state == StateError)
	m.keyMap.CancelCompletion.SetEnabled(m.state == StateStreamCompletion)
}

func (m *model) recomputeSize() {
	headerView := m.headerView()
	headerHeight := lipgloss.Height(headerView)
	textAreaView := m.textAreaView()
	textAreaHeight := lipgloss.Height(textAreaView)

	helpView := m.help.View(m.keyMap)
	helpViewHeight := lipgloss.Height(helpView)

	m.previousResponseHeight = textAreaHeight
	newHeight := m.height - textAreaHeight - headerHeight - helpViewHeight
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
		message := m.contextManager.GetMessagesWithSystemPrompt()[idx]
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
	switch m.state {
	case StateUserInput:
		v = m.style.FocusedMessage.Render(v)
	case StateMovingAround, StateStreamCompletion:
		v = m.style.UnselectedMessage.Render(v)
	case StateError:
	}

	return v
}

func (m model) View() string {
	headerView := m.headerView()
	viewportView := m.viewport.View()
	textAreaView := m.textAreaView()
	helpView := m.help.View(m.keyMap)

	viewportHeight := lipgloss.Height(viewportView)
	_ = viewportHeight
	textAreaHeight := lipgloss.Height(textAreaView)
	_ = textAreaHeight
	headerHeight := lipgloss.Height(headerView)
	_ = headerHeight
	helpViewHeight := lipgloss.Height(helpView)
	_ = helpViewHeight
	ret := headerView + "\n" + viewportView + "\n" + textAreaView + "\n" + helpView

	return ret
}

// Chat completion messages
func (m *model) submit() tea.Cmd {
	if m.stepResult != nil {
		return func() tea.Msg {
			return errMsg(errors.New("already streaming"))
		}
	}

	m.contextManager.AddMessages(&context.Message{
		Role: context.RoleUser,
		Text: m.textArea.Value(),
		Time: time.Now(),
	})

	ctx := context2.Background()
	var err error
	m.stepResult, err = m.step.Start(ctx, m.contextManager.GetMessagesWithSystemPrompt())

	m.state = StateStreamCompletion
	m.updateKeyBindings()
	m.currentResponse = ""
	m.previousResponseHeight = 0

	m.viewport.GotoBottom()

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
		m.getNextCompletion(),
	)
}

func (m model) getNextCompletion() tea.Cmd {
	return func() tea.Msg {
		if m.stepResult == nil {
			return nil
		}
		// TODO(manuel, 2023-12-09) stream answers into the context manager
		c, ok := <-m.stepResult.GetChannel()
		if !ok {
			return StreamDoneMsg{}
		}
		v, err := c.Value()
		if err != nil {
			if errors.Is(err, context2.Canceled) {
				return StreamDoneMsg{}
			}
			return StreamCompletionError{err}
		}

		return StreamCompletionMsg{Completion: v}
	}
}

type refreshMessageMsg struct {
	GoToBottom bool
}

func (m *model) finishCompletion() tea.Cmd {
	// completion already finished, happens when error and completion finish or cancellation happen
	if m.stepResult == nil {
		return nil
	}

	m.contextManager.AddMessages(&context.Message{
		Role: context.RoleAssistant,
		Text: m.currentResponse,
		Time: time.Now(),
	})
	m.currentResponse = ""
	m.previousResponseHeight = 0
	m.step.Interrupt()
	m.stepResult = nil

	m.state = StateUserInput
	m.textArea.Focus()
	m.textArea.SetValue("")

	m.recomputeSize()
	m.updateKeyBindings()

	if m.quitReceived {
		return tea.Quit
	}

	return func() tea.Msg {
		return refreshMessageMsg{
			GoToBottom: true,
		}
	}
}

func (m *model) setError(err error) tea.Cmd {
	cmd := m.finishCompletion()
	m.err = err
	m.state = StateError
	m.updateKeyBindings()
	return cmd
}
