package sections

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

const ProfileIntrospectionSectionSlug = "profile-introspection"

type ProfileIntrospectionSettings struct {
	PrintProfiles          bool   `glazed:"print-profiles"`
	PrintProfileResolution bool   `glazed:"print-profile-resolution"`
	ProfileOutput          string `glazed:"profile-output"`
}

func NewProfileIntrospectionSection() (schema.Section, error) {
	return schema.NewSection(
		ProfileIntrospectionSectionSlug,
		"Profile introspection",
		schema.WithFields(
			fields.New(
				"print-profiles",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Print loaded profile registries and profiles, then exit before running inference"),
			),
			fields.New(
				"print-profile-resolution",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Include selected profile stack lineage and redacted merged settings in --print-profiles output"),
			),
			fields.New(
				"profile-output",
				fields.TypeString,
				fields.WithDefault("text"),
				fields.WithHelp("Profile introspection output format (text, json, yaml)"),
			),
		),
	)
}
