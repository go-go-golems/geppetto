package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/serde"
	"github.com/rs/zerolog/log"
)

type ArtifactRun struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Timestamp int64  `json:"timestamp"`
	HasTurns  bool   `json:"hasTurns"`
	HasEvents bool   `json:"hasEvents"`
	HasLogs   bool   `json:"hasLogs"`
	HasRaw    bool   `json:"hasRaw"`
	TurnCount int    `json:"turnCount"`
}

type TurnDTO struct {
	ID              string     `json:"id"`
	Blocks          []BlockDTO `json:"blocks"`
	ExecutionIndex  int        `json:"executionIndex"`
	Label           string     `json:"label"`
	RawYAML         string     `json:"rawYaml,omitempty"`
	RawRequestIndex *int       `json:"rawRequestIndex,omitempty"` // Index into raw artifacts array
}

type BlockDTO struct {
	Kind    string                 `json:"kind"`
	Role    string                 `json:"role"`
	Payload map[string]interface{} `json:"payload"`
}

type ParsedRun struct {
	ID        string         `json:"id"`
	Path      string         `json:"path"`
	InputTurn *TurnDTO       `json:"inputTurn,omitempty"`
	Turns     []TurnDTO      `json:"turns"`
	Events    [][]Event      `json:"events"`
	Logs      []LogEntry     `json:"logs"`
	Raw       []RawArtifact  `json:"raw"`
	Errors    []ErrorContext `json:"errors"`
}

type Event struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

type LogEntry struct {
	Level   string                 `json:"level"`
	Time    string                 `json:"time"`
	Message string                 `json:"message"`
	Error   string                 `json:"error,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

type RawArtifact struct {
	TurnIndex       int              `json:"turnIndex"`
	InputTurnIndex  int              `json:"inputTurnIndex"`          // Which turn was used as input for this request
	InputTurnYAML   string           `json:"inputTurnYaml,omitempty"` // The actual turn YAML before conversion
	HTTPRequest     *HTTPRequest     `json:"httpRequest,omitempty"`
	HTTPResponse    *HTTPResponse    `json:"httpResponse,omitempty"`
	SSELog          string           `json:"sseLog,omitempty"`
	ProviderObjects []ProviderObject `json:"providerObjects"`
}

type HTTPRequest struct {
	TurnIndex int                 `json:"turn_index"`
	TurnID    string              `json:"turn_id"`
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body"`
}

type HTTPResponse struct {
	TurnIndex int                 `json:"turn_index"`
	TurnID    string              `json:"turn_id"`
	Status    int                 `json:"status"`
	Headers   map[string][]string `json:"headers"`
	Body      interface{}         `json:"body"`
}

type ProviderObject struct {
	Sequence int                    `json:"sequence"`
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data"`
}

type ErrorContext struct {
	TurnIndex     int           `json:"turnIndex"`
	Error         string        `json:"error"`
	RelatedLogs   []LogEntry    `json:"relatedLogs"`
	RelatedEvents []Event       `json:"relatedEvents"`
	HTTPRequest   *HTTPRequest  `json:"httpRequest,omitempty"`
	HTTPResponse  *HTTPResponse `json:"httpResponse,omitempty"`
}

type APIHandler struct {
	BaseDir string
}

func (h *APIHandler) GetRunsHandler(w http.ResponseWriter, r *http.Request) {
	runs, err := h.scanRuns()
	if err != nil {
		log.Error().Err(err).Msg("failed to scan runs")
		http.Error(w, "Failed to scan runs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(runs); err != nil {
		log.Error().Err(err).Msg("failed to encode runs response")
	}
}

func (h *APIHandler) GetRunHandler(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	if runID == "" {
		http.Error(w, "run ID required", http.StatusBadRequest)
		return
	}

	parsedRun, err := h.parseRun(runID)
	if err != nil {
		if errors.Is(err, ErrUnsafePath) {
			http.Error(w, "invalid run ID", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Str("runID", runID).Msg("failed to parse run")
		http.Error(w, fmt.Sprintf("Failed to parse run: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(parsedRun); err != nil {
		log.Error().Err(err).Str("runID", runID).Msg("failed to encode parsed run response")
	}
}

func (h *APIHandler) scanRuns() ([]ArtifactRun, error) {
	var runs []ArtifactRun

	if _, err := os.Stat(h.BaseDir); os.IsNotExist(err) {
		return runs, nil
	}

	entries, err := os.ReadDir(h.BaseDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		runPath := filepath.Join(h.BaseDir, entry.Name())
		run := h.analyzeRun(entry.Name(), runPath)
		if run.HasTurns || run.HasEvents || run.HasLogs {
			runs = append(runs, run)
		}
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Timestamp > runs[j].Timestamp
	})

	return runs, nil
}

func (h *APIHandler) analyzeRun(id, path string) ArtifactRun {
	run := ArtifactRun{
		ID:   id,
		Path: path,
	}

	if info, err := os.Stat(path); err == nil {
		run.Timestamp = info.ModTime().Unix()
	}

	if _, err := os.Stat(filepath.Join(path, "input_turn.yaml")); err == nil {
		run.HasTurns = true
	}

	if _, err := os.Stat(filepath.Join(path, "events.ndjson")); err == nil {
		run.HasEvents = true
	}

	if _, err := os.Stat(filepath.Join(path, "logs.jsonl")); err == nil {
		run.HasLogs = true
	}

	if _, err := os.Stat(filepath.Join(path, "raw")); err == nil {
		run.HasRaw = true
	}

	// Count turns
	turnCount := 0
	for i := 0; ; i++ {
		var path string
		if i == 0 {
			path = filepath.Join(run.Path, "final_turn.yaml")
		} else {
			path = filepath.Join(run.Path, fmt.Sprintf("final_turn_%d.yaml", i))
		}
		if _, err := os.Stat(path); err == nil {
			turnCount++
		} else {
			break
		}
	}
	run.TurnCount = turnCount

	return run
}

func toTurnDTO(turn *turns.Turn, index int, label string, rawYaml []byte, rawRequestIndex *int) TurnDTO {
	dto := TurnDTO{
		ID:              turn.ID,
		Blocks:          make([]BlockDTO, 0, len(turn.Blocks)),
		ExecutionIndex:  index,
		Label:           label,
		RawYAML:         string(rawYaml),
		RawRequestIndex: rawRequestIndex,
	}

	for _, block := range turn.Blocks {
		blockDTO := BlockDTO{
			Kind:    block.Kind.String(),
			Role:    string(block.Role),
			Payload: block.Payload,
		}

		// Ensure payload is not nil
		if blockDTO.Payload == nil {
			blockDTO.Payload = make(map[string]interface{})
		}

		dto.Blocks = append(dto.Blocks, blockDTO)
	}

	return dto
}

func (h *APIHandler) parseRun(runID string) (*ParsedRun, error) {
	runPath, err := secureJoinUnderBase(h.BaseDir, runID)
	if err != nil {
		return nil, err
	}
	runRoot, err := os.OpenRoot(runPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = runRoot.Close()
	}()

	parsed := &ParsedRun{
		ID:     runID,
		Path:   runPath,
		Turns:  []TurnDTO{},
		Events: [][]Event{},
		Logs:   []LogEntry{},
		Raw:    []RawArtifact{},
		Errors: []ErrorContext{},
	}

	// Parse input turn
	if inputData, err := runRoot.ReadFile("input_turn.yaml"); err == nil {
		if turn, err := serde.FromYAML(inputData); err == nil {
			dto := toTurnDTO(turn, -1, "Input Turn", inputData, nil)
			parsed.InputTurn = &dto
		}
	}

	// Parse turns
	apiCallIndex := 0 // Track which API call this corresponds to
	for i := 0; ; i++ {
		var path string
		var label string
		var rawRequestIndex *int

		if i == 0 {
			path = "final_turn.yaml"
			label = "After Initial Run"
			rawReqIdx := apiCallIndex
			rawRequestIndex = &rawReqIdx
			apiCallIndex++
		} else {
			path = fmt.Sprintf("final_turn_%d.yaml", i)
			if i%2 == 1 {
				label = fmt.Sprintf("After Follow-up #%d (before run)", (i+1)/2)
				// "before run" turns should show the NEXT API call that will be made
				rawReqIdx := apiCallIndex
				rawRequestIndex = &rawReqIdx
			} else {
				label = fmt.Sprintf("After Follow-up #%d Run", i/2)
				rawReqIdx := apiCallIndex
				rawRequestIndex = &rawReqIdx
				apiCallIndex++
			}
		}

		turnData, err := runRoot.ReadFile(path)
		if err != nil {
			break
		}

		turn, err := serde.FromYAML(turnData)
		if err != nil {
			log.Warn().Err(err).Int("index", i).Msg("failed to parse turn")
			continue
		}
		parsed.Turns = append(parsed.Turns, toTurnDTO(turn, i, label, turnData, rawRequestIndex))
	}

	// Parse events
	for i := 0; ; i++ {
		var path string
		if i == 0 {
			path = filepath.Join(runPath, "events.ndjson")
		} else {
			path = filepath.Join(runPath, fmt.Sprintf("events-%d.ndjson", i+1))
		}

		events, err := h.parseEvents(runPath, path)
		if err != nil {
			if i == 0 {
				log.Warn().Err(err).Msg("failed to parse events")
			}
			break
		}
		parsed.Events = append(parsed.Events, events)
	}

	// Parse logs
	logsPath := filepath.Join(runPath, "logs.jsonl")
	if logs, err := h.parseLogs(runPath, logsPath); err == nil {
		parsed.Logs = logs
	}

	// Parse raw artifacts
	rawDir := filepath.Join(runPath, "raw")
	if _, err := runRoot.Stat("raw"); err == nil {
		if raw, err := h.parseRawArtifacts(rawDir); err == nil {
			parsed.Raw = raw
		}
	}

	// Extract errors
	parsed.Errors = h.extractErrors(parsed)

	return parsed, nil
}

func (h *APIHandler) parseEvents(baseDir, path string) ([]Event, error) {
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return nil, err
	}
	data, _, err := readFileUnderBase(baseDir, relPath)
	if err != nil {
		return nil, err
	}

	var events []Event
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var raw map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		event := Event{
			Type:      getString(raw, "type"),
			Timestamp: int64(getFloat(raw, "ts")),
			Data:      getMap(raw, "event"),
		}

		if eventData, ok := raw["event"].(map[string]interface{}); ok {
			if meta, ok := eventData["meta"].(map[string]interface{}); ok {
				event.Meta = meta
			}
		}

		events = append(events, event)
	}

	return events, scanner.Err()
}

func (h *APIHandler) parseLogs(baseDir, path string) ([]LogEntry, error) {
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return nil, err
	}
	data, _, err := readFileUnderBase(baseDir, relPath)
	if err != nil {
		return nil, err
	}

	var logs []LogEntry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var raw map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		entry := LogEntry{
			Level:   getString(raw, "level"),
			Time:    getString(raw, "time"),
			Message: getString(raw, "message"),
			Error:   getString(raw, "error"),
			Extra:   make(map[string]interface{}),
		}

		// Collect extra fields
		for k, v := range raw {
			if k != "level" && k != "time" && k != "message" && k != "error" {
				entry.Extra[k] = v
			}
		}

		logs = append(logs, entry)
	}

	return logs, scanner.Err()
}

func (h *APIHandler) parseRawArtifacts(rawDir string) ([]RawArtifact, error) {
	artifacts := make(map[int]*RawArtifact)
	rawRoot, err := os.OpenRoot(rawDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rawRoot.Close()
	}()

	err = filepath.WalkDir(rawDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		name := d.Name()
		if !strings.HasPrefix(name, "turn-") {
			return nil
		}

		// Extract turn index
		var turnIdx int
		if _, err := fmt.Sscanf(name, "turn-%d-", &turnIdx); err != nil {
			return nil
		}

		if artifacts[turnIdx] == nil {
			artifacts[turnIdx] = &RawArtifact{
				TurnIndex:       turnIdx,
				ProviderObjects: []ProviderObject{},
			}
		}

		relPath, err := filepath.Rel(rawDir, path)
		if err != nil {
			return nil
		}
		data, err := rawRoot.ReadFile(relPath)
		if err != nil {
			return nil
		}

		switch {
		case strings.Contains(name, "http-request.json"):
			var req HTTPRequest
			if err := json.Unmarshal(data, &req); err == nil {
				artifacts[turnIdx].HTTPRequest = &req
			}
		case strings.Contains(name, "http-response.json"):
			var resp HTTPResponse
			if err := json.Unmarshal(data, &resp); err == nil {
				artifacts[turnIdx].HTTPResponse = &resp
			}
		case strings.Contains(name, "sse.log"):
			artifacts[turnIdx].SSELog = string(data)
		case strings.Contains(name, "input.yaml"):
			artifacts[turnIdx].InputTurnYAML = string(data)
		case strings.Contains(name, "provider-"):
			var obj map[string]interface{}
			if err := json.Unmarshal(data, &obj); err == nil {
				// Extract sequence and type from filename
				var seq int
				var typePart string
				if _, scanErr := fmt.Sscanf(name, "turn-%d-provider-%d-%s", &turnIdx, &seq, &typePart); scanErr != nil {
					log.Warn().Err(scanErr).Str("name", name).Msg("failed to parse provider artifact name")
					return nil
				}
				typePart = strings.TrimSuffix(typePart, ".json")

				artifacts[turnIdx].ProviderObjects = append(artifacts[turnIdx].ProviderObjects, ProviderObject{
					Sequence: seq,
					Type:     typePart,
					Data:     obj,
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to sorted slice
	var result []RawArtifact
	var indices []int
	for idx := range artifacts {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	for _, idx := range indices {
		art := artifacts[idx]
		// Sort provider objects by sequence
		sort.Slice(art.ProviderObjects, func(i, j int) bool {
			return art.ProviderObjects[i].Sequence < art.ProviderObjects[j].Sequence
		})

		// Calculate which turn execution index was used as input for this request
		// Raw artifacts are 1-indexed (turn-1, turn-2...), execution indices are 0-indexed
		// turn-1 uses execution index 0 (After Initial Run)
		// turn-2 uses execution index 1 (After Follow-up #1 before run)
		art.InputTurnIndex = idx - 1

		result = append(result, *art)
	}

	return result, nil
}

func (h *APIHandler) extractErrors(parsed *ParsedRun) []ErrorContext {
	var errors []ErrorContext

	// Find error logs
	for _, logEntry := range parsed.Logs {
		if logEntry.Level == "error" && logEntry.Error != "" {
			ctx := ErrorContext{
				Error:         logEntry.Error,
				RelatedLogs:   []LogEntry{logEntry},
				RelatedEvents: []Event{},
			}

			// Try to find turn index from log metadata
			if turnID, ok := logEntry.Extra["turn_id"].(string); ok {
				// Find related events
				for i, eventSet := range parsed.Events {
					for _, event := range eventSet {
						if event.Meta != nil {
							if eventTurnID, ok := event.Meta["turn_id"].(string); ok && eventTurnID == turnID {
								ctx.RelatedEvents = append(ctx.RelatedEvents, event)
								ctx.TurnIndex = i
							}
						}
					}
				}

				// Find HTTP data
				for _, raw := range parsed.Raw {
					if raw.TurnIndex == ctx.TurnIndex+1 {
						ctx.HTTPRequest = raw.HTTPRequest
						ctx.HTTPResponse = raw.HTTPResponse
						break
					}
				}
			}

			errors = append(errors, ctx)
		}
	}

	return errors
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return make(map[string]interface{})
}
