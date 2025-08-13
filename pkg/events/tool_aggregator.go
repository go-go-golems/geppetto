package events

// ToolEventEntry aggregates tool activity across provider tool-call, local execution, and results.
// It is keyed by the tool call ID.
type ToolEventEntry struct {
	ID             string
	Name           string
	Input          string
	ProviderCalled bool
	ExecStarted    bool
	Result         string
}

// ToolEventAggregator collects tool-related events into compact entries per tool call ID.
type ToolEventAggregator struct {
	index   map[string]int
	entries []ToolEventEntry
}

// NewToolEventAggregator creates a new aggregator.
func NewToolEventAggregator() *ToolEventAggregator {
	return &ToolEventAggregator{
		index:   make(map[string]int),
		entries: make([]ToolEventEntry, 0, 4),
	}
}

// Reset clears the aggregator state.
func (a *ToolEventAggregator) Reset() {
	a.index = make(map[string]int)
	a.entries = a.entries[:0]
}

// Entries returns a snapshot of current entries in insertion order.
func (a *ToolEventAggregator) Entries() []ToolEventEntry {
	// Return a shallow copy to avoid external mutation
	out := make([]ToolEventEntry, len(a.entries))
	copy(out, a.entries)
	return out
}

// Handle consumes an Event and updates entries when it is tool-related.
func (a *ToolEventAggregator) Handle(e Event) {
	switch ev := e.(type) {
	case *EventToolCall:
		if ev.ToolCall.ID == "" {
			return
		}
		idx := a.ensure(ev.ToolCall.ID)
		a.entries[idx].ProviderCalled = true
		a.entries[idx].Name = ev.ToolCall.Name
		if ev.ToolCall.Input != "" {
			a.entries[idx].Input = ev.ToolCall.Input
		}
	case *EventToolCallExecute:
		if ev.ToolCall.ID == "" {
			return
		}
		idx := a.ensure(ev.ToolCall.ID)
		a.entries[idx].ExecStarted = true
		if ev.ToolCall.Name != "" {
			a.entries[idx].Name = ev.ToolCall.Name
		}
		if ev.ToolCall.Input != "" && a.entries[idx].Input == "" {
			a.entries[idx].Input = ev.ToolCall.Input
		}
	case *EventToolResult:
		if ev.ToolResult.ID == "" {
			return
		}
		idx := a.ensure(ev.ToolResult.ID)
		a.entries[idx].Result = ev.ToolResult.Result
	case *EventToolCallExecutionResult:
		if ev.ToolResult.ID == "" {
			return
		}
		idx := a.ensure(ev.ToolResult.ID)
		a.entries[idx].Result = ev.ToolResult.Result
	}
}

// Lines returns a compact, plain-text representation for each entry.
// UI layers can style these strings as needed.
func (a *ToolEventAggregator) Lines() []string {
	lines := make([]string, 0, len(a.entries))
	for _, e := range a.entries {
		name := e.Name
		if name == "" {
			name = e.ID
		}
		parts := make([]string, 0, 4)
		if e.ProviderCalled {
			parts = append(parts, "→ "+name)
		}
		if e.ExecStarted {
			parts = append(parts, "↳ exec")
		}
		if e.Result != "" {
			parts = append(parts, "← "+e.Result)
		}
		if e.Input != "" {
			parts = append(parts, e.Input)
		}
		lines = append(lines, joinWithDoubleSpace(parts))
	}
	return lines
}

func (a *ToolEventAggregator) ensure(id string) int {
	if idx, ok := a.index[id]; ok {
		return idx
	}
	idx := len(a.entries)
	a.index[id] = idx
	a.entries = append(a.entries, ToolEventEntry{ID: id})
	return idx
}

func joinWithDoubleSpace(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	totalLen := 0
	for _, p := range parts {
		totalLen += len(p)
	}
	// preallocate with some extra for separators
	b := make([]byte, 0, totalLen+2*(len(parts)-1))
	for i, p := range parts {
		if i > 0 {
			b = append(b, ' ', ' ')
		}
		b = append(b, p...)
	}
	return string(b)
}
