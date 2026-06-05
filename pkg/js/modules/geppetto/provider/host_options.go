package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettomodule "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const HostOptionsServiceKey = "geppetto.provider.host-options.v1"

type HostOptionsContribution struct {
	ToolRegistry        tools.ToolRegistry
	MiddlewareFactories map[string]geppettomodule.MiddlewareFactory
	DefaultEventSinks   []events.EventSink
	Configure           func(context.Context, Config, *geppettomodule.Options) error
}

type HostOptionsContributionOption func(*HostOptionsContribution)

func NewHostOptionsContribution(options ...HostOptionsContributionOption) HostOptionsContribution {
	ret := HostOptionsContribution{}
	for _, option := range options {
		if option != nil {
			option(&ret)
		}
	}
	return ret
}

func WithToolRegistry(registry tools.ToolRegistry) HostOptionsContributionOption {
	return func(c *HostOptionsContribution) { c.ToolRegistry = registry }
}

func WithMiddlewareFactory(name string, factory geppettomodule.MiddlewareFactory) HostOptionsContributionOption {
	return func(c *HostOptionsContribution) {
		name = strings.TrimSpace(name)
		if name == "" || factory == nil {
			return
		}
		if c.MiddlewareFactories == nil {
			c.MiddlewareFactories = map[string]geppettomodule.MiddlewareFactory{}
		}
		c.MiddlewareFactories[name] = factory
	}
}

func WithDefaultEventSink(sink events.EventSink) HostOptionsContributionOption {
	return func(c *HostOptionsContribution) {
		if sink != nil {
			c.DefaultEventSinks = append(c.DefaultEventSinks, sink)
		}
	}
}

func WithOptionsConfigurator(configure func(context.Context, Config, *geppettomodule.Options) error) HostOptionsContributionOption {
	return func(c *HostOptionsContribution) { c.Configure = configure }
}

func applyHostOptionsContributions(ctx context.Context, host providerapi.HostServices, cfg Config, opts *geppettomodule.Options) error {
	if opts == nil || host == nil {
		return nil
	}
	lookup, ok := host.(providerapi.HostServiceLookup)
	if !ok || lookup == nil {
		return nil
	}
	for i, raw := range lookup.HostServiceValues(HostOptionsServiceKey) {
		contribution, err := normalizeHostOptionsContribution(raw)
		if err != nil {
			return fmt.Errorf("geppetto host options contribution %d: %w", i, err)
		}
		if err := applyHostOptionsContribution(ctx, cfg, opts, contribution); err != nil {
			return fmt.Errorf("geppetto host options contribution %d: %w", i, err)
		}
	}
	return nil
}

func normalizeHostOptionsContribution(raw any) (HostOptionsContribution, error) {
	switch x := raw.(type) {
	case HostOptionsContribution:
		return x, nil
	case *HostOptionsContribution:
		if x == nil {
			return HostOptionsContribution{}, fmt.Errorf("nil contribution")
		}
		return *x, nil
	default:
		return HostOptionsContribution{}, fmt.Errorf("expected HostOptionsContribution, got %T", raw)
	}
}

func applyHostOptionsContribution(ctx context.Context, cfg Config, opts *geppettomodule.Options, contribution HostOptionsContribution) error {
	if contribution.ToolRegistry != nil {
		merged, err := mergeToolRegistriesStrict(opts.GoToolRegistry, contribution.ToolRegistry)
		if err != nil {
			return err
		}
		opts.GoToolRegistry = merged
	}
	if len(contribution.MiddlewareFactories) > 0 {
		if opts.GoMiddlewareFactories == nil {
			opts.GoMiddlewareFactories = map[string]geppettomodule.MiddlewareFactory{}
		}
		for name, factory := range contribution.MiddlewareFactories {
			name = strings.TrimSpace(name)
			if name == "" || factory == nil {
				continue
			}
			if _, ok := opts.GoMiddlewareFactories[name]; ok {
				return fmt.Errorf("duplicate Geppetto middleware factory %q", name)
			}
			opts.GoMiddlewareFactories[name] = factory
		}
	}
	if len(contribution.DefaultEventSinks) > 0 {
		opts.DefaultEventSinks = append(opts.DefaultEventSinks, contribution.DefaultEventSinks...)
	}
	if contribution.Configure != nil {
		if err := contribution.Configure(ctx, cfg, opts); err != nil {
			return err
		}
	}
	return nil
}

func mergeToolRegistriesStrict(existing, incoming tools.ToolRegistry) (tools.ToolRegistry, error) {
	if existing == nil {
		return incoming, nil
	}
	if incoming == nil {
		return existing, nil
	}
	out := tools.NewInMemoryToolRegistry()
	seen := map[string]struct{}{}
	for _, tool := range existing.ListTools() {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		if err := out.RegisterTool(name, tool); err != nil {
			return nil, err
		}
		seen[name] = struct{}{}
	}
	for _, tool := range incoming.ListTools() {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("duplicate Geppetto tool %q", name)
		}
		if err := out.RegisterTool(name, tool); err != nil {
			return nil, err
		}
	}
	return out, nil
}
