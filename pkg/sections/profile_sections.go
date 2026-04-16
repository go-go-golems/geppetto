package sections

import (
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

type ProfileSettings struct {
	Profile           string   `glazed:"profile"`
	ProfileRegistries []string `glazed:"profile-registries"`
}

type ProfileSettingsSectionOption func(*profileSettingsSectionOptions)

type profileSettingsSectionOptions struct {
	profileDefault           string
	profileRegistriesDefault []string
}

func WithProfileDefault(profile string) ProfileSettingsSectionOption {
	return func(o *profileSettingsSectionOptions) {
		o.profileDefault = strings.TrimSpace(profile)
	}
}

func WithProfileRegistriesDefault(entries ...string) ProfileSettingsSectionOption {
	return func(o *profileSettingsSectionOptions) {
		o.profileRegistriesDefault = o.profileRegistriesDefault[:0]
		for _, entry := range entries {
			if trimmed := strings.TrimSpace(entry); trimmed != "" {
				o.profileRegistriesDefault = append(o.profileRegistriesDefault, trimmed)
			}
		}
	}
}

const ProfileSettingsSectionSlug = "profile-settings"

func NewProfileSettingsSection(opts ...ProfileSettingsSectionOption) (schema.Section, error) {
	var sectionOptions profileSettingsSectionOptions
	for _, opt := range opts {
		opt(&sectionOptions)
	}

	profileOptions := []fields.Option{
		fields.WithHelp("Load the profile"),
	}
	if sectionOptions.profileDefault != "" {
		profileOptions = append(profileOptions, fields.WithDefault(sectionOptions.profileDefault))
	}

	profileRegistriesOptions := []fields.Option{
		fields.WithHelp("Comma-separated profile registry sources (yaml/sqlite/sqlite-dsn)"),
	}
	if len(sectionOptions.profileRegistriesDefault) > 0 {
		profileRegistriesOptions = append(profileRegistriesOptions, fields.WithDefault(append([]string(nil), sectionOptions.profileRegistriesDefault...)))
	}

	return schema.NewSection(
		ProfileSettingsSectionSlug,
		"Profile settings",
		schema.WithFields(
			fields.New("profile", fields.TypeString, profileOptions...),
			fields.New("profile-registries", fields.TypeStringList, profileRegistriesOptions...),
		),
	)
}
