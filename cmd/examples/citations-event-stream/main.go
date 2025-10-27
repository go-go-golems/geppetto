package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/events/structuredsink"
	"github.com/google/uuid"
)

// --- Citation extractor (same as test) ---

type CitationItem struct {
	Title   string
	Authors []string
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
}

func (cs *citationsSession) OnStart(ctx context.Context) []events.Event {
	return []events.Event{&EventCitationStarted{EventImpl: events.EventImpl{Type_: "citations-started"}, ItemID: cs.itemID}}
}

func (cs *citationsSession) OnDelta(ctx context.Context, raw string) []events.Event {
	return []events.Event{&EventCitationDelta{EventImpl: events.EventImpl{Type_: "citations-delta"}, ItemID: cs.itemID, Delta: raw}}
}

func (cs *citationsSession) OnUpdate(ctx context.Context, snapshot map[string]any, parseErr error) []events.Event {
	// If parse error, return last valid state without error (incomplete YAML is expected during streaming)
	if parseErr != nil {
		if cs.lastValidOK {
			return []events.Event{&EventCitationUpdate{EventImpl: events.EventImpl{Type_: "citations-update"}, ItemID: cs.itemID, Entries: cs.lastValid, Error: ""}}
		}
		return nil // No valid state yet, skip update
	}

	var entries []CitationItem
	hasInvalid := false
	if snapshot != nil {
		v, ok := snapshot["citations"]
		if !ok {
			hasInvalid = true
		} else {
			list, ok := v.([]any)
			if !ok {
				hasInvalid = true
			} else {
				for _, it := range list {
					m, ok := it.(map[string]any)
					if !ok {
						hasInvalid = true
						continue
					}
					t, ok := m["title"].(string)
					if !ok || t == "" {
						hasInvalid = true
						continue
					}
					rawAuthors, ok := m["authors"].([]any)
					if !ok {
						hasInvalid = true
						continue
					}
					var authors []string
					validAuthors := true
					for _, au := range rawAuthors {
						s, ok := au.(string)
						if !ok || s == "" {
							validAuthors = false
							break
						}
						authors = append(authors, s)
					}
					if !validAuthors {
						hasInvalid = true
						continue
					}
					entries = append(entries, CitationItem{Title: t, Authors: authors})
				}
			}
		}
	}

	// If we got valid entries, update last valid state
	if !hasInvalid && len(entries) > 0 {
		cs.lastValid = entries
		cs.lastValidOK = true
		return []events.Event{&EventCitationUpdate{EventImpl: events.EventImpl{Type_: "citations-update"}, ItemID: cs.itemID, Entries: entries, Error: ""}}
	}

	// Invalid schema but we have last valid state - return it
	if cs.lastValidOK {
		return []events.Event{&EventCitationUpdate{EventImpl: events.EventImpl{Type_: "citations-update"}, ItemID: cs.itemID, Entries: cs.lastValid, Error: ""}}
	}

	// No valid state yet
	return nil
}

func (cs *citationsSession) OnCompleted(ctx context.Context, final map[string]any, success bool, err error) []events.Event {
	// Use last valid state or try to parse final
	entries := cs.lastValid
	errStr := ""
	if err != nil {
		errStr = err.Error()
		success = false
	} else if final != nil {
		// Try one last parse
		parsed := cs.parseEntries(final)
		if len(parsed) > 0 {
			entries = parsed
		}
	}
	return []events.Event{&EventCitationCompleted{
		EventImpl: events.EventImpl{Type_: "citations-completed"},
		ItemID:    cs.itemID,
		Entries:   entries,
		Success:   success,
		Error:     errStr,
	}}
}

func (cs *citationsSession) parseEntries(snapshot map[string]any) []CitationItem {
	var entries []CitationItem
	if snapshot == nil {
		return entries
	}
	v, ok := snapshot["citations"]
	if !ok {
		return entries
	}
	list, ok := v.([]any)
	if !ok {
		return entries
	}
	for _, it := range list {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		t, ok := m["title"].(string)
		if !ok || t == "" {
			continue
		}
		rawAuthors, ok := m["authors"].([]any)
		if !ok {
			continue
		}
		var authors []string
		validAuthors := true
		for _, au := range rawAuthors {
			s, ok := au.(string)
			if !ok || s == "" {
				validAuthors = false
				break
			}
			authors = append(authors, s)
		}
		if !validAuthors {
			continue
		}
		entries = append(entries, CitationItem{Title: t, Authors: authors})
	}
	return entries
}

// --- Bubbletea model ---

type model struct {
	// Simulation state
	parts       []string
	currentIdx  int
	autoPlay    bool
	speed       time.Duration
	streamID    uuid.UUID
	meta        events.EventMetadata
	sink        *structuredsink.FilteringSink
	collector   *eventCollector
	completion  string

	// Display state
	rawText         string
	filteredText    string
	citations       []CitationItem
	citationErr     string
	rawEvents       []string
	filteredEvents  []string
	citationEvents  []string
	width           int
	height          int
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
		"Here are some relevant papers:\n\n",
		"<$citations:v1>",
		"```yaml\n",
		"citations:\n",
		"  - title: Attention Is All You Need\n",
		"    authors:\n",
		"      - Vaswani\n",
		"      - Shazeer\n",
		"  - title: BERT Pre-training\n",
		"    authors:\n",
		"      - Devlin\n",
		"      - Chang\n",
		"```\n",
		"</$citations:v1>",
		"\n\nThese papers are foundational.",
	}

	streamID := uuid.New()
	meta := events.EventMetadata{ID: streamID}
	collector := &eventCollector{}
	ex := &citationsExtractor{name: "citations", dtype: "v1"}
	sink := structuredsink.NewFilteringSink(collector, structuredsink.Options{
		EmitRawDeltas:       true,
		EmitParsedSnapshots: true,
	}, ex)

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
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return initialModel(), nil
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
				return m, tick(m.speed)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
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
		case *events.EventFinal:
			m.filteredEvents = append(m.filteredEvents, "final")
		case *EventCitationStarted:
			m.citationEvents = append(m.citationEvents, fmt.Sprintf("started: %s", e.ItemID))
		case *EventCitationDelta:
			m.citationEvents = append(m.citationEvents, fmt.Sprintf("delta: %q", truncate(e.Delta, 30)))
		case *EventCitationUpdate:
			m.citationEvents = append(m.citationEvents, fmt.Sprintf("update: %d entries", len(e.Entries)))
		case *EventCitationCompleted:
			status := "success"
			if !e.Success {
				status = "failed"
			}
			m.citationEvents = append(m.citationEvents, fmt.Sprintf("completed: %s (%d entries)", status, len(e.Entries)))
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

	// Extract latest citation update
	for i := len(m.collector.list) - 1; i >= 0; i-- {
		ev := m.collector.list[i]
		if cu, ok := ev.(*EventCitationUpdate); ok {
			m.citations = cu.Entries
			m.citationErr = cu.Error
			break
		}
	}

	return m
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (m model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))

	title := titleStyle.Render("Citations Event Stream Demo")

	// Calculate box width for 3 columns
	boxWidth := (m.width / 3) - 3
	boxHeight := (m.height - 15) / 2 // Split height between content and events

	// --- Top Section: Content Streams ---
	contentSection := sectionStyle.Render("Content Streams")

	// Left: raw text stream
	rawTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("Raw Stream")
	rawContent := m.rawText
	if rawContent == "" {
		rawContent = "(waiting...)"
	}
	rawBox := boxStyle.Width(boxWidth).Height(boxHeight).Render(rawTitle + "\n\n" + rawContent)

	// Middle: filtered text
	filteredTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("Filtered Text")
	filteredContent := m.filteredText
	if filteredContent == "" {
		filteredContent = "(waiting...)"
	}
	filteredBox := boxStyle.Width(boxWidth).Height(boxHeight).Render(filteredTitle + "\n\n" + filteredContent)

	// Right: citations
	citationsTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("Citations Extracted")
	var citationsContent strings.Builder
	if len(m.citations) == 0 {
		citationsContent.WriteString("(no citations yet)")
	} else {
		for i, c := range m.citations {
			citationsContent.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
			citationsContent.WriteString(fmt.Sprintf("   Authors: %s\n", strings.Join(c.Authors, ", ")))
		}
	}
	if m.citationErr != "" {
		citationsContent.WriteString(fmt.Sprintf("\nError: %s", m.citationErr))
	}
	citationsBox := boxStyle.Width(boxWidth).Height(boxHeight).Render(citationsTitle + "\n\n" + citationsContent.String())

	contentRow := lipgloss.JoinHorizontal(lipgloss.Top, rawBox, filteredBox, citationsBox)

	// --- Bottom Section: Event Streams ---
	eventsSection := sectionStyle.Render("Event Streams")

	// Left: raw events
	rawEventsTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("Input Events")
	rawEventsContent := formatEventList(m.rawEvents, 5)
	rawEventsBox := boxStyle.Width(boxWidth).Height(boxHeight).Render(rawEventsTitle + "\n\n" + rawEventsContent)

	// Middle: filtered events
	filteredEventsTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("Filtered Events")
	filteredEventsContent := formatEventList(m.filteredEvents, 5)
	filteredEventsBox := boxStyle.Width(boxWidth).Height(boxHeight).Render(filteredEventsTitle + "\n\n" + filteredEventsContent)

	// Right: citation events
	citationEventsTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("Citation Events")
	citationEventsContent := formatEventList(m.citationEvents, 5)
	citationEventsBox := boxStyle.Width(boxWidth).Height(boxHeight).Render(citationEventsTitle + "\n\n" + citationEventsContent)

	eventsRow := lipgloss.JoinHorizontal(lipgloss.Top, rawEventsBox, filteredEventsBox, citationEventsBox)

	// Status
	status := fmt.Sprintf("Step %d/%d", m.currentIdx, len(m.parts))
	if m.autoPlay {
		status += " [AUTO-PLAY]"
	}
	status += fmt.Sprintf(" | Speed: %dms", m.speed.Milliseconds())

	// Help
	help := helpStyle.Render("space: toggle auto-play | n/→: step | +/-: speed | r: reset | q: quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
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
	if len(events) > maxRecent {
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

