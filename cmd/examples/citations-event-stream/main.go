package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/events/structuredsink"
	"github.com/go-go-golems/geppetto/pkg/events/structuredsink/parsehelpers"
	"github.com/google/uuid"
)

// --- Citation extractor (same as test) ---

type CitationItem struct {
	Title   string
	Authors []string
}

type citationsPayload struct {
	Citations []CitationItem `yaml:"citations"`
}

type EventCitationStarted struct {
	events.EventImpl
	ItemID string `json:"item_id"`
}

type EventCitationDelta struct {
	events.EventImpl
	ItemID string `json:"item_id"`
	Delta  string `json:"delta"`
}

type EventCitationUpdate struct {
	events.EventImpl
	ItemID  string         `json:"item_id"`
	Entries []CitationItem `json:"entries,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type EventCitationCompleted struct {
	events.EventImpl
	ItemID  string         `json:"item_id"`
	Entries []CitationItem `json:"entries,omitempty"`
	Success bool           `json:"success"`
	Error   string         `json:"error,omitempty"`
}

type citationsExtractor struct{ name, dtype string }

func (ce *citationsExtractor) Name() string     { return ce.name }
func (ce *citationsExtractor) DataType() string { return ce.dtype }
func (ce *citationsExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) structuredsink.ExtractorSession {
	return &citationsSession{ctx: ctx, itemID: itemID}
}

type citationsSession struct {
	ctx         context.Context
	itemID      string
	lastValid   []CitationItem
	lastValidOK bool
	ctrl        *parsehelpers.YAMLController[citationsPayload]
}

func (cs *citationsSession) OnStart(ctx context.Context) []events.Event {
	// reset streaming YAML controller for a fresh block
	cs.ctrl = nil
	return []events.Event{&EventCitationStarted{EventImpl: events.EventImpl{Type_: "citations-started"}, ItemID: cs.itemID}}
}

func (cs *citationsSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
	if cs.ctrl == nil {
		cs.ctrl = parsehelpers.NewDebouncedYAML[citationsPayload](parsehelpers.DebounceConfig{
			SnapshotEveryBytes: 512,
			SnapshotOnNewline:  true,
			ParseTimeout:       0,
			MaxBytes:           64 << 10,
		})
	}
	evs := []events.Event{&EventCitationDelta{EventImpl: events.EventImpl{Type_: "citations-delta"}, ItemID: cs.itemID, Delta: string(chunk)}}
	if snap, err := cs.ctrl.FeedBytes(chunk); snap != nil || err != nil {
		var entries []CitationItem
		if err == nil && snap != nil {
			entries = snap.Citations
			if len(entries) > 0 {
				cs.lastValid = entries
				cs.lastValidOK = true
			}
		}
		// emit update when we have either entries or we already had a last valid (smooth UX)
		if len(entries) > 0 || cs.lastValidOK {
			if len(entries) == 0 {
				entries = cs.lastValid
			}
			evs = append(evs, &EventCitationUpdate{EventImpl: events.EventImpl{Type_: "citations-update"}, ItemID: cs.itemID, Entries: entries, Error: ""})
		}
	}
	return evs
}

func (cs *citationsSession) OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event {
	entries := cs.lastValid
	errStr := ""
	if err != nil {
		errStr = err.Error()
		success = false
	} else if raw != nil {
		if cs.ctrl == nil {
			cs.ctrl = parsehelpers.NewDebouncedYAML[citationsPayload](parsehelpers.DebounceConfig{})
		}
		if snap, perr := cs.ctrl.FinalBytes(raw); perr == nil && snap != nil && len(snap.Citations) > 0 {
			entries = snap.Citations
			cs.lastValid = entries
			cs.lastValidOK = true
		} else if perr != nil {
			errStr = perr.Error()
			success = false
		}
	}
	return []events.Event{&EventCitationCompleted{EventImpl: events.EventImpl{Type_: "citations-completed"}, ItemID: cs.itemID, Entries: entries, Success: success, Error: errStr}}
}

// stripCodeFenceBytes detects ```lang\n body \n``` blocks and returns (lang, body)
// note: fence stripping is handled by parsehelpers in this example

// --- Bubbletea model ---

type model struct {
	// Simulation state
	parts      []string
	currentIdx int
	autoPlay   bool
	speed      time.Duration
	streamID   uuid.UUID
	meta       events.EventMetadata
	sink       *structuredsink.FilteringSink
	collector  *eventCollector
	completion string

	// Display state
	rawText        string
	filteredText   string
	citations      []CitationItem
	allCitations   []CitationItem
	citationErr    string
	rawEvents      []string
	filteredEvents []string
	citationEvents []string
	width          int
	height         int
	// User view state
	activeCitations int
	spinnerIdx      int
	segments        []userSeg
	spinner         spinner.Model
	// Viewports for scrollable content
	userViewport           viewport.Model
	rawTextViewport        viewport.Model
	filteredTextViewport   viewport.Model
	citationsViewport      viewport.Model
	rawEventsViewport      viewport.Model
	filteredEventsViewport viewport.Model
	citationEventsViewport viewport.Model
}

type userSeg struct {
	kind    string // "text" | "citations"
	text    string
	entries []CitationItem
	active  bool
	error   string
}

type eventCollector struct {
	list []events.Event
}

func (ec *eventCollector) PublishEvent(ev events.Event) error {
	ec.list = append(ec.list, ev)
	return nil
}

type tickMsg time.Time

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func initialModel() model {
	// Build a sample stream with citations
	parts := []string{
		"Intro: In this demo we will stream multiple fenced blocks interlaced with text.\n\n",
		// Block 1
		"<$citations:v1>",
		"```yaml\n",
		"citations:\n",
		"  - title: AlphaGo\n",
		"    authors:\n",
		"      - Silver\n",
		"  - title: GPT-4 Technical Report\n",
		"    authors:\n",
		"      - OpenAI\n",
		"```\n",
		"</$citations:v1>",
		"\nBetween blocks, normal prose appears here.\n\n",
		// Block 2
		"<$citations:v1>",
		"```yaml\n",
		"citations:\n",
		"  - title: Sequence to Sequence Learning\n",
		"    authors:\n",
		"      - Sutskever\n",
		"  - title: Transformer-XL\n",
		"    authors:\n",
		"      - Dai\n",
		"```\n",
		"</$citations:v1>",
		"\nSome additional commentary between blocks.\n\n",
		// Block 3 (slightly more compact emission)
		"<$citations:v1>",
		"```yaml\ncitations:\n",
		"  - title: T5\n",
		"    authors:\n",
		"      - Raffel\n",
		"```\n",
		"</$citations:v1>",
		"\nDone.",
	}

	streamID := uuid.New()
	meta := events.EventMetadata{ID: streamID}
	collector := &eventCollector{}
	ex := &citationsExtractor{name: "citations", dtype: "v1"}
	sink := structuredsink.NewFilteringSink(collector, structuredsink.Options{
		Debug: false,
	}, ex)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	return model{
		parts:      parts,
		currentIdx: 0,
		autoPlay:   false,
		speed:      300 * time.Millisecond,
		streamID:   streamID,
		meta:       meta,
		sink:       sink,
		collector:  collector,
		completion: "",
		spinner:    s,
		// Initialize viewports with default sizes (will be resized on first render)
		userViewport:           viewport.New(100, 10),
		rawTextViewport:        viewport.New(30, 5),
		filteredTextViewport:   viewport.New(30, 5),
		citationsViewport:      viewport.New(30, 5),
		rawEventsViewport:      viewport.New(30, 5),
		filteredEventsViewport: viewport.New(30, 5),
		citationEventsViewport: viewport.New(30, 5),
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ":
			// Toggle auto-play
			m.autoPlay = !m.autoPlay
			if m.autoPlay && m.currentIdx < len(m.parts) {
				return m, tick(m.speed)
			}
		case "n", "right":
			// Step forward
			if m.currentIdx < len(m.parts) {
				m = m.step()
			}
		case "r":
			// Reset
			return initialModel(), initialModel().Init()
		case "+":
			// Speed up
			if m.speed > 50*time.Millisecond {
				m.speed -= 50 * time.Millisecond
			}
		case "-":
			// Slow down
			m.speed += 50 * time.Millisecond
		}

	case tickMsg:
		if m.autoPlay && m.currentIdx < len(m.parts) {
			m = m.step()
			if m.currentIdx < len(m.parts) {
				cmds = append(cmds, tick(m.speed))
			}
		}
		// advance spinner regardless (it only shows when active)
		m.spinnerIdx = (m.spinnerIdx + 1) % 4

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update spinner
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) step() model {
	if m.currentIdx >= len(m.parts) {
		return m
	}

	part := m.parts[m.currentIdx]
	m.completion += part
	m.rawText = m.completion
	m.currentIdx++

	// Track events before this step
	beforeCount := len(m.collector.list)

	// Publish partial
	_ = m.sink.PublishEvent(events.NewPartialCompletionEvent(m.meta, part, m.completion))

	// Track raw event
	m.rawEvents = append(m.rawEvents, fmt.Sprintf("partial: delta=%q", truncate(part, 30)))

	// If we reached the end, publish final
	if m.currentIdx == len(m.parts) {
		_ = m.sink.PublishEvent(events.NewFinalEvent(m.meta, m.completion))
		m.rawEvents = append(m.rawEvents, "final")
	}

	// Collect new events emitted by sink
	for i := beforeCount; i < len(m.collector.list); i++ {
		ev := m.collector.list[i]
		switch e := ev.(type) {
		case *events.EventPartialCompletion:
			m.filteredEvents = append(m.filteredEvents, fmt.Sprintf("partial: delta=%q", truncate(e.Delta, 30)))
			if e.Delta != "" {
				m.segments = append(m.segments, userSeg{kind: "text", text: e.Delta})
			}
		case *events.EventFinal:
			m.filteredEvents = append(m.filteredEvents, "final")
		case *EventCitationStarted:
			m.citationEvents = append(m.citationEvents, fmt.Sprintf("started: %s", e.ItemID))
			m.activeCitations++
			m.segments = append(m.segments, userSeg{kind: "citations", active: true})
		case *EventCitationDelta:
			m.citationEvents = append(m.citationEvents, fmt.Sprintf("delta: %q", truncate(e.Delta, 30)))
		case *EventCitationUpdate:
			m.citationEvents = append(m.citationEvents, fmt.Sprintf("update: %d entries", len(e.Entries)))
			// update last active citations segment
			for j := len(m.segments) - 1; j >= 0; j-- {
				if m.segments[j].kind == "citations" && m.segments[j].active {
					m.segments[j].entries = e.Entries
					m.segments[j].error = e.Error
					break
				}
			}
		case *EventCitationCompleted:
			status := "success"
			if !e.Success {
				status = "failed"
			}
			m.citationEvents = append(m.citationEvents, fmt.Sprintf("completed: %s (%d entries)", status, len(e.Entries)))
			if m.activeCitations > 0 {
				m.activeCitations--
			}
			// append entries from this completed block to the aggregated list
			if len(e.Entries) > 0 {
				m.allCitations = append(m.allCitations, e.Entries...)
			}
			for j := len(m.segments) - 1; j >= 0; j-- {
				if m.segments[j].kind == "citations" && m.segments[j].active {
					m.segments[j].entries = e.Entries
					m.segments[j].active = false
					m.segments[j].error = e.Error
					break
				}
			}
		}
	}

	// Extract filtered text from last partial/final
	for i := len(m.collector.list) - 1; i >= 0; i-- {
		ev := m.collector.list[i]
		if pc, ok := ev.(*events.EventPartialCompletion); ok {
			m.filteredText = pc.Completion
			break
		}
		if fe, ok := ev.(*events.EventFinal); ok {
			m.filteredText = fe.Text
			break
		}
	}

	// Extract latest citation update (or final)
	for i := len(m.collector.list) - 1; i >= 0; i-- {
		ev := m.collector.list[i]
		if cu, ok := ev.(*EventCitationUpdate); ok {
			m.citations = cu.Entries
			m.citationErr = cu.Error
			break
		}
		if cc, ok := ev.(*EventCitationCompleted); ok {
			m.citations = cc.Entries
			if !cc.Success {
				m.citationErr = cc.Error
			} else {
				m.citationErr = ""
			}
			break
		}
	}

	return m
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

func (m model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))

	title := titleStyle.Render("Citations Event Stream Demo")

	// Calculate box width for 3 columns and dynamic heights
	boxWidth := (m.width / 3) - 3
	if m.width <= 0 {
		m.width = 120
	}
	if m.height <= 0 {
		m.height = 40
	}

	// Fixed overhead calculation:
	// - title(1) + blank(1) = 2
	// - userSection label(1) + blank(1) = 2
	// - contentSection label(1) + blank(1) = 2
	// - eventsSection label(1) + blank(1) = 2
	// - status(1) + help(1) = 2
	// Total non-box overhead: 10 lines
	//
	// Box border overhead (lipgloss adds borders):
	// - Each box with Height(N) actually renders N+2 lines (top+bottom border)
	// - We have 3 boxes: userBox, contentRow (with 3 sub-boxes), eventsRow (with 3 sub-boxes)
	// - But contentRow and eventsRow are joined horizontally, so only 2 lines border each
	// Total box border overhead: userBox(2) + contentRow(2) + eventsRow(2) = 6 lines
	//
	// Total fixed overhead: 10 + 6 = 16 lines
	fixedOverhead := 16
	availableHeight := m.height - fixedOverhead
	if availableHeight < 12 {
		availableHeight = 12 // absolute minimum
	}

	// Distribute: User View 50%, Content 25%, Events 25%
	// But enforce hard constraint: never exceed terminal height
	userHeight := availableHeight / 2
	if userHeight < 4 {
		userHeight = 4
	}
	remaining := availableHeight - userHeight
	contentBoxHeight := remaining / 2
	eventBoxHeight := remaining - contentBoxHeight
	if contentBoxHeight < 3 {
		contentBoxHeight = 3
	}
	if eventBoxHeight < 3 {
		eventBoxHeight = 3
	}

	// Final sanity check: ensure we don't exceed available height
	totalBoxHeight := userHeight + contentBoxHeight + eventBoxHeight
	if totalBoxHeight > availableHeight {
		// Scale down proportionally
		scale := float64(availableHeight) / float64(totalBoxHeight)
		userHeight = int(float64(userHeight) * scale)
		contentBoxHeight = int(float64(contentBoxHeight) * scale)
		eventBoxHeight = availableHeight - userHeight - contentBoxHeight
	}

	// --- Top Section: Content Streams ---
	contentSection := sectionStyle.Render("Content Streams")

	// Left: raw text stream (viewport-capped)
	rawTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("Raw Stream")
	rawContent := m.rawText
	if rawContent == "" {
		rawContent = "(waiting...)"
	}
	m.rawTextViewport.Width = boxWidth - 4          // box border (2) + viewport padding (2)
	m.rawTextViewport.Height = contentBoxHeight - 4 // box border (2) + title+blank (2)
	m.rawTextViewport.SetContent(rawContent)
	m.rawTextViewport.GotoBottom()
	rawBoxContent := rawTitle + "\n\n" + m.rawTextViewport.View()
	rawBox := boxStyle.Width(boxWidth).Render(rawBoxContent)

	// Middle: filtered text (viewport-capped)
	filteredTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("Filtered Text")
	filteredContent := m.filteredText
	if filteredContent == "" {
		filteredContent = "(waiting...)"
	}
	m.filteredTextViewport.Width = boxWidth - 4
	m.filteredTextViewport.Height = contentBoxHeight - 4
	m.filteredTextViewport.SetContent(filteredContent)
	m.filteredTextViewport.GotoBottom()
	filteredBoxContent := filteredTitle + "\n\n" + m.filteredTextViewport.View()
	filteredBox := boxStyle.Width(boxWidth).Render(filteredBoxContent)

	// Right: citations (streaming by block) (viewport-capped)
	citationsTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("Citations Extracted")
	var citationsContent strings.Builder
	frames2 := []string{"|", "/", "-", "\\"}
	spinner2 := frames2[m.spinnerIdx%len(frames2)]
	blockNum := 0
	for _, seg := range m.segments {
		if seg.kind != "citations" {
			continue
		}
		blockNum++
		status := ""
		if seg.active {
			status = fmt.Sprintf(" [%s streaming]", spinner2)
		}
		citationsContent.WriteString(fmt.Sprintf("Block %d%s\n", blockNum, status))
		if len(seg.entries) == 0 {
			if seg.active {
				citationsContent.WriteString("  (waiting...)\n")
			} else {
				citationsContent.WriteString("  (no citations)\n")
			}
		} else {
			for i, c := range seg.entries {
				citationsContent.WriteString(fmt.Sprintf("  %d. %s\n", i+1, c.Title))
				if len(c.Authors) > 0 {
					citationsContent.WriteString(fmt.Sprintf("     Authors: %s\n", strings.Join(c.Authors, ", ")))
				}
			}
		}
		citationsContent.WriteString("\n")
	}
	if blockNum == 0 {
		citationsContent.WriteString("(no citation blocks yet)")
	}
	m.citationsViewport.Width = boxWidth - 4
	m.citationsViewport.Height = contentBoxHeight - 4
	m.citationsViewport.SetContent(citationsContent.String())
	m.citationsViewport.GotoBottom()
	citationsBoxContent := citationsTitle + "\n\n" + m.citationsViewport.View()
	citationsBox := boxStyle.Width(boxWidth).Render(citationsBoxContent)

	contentRow := lipgloss.JoinHorizontal(lipgloss.Top, rawBox, filteredBox, citationsBox)

	// --- Bottom Section: Event Streams ---
	eventsSection := sectionStyle.Render("Event Streams")

	// Left: raw events (viewport-capped)
	rawEventsTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("Input Events")
	rawEventsContent := formatEventList(m.rawEvents, -1) // -1 = all events
	m.rawEventsViewport.Width = boxWidth - 4
	m.rawEventsViewport.Height = eventBoxHeight - 4
	m.rawEventsViewport.SetContent(rawEventsContent)
	m.rawEventsViewport.GotoBottom()
	rawEventsBoxContent := rawEventsTitle + "\n\n" + m.rawEventsViewport.View()
	rawEventsBox := boxStyle.Width(boxWidth).Render(rawEventsBoxContent)

	// Middle: filtered events (viewport-capped)
	filteredEventsTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("Filtered Events")
	filteredEventsContent := formatEventList(m.filteredEvents, -1) // -1 = all events
	m.filteredEventsViewport.Width = boxWidth - 4
	m.filteredEventsViewport.Height = eventBoxHeight - 4
	m.filteredEventsViewport.SetContent(filteredEventsContent)
	m.filteredEventsViewport.GotoBottom()
	filteredEventsBoxContent := filteredEventsTitle + "\n\n" + m.filteredEventsViewport.View()
	filteredEventsBox := boxStyle.Width(boxWidth).Render(filteredEventsBoxContent)

	// Right: citation events (viewport-capped)
	citationEventsTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("Citation Events")
	citationEventsContent := formatEventList(m.citationEvents, -1) // -1 = all events
	m.citationEventsViewport.Width = boxWidth - 4
	m.citationEventsViewport.Height = eventBoxHeight - 4
	m.citationEventsViewport.SetContent(citationEventsContent)
	m.citationEventsViewport.GotoBottom()
	citationEventsBoxContent := citationEventsTitle + "\n\n" + m.citationEventsViewport.View()
	citationEventsBox := boxStyle.Width(boxWidth).Render(citationEventsBoxContent)

	eventsRow := lipgloss.JoinHorizontal(lipgloss.Top, rawEventsBox, filteredEventsBox, citationEventsBox)

	// Status
	status := fmt.Sprintf("Step %d/%d", m.currentIdx, len(m.parts))
	if m.autoPlay {
		status += " [AUTO-PLAY]"
	}
	status += fmt.Sprintf(" | Speed: %dms", m.speed.Milliseconds())

	// Help
	help := helpStyle.Render("space: toggle auto-play | n/→: step | +/-: speed | r: reset | q: quit")

	// --- User View (chat-like) at the top ---
	userSection := sectionStyle.Render("User View")
	// nested widget style for citations blocks
	citationWidgetStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).Padding(0, 1).Margin(0, 0, 1, 0)
	var userContent strings.Builder
	for _, seg := range m.segments {
		switch seg.kind {
		case "text":
			userContent.WriteString(seg.text)
		case "citations":
			// render inline citations widget with a colored border
			var segBuf strings.Builder
			if len(seg.entries) == 0 {
				if seg.active {
					segBuf.WriteString(m.spinner.View() + " streaming citations...\n")
				} else {
					segBuf.WriteString("(no citations)\n")
				}
			} else {
				for i, c := range seg.entries {
					segBuf.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
					if len(c.Authors) > 0 {
						segBuf.WriteString(fmt.Sprintf("   Authors: %s\n", strings.Join(c.Authors, ", ")))
					}
				}
				if seg.active {
					segBuf.WriteString("\n" + m.spinner.View() + " streaming citations...\n")
				}
			}
			userContent.WriteString("\n")
			userContent.WriteString(citationWidgetStyle.Render(segBuf.String()))
			userContent.WriteString("\n")
		}
	}
	// Cap user view with viewport to avoid pushing other sections
	m.userViewport.Width = m.width - 4 - 4 // box border (2) + internal padding (2)
	m.userViewport.Height = userHeight - 2 // box border (2)
	m.userViewport.SetContent(userContent.String())
	m.userViewport.GotoBottom()
	userBox := boxStyle.Width(m.width - 4).Render(m.userViewport.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		userSection,
		userBox,
		"",
		contentSection,
		contentRow,
		"",
		eventsSection,
		eventsRow,
		"",
		status,
		help,
	)
}

func formatEventList(events []string, maxRecent int) string {
	if len(events) == 0 {
		return "(no events yet)"
	}
	start := 0
	if maxRecent > 0 && len(events) > maxRecent {
		start = len(events) - maxRecent
	}
	var b strings.Builder
	for i := start; i < len(events); i++ {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, events[i]))
	}
	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
