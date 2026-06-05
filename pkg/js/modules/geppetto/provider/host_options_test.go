package provider

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettomodule "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestApplyHostOptionsContributionsMergesToolsMiddlewareAndSinks(t *testing.T) {
	registry := tools.NewInMemoryToolRegistry()
	tool, err := tools.NewToolFromFunc("wordCount", "Count words", func(in struct {
		Text string `json:"text"`
	}) (map[string]int, error) {
		return map[string]int{"count": len(in.Text)}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := registry.RegisterTool("wordCount", *tool); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}
	sink := &recordingEventSink{}
	host := hostServicesForTest{values: map[string][]any{HostOptionsServiceKey: {
		NewHostOptionsContribution(
			WithToolRegistry(registry),
			WithMiddlewareFactory("addSystemPrompt", func(options map[string]any) (middleware.Middleware, error) {
				return middleware.NewSystemPromptMiddleware("demo"), nil
			}),
			WithDefaultEventSink(sink),
		),
	}}}
	opts := geppettomodule.Options{}
	if err := applyHostOptionsContributions(context.Background(), host, Config{}, &opts); err != nil {
		t.Fatalf("applyHostOptionsContributions: %v", err)
	}
	if opts.GoToolRegistry == nil {
		t.Fatalf("GoToolRegistry missing")
	}
	if _, err := opts.GoToolRegistry.GetTool("wordCount"); err != nil {
		t.Fatalf("wordCount missing: %v", err)
	}
	if opts.GoMiddlewareFactories["addSystemPrompt"] == nil {
		t.Fatalf("middleware factory missing")
	}
	if len(opts.DefaultEventSinks) != 1 || opts.DefaultEventSinks[0] != sink {
		t.Fatalf("event sinks = %#v", opts.DefaultEventSinks)
	}
}

func TestApplyHostOptionsContributionsRejectsDuplicateTools(t *testing.T) {
	first := tools.NewInMemoryToolRegistry()
	second := tools.NewInMemoryToolRegistry()
	for _, registry := range []*tools.InMemoryToolRegistry{first, second} {
		tool, err := tools.NewToolFromFunc("wordCount", "Count words", func() (string, error) { return "ok", nil })
		if err != nil {
			t.Fatalf("NewToolFromFunc: %v", err)
		}
		if err := registry.RegisterTool("wordCount", *tool); err != nil {
			t.Fatalf("RegisterTool: %v", err)
		}
	}
	host := hostServicesForTest{values: map[string][]any{HostOptionsServiceKey: {
		NewHostOptionsContribution(WithToolRegistry(first)),
		NewHostOptionsContribution(WithToolRegistry(second)),
	}}}
	err := applyHostOptionsContributions(context.Background(), host, Config{}, &geppettomodule.Options{})
	if err == nil {
		t.Fatalf("expected duplicate tool error")
	}
}

type hostServicesForTest struct {
	values map[string][]any
}

func (h hostServicesForTest) AssetResolver() providerapi.AssetResolver { return nil }
func (h hostServicesForTest) HostService(key string) (any, bool) {
	values := h.HostServiceValues(key)
	if len(values) == 0 {
		return nil, false
	}
	if len(values) == 1 {
		return values[0], true
	}
	return values, true
}
func (h hostServicesForTest) HostServiceValues(key string) []any {
	return append([]any(nil), h.values[key]...)
}

type recordingEventSink struct{ events []events.Event }

func (s *recordingEventSink) PublishEvent(event events.Event) error {
	s.events = append(s.events, event)
	return nil
}
