package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"gopkg.in/yaml.v3"
)

type ProfileIntrospectionSettings = geppettosections.ProfileIntrospectionSettings

type ProfileRegistryReportOptions struct {
	IncludeResolution     bool
	IncludeMergedSettings bool
	RedactSecrets         bool
}

type ProfileRegistryReportInput struct {
	SourceEntries       []string
	Registry            gepprofiles.Registry
	DefaultRegistrySlug gepprofiles.RegistrySlug
	DefaultProfileSlug  gepprofiles.EngineProfileSlug
	ResolveInput        gepprofiles.ResolveInput
}

type ProfileRegistryReport struct {
	Sources          []ProfileRegistrySourceReport  `json:"sources,omitempty" yaml:"sources,omitempty"`
	DefaultRegistry  string                         `json:"default_registry,omitempty" yaml:"default_registry,omitempty"`
	DefaultProfile   string                         `json:"default_profile,omitempty" yaml:"default_profile,omitempty"`
	SelectedRegistry string                         `json:"selected_registry,omitempty" yaml:"selected_registry,omitempty"`
	SelectedProfile  string                         `json:"selected_profile,omitempty" yaml:"selected_profile,omitempty"`
	Registries       []ProfileRegistrySummaryReport `json:"registries,omitempty" yaml:"registries,omitempty"`
	Profiles         []ProfileSummaryReport         `json:"profiles,omitempty" yaml:"profiles,omitempty"`
	Resolution       *ProfileResolutionReport       `json:"resolution,omitempty" yaml:"resolution,omitempty"`
}

type ProfileRegistrySourceReport struct {
	Raw  string `json:"raw" yaml:"raw"`
	Kind string `json:"kind" yaml:"kind"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	DSN  string `json:"dsn,omitempty" yaml:"dsn,omitempty"`
}

type ProfileRegistrySummaryReport struct {
	Slug           string `json:"slug" yaml:"slug"`
	DisplayName    string `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description    string `json:"description,omitempty" yaml:"description,omitempty"`
	DefaultProfile string `json:"default_profile,omitempty" yaml:"default_profile,omitempty"`
	ProfileCount   int    `json:"profile_count" yaml:"profile_count"`
	IsDefault      bool   `json:"is_default,omitempty" yaml:"is_default,omitempty"`
}

type ProfileSummaryReport struct {
	Registry    string `json:"registry" yaml:"registry"`
	Slug        string `json:"slug" yaml:"slug"`
	DisplayName string `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Model       string `json:"model,omitempty" yaml:"model,omitempty"`
	APIType     string `json:"api_type,omitempty" yaml:"api_type,omitempty"`
	Version     uint64 `json:"version,omitempty" yaml:"version,omitempty"`
	Source      string `json:"source,omitempty" yaml:"source,omitempty"`
	IsDefault   bool   `json:"is_default,omitempty" yaml:"is_default,omitempty"`
	IsSelected  bool   `json:"is_selected,omitempty" yaml:"is_selected,omitempty"`
}

type ProfileResolutionReport struct {
	Registry          string                                  `json:"registry" yaml:"registry"`
	Profile           string                                  `json:"profile" yaml:"profile"`
	Lineage           []gepprofiles.ResolvedProfileStackEntry `json:"lineage,omitempty" yaml:"lineage,omitempty"`
	Metadata          map[string]any                          `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	InferenceSettings map[string]any                          `json:"inference_settings,omitempty" yaml:"inference_settings,omitempty"`
}

func ResolveProfileIntrospectionSettings(parsed *values.Values) ProfileIntrospectionSettings {
	ret := ProfileIntrospectionSettings{}
	if parsed != nil {
		_ = parsed.DecodeSectionInto(geppettosections.ProfileIntrospectionSectionSlug, &ret)
	}
	ret.ProfileOutput = strings.TrimSpace(ret.ProfileOutput)
	if ret.ProfileOutput == "" {
		ret.ProfileOutput = "text"
	}
	return ret
}

func BuildProfileRegistryReport(ctx context.Context, cfg AppBootstrapConfig, parsed *values.Values, opts ProfileRegistryReportOptions) (*ProfileRegistryReport, func(), error) {
	runtime, err := ResolveCLIProfileRuntime(ctx, cfg, parsed)
	if err != nil {
		return nil, nil, err
	}
	cleanup := runtime.Close
	chain := runtime.ProfileRegistryChain
	if chain == nil {
		return &ProfileRegistryReport{}, cleanup, nil
	}
	report, err := BuildProfileRegistryReportFromRegistry(ctx, ProfileRegistryReportInput{
		SourceEntries:       runtime.ProfileSettings.ProfileRegistries,
		Registry:            chain.Registry,
		DefaultRegistrySlug: chain.DefaultRegistrySlug,
		ResolveInput:        chain.DefaultProfileResolve,
	}, opts)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	return report, cleanup, nil
}

func BuildProfileRegistryReportFromRegistry(ctx context.Context, in ProfileRegistryReportInput, opts ProfileRegistryReportOptions) (*ProfileRegistryReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if in.Registry == nil {
		return &ProfileRegistryReport{Sources: sourceReports(in.SourceEntries)}, nil
	}
	if !opts.RedactSecrets {
		opts.RedactSecrets = true
	}

	report := &ProfileRegistryReport{
		Sources:         sourceReports(in.SourceEntries),
		DefaultRegistry: in.DefaultRegistrySlug.String(),
	}
	if !in.DefaultProfileSlug.IsZero() {
		report.DefaultProfile = in.DefaultProfileSlug.String()
	}
	if !in.ResolveInput.RegistrySlug.IsZero() {
		report.SelectedRegistry = in.ResolveInput.RegistrySlug.String()
	}
	if !in.ResolveInput.EngineProfileSlug.IsZero() {
		report.SelectedProfile = in.ResolveInput.EngineProfileSlug.String()
	}

	registries, err := in.Registry.ListRegistries(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(registries, func(i, j int) bool { return registries[i].Slug < registries[j].Slug })
	for _, summary := range registries {
		reg, err := in.Registry.GetRegistry(ctx, summary.Slug)
		if err != nil {
			return nil, err
		}
		defaultProfile := summary.DefaultEngineProfileSlug
		if defaultProfile.IsZero() && reg != nil {
			defaultProfile = reg.DefaultEngineProfileSlug
		}
		description := ""
		if reg != nil {
			description = strings.TrimSpace(reg.Description)
		}
		report.Registries = append(report.Registries, ProfileRegistrySummaryReport{
			Slug:           summary.Slug.String(),
			DisplayName:    strings.TrimSpace(summary.DisplayName),
			Description:    description,
			DefaultProfile: defaultProfile.String(),
			ProfileCount:   summary.EngineProfileCount,
			IsDefault:      summary.Slug == in.DefaultRegistrySlug,
		})

		profiles, err := in.Registry.ListEngineProfiles(ctx, summary.Slug)
		if err != nil {
			return nil, err
		}
		sort.Slice(profiles, func(i, j int) bool {
			if profiles[i] == nil {
				return false
			}
			if profiles[j] == nil {
				return true
			}
			return profiles[i].Slug < profiles[j].Slug
		})
		for _, profile := range profiles {
			if profile == nil {
				continue
			}
			model, apiType := profileModelAndAPIType(profile)
			report.Profiles = append(report.Profiles, ProfileSummaryReport{
				Registry:    summary.Slug.String(),
				Slug:        profile.Slug.String(),
				DisplayName: strings.TrimSpace(profile.DisplayName),
				Description: strings.TrimSpace(profile.Description),
				Model:       model,
				APIType:     apiType,
				Version:     profile.Metadata.Version,
				Source:      strings.TrimSpace(profile.Metadata.Source),
				IsDefault:   profile.Slug == defaultProfile,
				IsSelected:  isSelectedProfile(summary.Slug, profile.Slug, in.ResolveInput),
			})
		}
	}

	if opts.IncludeResolution {
		resolveInput := in.ResolveInput
		if resolveInput.RegistrySlug.IsZero() {
			resolveInput.RegistrySlug = in.DefaultRegistrySlug
		}
		if resolveInput.EngineProfileSlug.IsZero() && !in.DefaultProfileSlug.IsZero() {
			resolveInput.EngineProfileSlug = in.DefaultProfileSlug
		}
		resolved, err := in.Registry.ResolveEngineProfile(ctx, resolveInput)
		if err != nil {
			return nil, err
		}
		report.SelectedRegistry = resolved.RegistrySlug.String()
		report.SelectedProfile = resolved.EngineProfileSlug.String()
		report.Resolution = &ProfileResolutionReport{
			Registry: resolved.RegistrySlug.String(),
			Profile:  resolved.EngineProfileSlug.String(),
			Lineage:  append([]gepprofiles.ResolvedProfileStackEntry(nil), resolved.StackLineage...),
			Metadata: RedactProfileSecrets(resolved.Metadata).(map[string]any),
		}
		if opts.IncludeMergedSettings && resolved.InferenceSettings != nil {
			settingsMap, err := inferenceSettingsMap(resolved.InferenceSettings)
			if err != nil {
				return nil, err
			}
			report.Resolution.InferenceSettings = RedactProfileSecrets(settingsMap).(map[string]any)
		}
	}

	return report, nil
}

func RenderProfileRegistryReport(w io.Writer, report *ProfileRegistryReport, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text", "table":
		return RenderProfileRegistryReportText(w, report)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	case "yaml", "yml":
		b, err := yaml.Marshal(report)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	default:
		return fmt.Errorf("unsupported profile output format %q", format)
	}
}

func RenderProfileRegistryReportText(w io.Writer, report *ProfileRegistryReport) error {
	if report == nil {
		_, err := fmt.Fprintln(w, "No profile report available.")
		return err
	}
	if _, err := fmt.Fprintln(w, "Profile sources"); err != nil {
		return err
	}
	if len(report.Sources) == 0 {
		if _, err := fmt.Fprintln(w, "  none"); err != nil {
			return err
		}
	}
	for i, source := range report.Sources {
		loc := firstNonEmpty(source.Path, source.DSN, source.Raw)
		if _, err := fmt.Fprintf(w, "  %d. %s %s\n", i+1, source.Kind, loc); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "\nDefault selection\n  registry: %s\n  profile:  %s\n", firstNonEmpty(report.DefaultRegistry, "none"), firstNonEmpty(report.DefaultProfile, "none")); err != nil {
		return err
	}
	if report.SelectedRegistry != "" || report.SelectedProfile != "" {
		if _, err := fmt.Fprintf(w, "\nSelected profile\n  registry: %s\n  profile:  %s\n", firstNonEmpty(report.SelectedRegistry, report.DefaultRegistry, "none"), firstNonEmpty(report.SelectedProfile, report.DefaultProfile, "none")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "\nRegistries"); err != nil {
		return err
	}
	if len(report.Registries) == 0 {
		if _, err := fmt.Fprintln(w, "  none"); err != nil {
			return err
		}
	}
	for _, reg := range report.Registries {
		marker := " "
		if reg.IsDefault {
			marker = "*"
		}
		if _, err := fmt.Fprintf(w, "  %s %s profiles=%d default_profile=%s\n", marker, reg.Slug, reg.ProfileCount, firstNonEmpty(reg.DefaultProfile, "none")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "\nProfiles"); err != nil {
		return err
	}
	if len(report.Profiles) == 0 {
		if _, err := fmt.Fprintln(w, "  none"); err != nil {
			return err
		}
	}
	for _, profile := range report.Profiles {
		marker := " "
		if profile.IsDefault {
			marker = "*"
		}
		selected := " "
		if profile.IsSelected || (report.SelectedProfile != "" && profile.Slug == report.SelectedProfile && (report.SelectedRegistry == "" || profile.Registry == report.SelectedRegistry)) {
			selected = ">"
		}
		desc := firstNonEmpty(profile.Description, profile.DisplayName)
		if _, err := fmt.Fprintf(w, "  %s%s %-14s %-24s model=%-16s api=%-18s %s\n", selected, marker, profile.Registry, profile.Slug, firstNonEmpty(profile.Model, "-"), firstNonEmpty(profile.APIType, "-"), desc); err != nil {
			return err
		}
	}
	if report.Resolution != nil {
		if _, err := fmt.Fprintf(w, "\nResolved profile\n  registry: %s\n  profile:  %s\n", report.Resolution.Registry, report.Resolution.Profile); err != nil {
			return err
		}
		if len(report.Resolution.Lineage) > 0 {
			if _, err := fmt.Fprintln(w, "\nStack lineage"); err != nil {
				return err
			}
			for i, entry := range report.Resolution.Lineage {
				if _, err := fmt.Fprintf(w, "  %d. %s/%s version=%d source=%s\n", i+1, entry.RegistrySlug, entry.EngineProfileSlug, entry.Version, firstNonEmpty(entry.Source, "-")); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func sourceReports(entries []string) []ProfileRegistrySourceReport {
	specs, err := gepprofiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		ret := make([]ProfileRegistrySourceReport, 0, len(entries))
		for _, entry := range entries {
			if strings.TrimSpace(entry) != "" {
				ret = append(ret, ProfileRegistrySourceReport{Raw: strings.TrimSpace(entry), Kind: "invalid"})
			}
		}
		return ret
	}
	ret := make([]ProfileRegistrySourceReport, 0, len(specs))
	for _, spec := range specs {
		ret = append(ret, ProfileRegistrySourceReport{Raw: spec.Raw, Kind: string(spec.Kind), Path: spec.Path, DSN: spec.DSN})
	}
	return ret
}

func profileModelAndAPIType(profile *gepprofiles.EngineProfile) (string, string) {
	if profile == nil || profile.InferenceSettings == nil || profile.InferenceSettings.Chat == nil {
		return "", ""
	}
	model := ""
	if profile.InferenceSettings.Chat.Engine != nil {
		model = strings.TrimSpace(*profile.InferenceSettings.Chat.Engine)
	}
	apiType := ""
	if profile.InferenceSettings.Chat.ApiType != nil {
		apiType = strings.TrimSpace(string(*profile.InferenceSettings.Chat.ApiType))
	}
	return model, apiType
}

func isSelectedProfile(registry gepprofiles.RegistrySlug, profile gepprofiles.EngineProfileSlug, in gepprofiles.ResolveInput) bool {
	if in.EngineProfileSlug.IsZero() || profile != in.EngineProfileSlug {
		return false
	}
	return in.RegistrySlug.IsZero() || registry == in.RegistrySlug
}

func inferenceSettingsMap(in any) (map[string]any, error) {
	b, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func RedactProfileSecrets(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, child := range typed {
			if isSensitiveProfileKey(k) {
				out[k] = "***REDACTED***"
				continue
			}
			out[k] = RedactProfileSecrets(child)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, child := range typed {
			out = append(out, RedactProfileSecrets(child))
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, 0, len(typed))
		for _, child := range typed {
			redacted, _ := RedactProfileSecrets(child).(map[string]any)
			out = append(out, redacted)
		}
		return out
	default:
		return typed
	}
}

func isSensitiveProfileKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, needle := range []string{"api_key", "api-key", "apikey", "token", "secret", "password", "credential", "authorization", "auth_header", "key"} {
		if strings.Contains(k, needle) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
