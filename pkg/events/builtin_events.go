package events

// --- Web search custom events ---

type EventWebSearchStarted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	Query  string `json:"query,omitempty"`
}

func NewWebSearchStarted(metadata EventMetadata, itemID, query string) *EventWebSearchStarted {
	return &EventWebSearchStarted{EventImpl: EventImpl{Type_: EventTypeWebSearchStarted, Metadata_: metadata}, ItemID: itemID, Query: query}
}

type EventWebSearchSearching struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewWebSearchSearching(metadata EventMetadata, itemID string) *EventWebSearchSearching {
	return &EventWebSearchSearching{EventImpl: EventImpl{Type_: EventTypeWebSearchSearching, Metadata_: metadata}, ItemID: itemID}
}

type EventWebSearchOpenPage struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	URL    string `json:"url,omitempty"`
}

func NewWebSearchOpenPage(metadata EventMetadata, itemID, url string) *EventWebSearchOpenPage {
	return &EventWebSearchOpenPage{EventImpl: EventImpl{Type_: EventTypeWebSearchOpenPage, Metadata_: metadata}, ItemID: itemID, URL: url}
}

type EventWebSearchDone struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewWebSearchDone(metadata EventMetadata, itemID string) *EventWebSearchDone {
	return &EventWebSearchDone{EventImpl: EventImpl{Type_: EventTypeWebSearchDone, Metadata_: metadata}, ItemID: itemID}
}

// Citation event attached to streamed output text
type EventCitation struct {
	EventImpl
	Title           string `json:"title,omitempty"`
	URL             string `json:"url,omitempty"`
	StartIndex      *int   `json:"start_index,omitempty"`
	EndIndex        *int   `json:"end_index,omitempty"`
	OutputIndex     *int   `json:"output_index,omitempty"`
	ContentIndex    *int   `json:"content_index,omitempty"`
	AnnotationIndex *int   `json:"annotation_index,omitempty"`
}

func NewCitation(metadata EventMetadata, title, url string, start, end, outputIdx, contentIdx, annIdx *int) *EventCitation {
	return &EventCitation{EventImpl: EventImpl{Type_: EventTypeCitation, Metadata_: metadata}, Title: title, URL: url, StartIndex: start, EndIndex: end, OutputIndex: outputIdx, ContentIndex: contentIdx, AnnotationIndex: annIdx}
}

// File search custom events
type EventFileSearchStarted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewFileSearchStarted(metadata EventMetadata, itemID string) *EventFileSearchStarted {
	return &EventFileSearchStarted{EventImpl: EventImpl{Type_: EventTypeFileSearchStarted, Metadata_: metadata}, ItemID: itemID}
}

type EventFileSearchSearching struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewFileSearchSearching(metadata EventMetadata, itemID string) *EventFileSearchSearching {
	return &EventFileSearchSearching{EventImpl: EventImpl{Type_: EventTypeFileSearchSearching, Metadata_: metadata}, ItemID: itemID}
}

type EventFileSearchDone struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewFileSearchDone(metadata EventMetadata, itemID string) *EventFileSearchDone {
	return &EventFileSearchDone{EventImpl: EventImpl{Type_: EventTypeFileSearchDone, Metadata_: metadata}, ItemID: itemID}
}

// Code interpreter custom events
type EventCodeInterpreterStarted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewCodeInterpreterStarted(metadata EventMetadata, itemID string) *EventCodeInterpreterStarted {
	return &EventCodeInterpreterStarted{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterStarted, Metadata_: metadata}, ItemID: itemID}
}

type EventCodeInterpreterInterpreting struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewCodeInterpreterInterpreting(metadata EventMetadata, itemID string) *EventCodeInterpreterInterpreting {
	return &EventCodeInterpreterInterpreting{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterInterpreting, Metadata_: metadata}, ItemID: itemID}
}

type EventCodeInterpreterDone struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewCodeInterpreterDone(metadata EventMetadata, itemID string) *EventCodeInterpreterDone {
	return &EventCodeInterpreterDone{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterDone, Metadata_: metadata}, ItemID: itemID}
}

type EventCodeInterpreterCodeDelta struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	Delta  string `json:"delta"`
}

func NewCodeInterpreterCodeDelta(metadata EventMetadata, itemID, delta string) *EventCodeInterpreterCodeDelta {
	return &EventCodeInterpreterCodeDelta{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterCodeDelta, Metadata_: metadata}, ItemID: itemID, Delta: delta}
}

type EventCodeInterpreterCodeDone struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	Code   string `json:"code"`
}

func NewCodeInterpreterCodeDone(metadata EventMetadata, itemID, code string) *EventCodeInterpreterCodeDone {
	return &EventCodeInterpreterCodeDone{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterCodeDone, Metadata_: metadata}, ItemID: itemID, Code: code}
}

// MCP
type EventMCPArgsDelta struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	Delta  string `json:"delta"`
}

func NewMCPArgsDelta(metadata EventMetadata, itemID, delta string) *EventMCPArgsDelta {
	return &EventMCPArgsDelta{EventImpl: EventImpl{Type_: EventTypeMCPArgsDelta, Metadata_: metadata}, ItemID: itemID, Delta: delta}
}

type EventMCPArgsDone struct {
	EventImpl
	ItemID    string `json:"item_id,omitempty"`
	Arguments string `json:"arguments"`
}

func NewMCPArgsDone(metadata EventMetadata, itemID, args string) *EventMCPArgsDone {
	return &EventMCPArgsDone{EventImpl: EventImpl{Type_: EventTypeMCPArgsDone, Metadata_: metadata}, ItemID: itemID, Arguments: args}
}

type EventMCPInProgress struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPInProgress(metadata EventMetadata, itemID string) *EventMCPInProgress {
	return &EventMCPInProgress{EventImpl: EventImpl{Type_: EventTypeMCPInProgress, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPCompleted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPCompleted(metadata EventMetadata, itemID string) *EventMCPCompleted {
	return &EventMCPCompleted{EventImpl: EventImpl{Type_: EventTypeMCPCompleted, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPFailed struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPFailed(metadata EventMetadata, itemID string) *EventMCPFailed {
	return &EventMCPFailed{EventImpl: EventImpl{Type_: EventTypeMCPFailed, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPListInProgress struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPListInProgress(metadata EventMetadata, itemID string) *EventMCPListInProgress {
	return &EventMCPListInProgress{EventImpl: EventImpl{Type_: EventTypeMCPListInProgress, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPListCompleted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPListCompleted(metadata EventMetadata, itemID string) *EventMCPListCompleted {
	return &EventMCPListCompleted{EventImpl: EventImpl{Type_: EventTypeMCPListCompleted, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPListFailed struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPListFailed(metadata EventMetadata, itemID string) *EventMCPListFailed {
	return &EventMCPListFailed{EventImpl: EventImpl{Type_: EventTypeMCPListFailed, Metadata_: metadata}, ItemID: itemID}
}

// Image generation
type EventImageGenInProgress struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewImageGenInProgress(metadata EventMetadata, itemID string) *EventImageGenInProgress {
	return &EventImageGenInProgress{EventImpl: EventImpl{Type_: EventTypeImageGenInProgress, Metadata_: metadata}, ItemID: itemID}
}

type EventImageGenGenerating struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewImageGenGenerating(metadata EventMetadata, itemID string) *EventImageGenGenerating {
	return &EventImageGenGenerating{EventImpl: EventImpl{Type_: EventTypeImageGenGenerating, Metadata_: metadata}, ItemID: itemID}
}

type EventImageGenPartialImage struct {
	EventImpl
	ItemID             string `json:"item_id,omitempty"`
	PartialImageBase64 string `json:"partial_image_base64,omitempty"`
}

func NewImageGenPartialImage(metadata EventMetadata, itemID, b64 string) *EventImageGenPartialImage {
	return &EventImageGenPartialImage{EventImpl: EventImpl{Type_: EventTypeImageGenPartialImage, Metadata_: metadata}, ItemID: itemID, PartialImageBase64: b64}
}

type EventImageGenCompleted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewImageGenCompleted(metadata EventMetadata, itemID string) *EventImageGenCompleted {
	return &EventImageGenCompleted{EventImpl: EventImpl{Type_: EventTypeImageGenCompleted, Metadata_: metadata}, ItemID: itemID}
}

// Normalized results
type SearchResult struct {
	URL        string         `json:"url,omitempty"`
	Title      string         `json:"title,omitempty"`
	Snippet    string         `json:"snippet,omitempty"`
	Extensions map[string]any `json:"ext,omitempty"`
}
type EventToolSearchResults struct {
	EventImpl
	Tool    string         `json:"tool"`
	ItemID  string         `json:"item_id,omitempty"`
	Results []SearchResult `json:"results"`
}

func NewToolSearchResults(metadata EventMetadata, tool, itemID string, res []SearchResult) *EventToolSearchResults {
	return &EventToolSearchResults{EventImpl: EventImpl{Type_: EventTypeToolSearchResults, Metadata_: metadata}, Tool: tool, ItemID: itemID, Results: res}
}
