package hostservicesexample

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettoprovider "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto/provider"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "geppetto-host-services-example"
const sectionSlug = "geppetto-host-services"

func Register(registry *providerapi.ProviderRegistry) error {
	capability := capability{}
	return registry.Package(PackageID,
		providerapi.Module{
			Name:        "host-services",
			DefaultAs:   "geppetto-host-services",
			Description: "Contributes example Geppetto host services for generated xgoja demos.",
			NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				return func(vm *goja.Runtime, module *goja.Object) {
					exports := module.Get("exports").(*goja.Object)
					_ = exports.Set("version", "0.1.0")
				}, nil
			},
		},
		providerapi.WithPackageCapability(capability),
	)
}

type capability struct{}

func (capability) CapabilityID() string { return "geppetto-host-services-example" }

func (capability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
	section, err := schema.NewSection(sectionSlug, "Geppetto host services example",
		schema.WithFields(
			fields.New("event-log", fields.TypeString, fields.WithHelp("JSONL file that receives Geppetto inference events")),
		),
	)
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

func (capability) ContributeHostServices(ctx context.Context, req providerapi.HostServiceContributionRequest, sink providerapi.HostServiceSink) error {
	registry, err := exampleToolRegistry()
	if err != nil {
		return err
	}
	contribution := geppettoprovider.NewHostOptionsContribution(
		geppettoprovider.WithToolRegistry(registry),
		geppettoprovider.WithMiddlewareFactory("addSystemPrompt", addSystemPromptFactory),
	)
	if eventLog := eventLogPath(req.Values); eventLog != "" {
		eventSink, err := newJSONLEventSink(eventLog)
		if err != nil {
			return err
		}
		contribution.DefaultEventSinks = append(contribution.DefaultEventSinks, eventSink)
	}
	return sink.AddHostService(geppettoprovider.HostOptionsServiceKey, contribution)
}

func eventLogPath(vals *values.Values) string {
	if vals == nil {
		return ""
	}
	field, ok := vals.GetField(sectionSlug, "event-log")
	if !ok || field == nil || field.Value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(field.Value))
}

type wordCountInput struct {
	Text string `json:"text"`
}

func exampleToolRegistry() (tools.ToolRegistry, error) {
	registry := tools.NewInMemoryToolRegistry()
	tool, err := tools.NewToolFromFunc("wordCount", "Count whitespace-separated words", func(_ context.Context, input wordCountInput) (map[string]any, error) {
		return map[string]any{"count": len(strings.Fields(input.Text))}, nil
	})
	if err != nil {
		return nil, err
	}
	if err := registry.RegisterTool("wordCount", *tool); err != nil {
		return nil, err
	}
	return registry, nil
}

func addSystemPromptFactory(options map[string]any) (middleware.Middleware, error) {
	prompt := "Answer briefly."
	if raw, ok := options["prompt"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(raw)); s != "" {
			prompt = s
		}
	}
	return middleware.NewSystemPromptMiddleware(prompt), nil
}

type jsonlEventSink struct {
	mu     sync.Mutex
	file   *os.File
	writer *bufio.Writer
}

func newJSONLEventSink(path string) (*jsonlEventSink, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("event log path is empty")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &jsonlEventSink{file: file, writer: bufio.NewWriter(file)}, nil
}

func (s *jsonlEventSink) PublishEvent(ev events.Event) error {
	if s == nil || ev == nil || s.writer == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	payload := map[string]any{
		"type": string(ev.Type()),
		"meta": ev.Metadata(),
	}
	if raw := ev.Payload(); len(raw) > 0 {
		payload["payload"] = json.RawMessage(raw)
	}
	if err := json.NewEncoder(s.writer).Encode(payload); err != nil {
		return err
	}
	return s.writer.Flush()
}

func (s *jsonlEventSink) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer != nil {
		if err := s.writer.Flush(); err != nil {
			return err
		}
	}
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
