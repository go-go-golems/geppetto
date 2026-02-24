package middlewarecfg

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	cmdschema "github.com/go-go-golems/glazed/pkg/cmds/schema"
)

// AdapterResult contains the generated Glazed section and non-fatal mapping limitations.
type AdapterResult struct {
	Section     *cmdschema.SectionImpl
	Limitations []string
}

// AdaptSchemaToGlazedSection maps a JSON schema object to a Glazed section definition.
//
// Supported mapping:
// - required -> fields.WithRequired
// - default -> fields.WithDefault
// - description/title -> fields.WithHelp
// - string enum -> fields.TypeChoice + fields.WithChoices
//
// Limitations are returned for schema constructs that do not map cleanly to Glazed fields.
func AdaptSchemaToGlazedSection(
	sectionSlug string,
	sectionName string,
	schema map[string]any,
) (*AdapterResult, error) {
	slug := strings.TrimSpace(sectionSlug)
	if slug == "" {
		return nil, fmt.Errorf("glazed adapter: section slug is empty")
	}
	name := strings.TrimSpace(sectionName)
	if name == "" {
		return nil, fmt.Errorf("glazed adapter: section name is empty")
	}

	root := copyStringAnyMap(schema)
	if len(root) == 0 {
		root = map[string]any{"type": "object"}
	}
	if rootType := schemaType(root); rootType != "object" {
		return nil, fmt.Errorf("glazed adapter: root schema type must be object, got %q", rootType)
	}

	requiredSet := map[string]struct{}{}
	for _, key := range schemaRequiredFields(root) {
		requiredSet[key] = struct{}{}
	}

	limitations := make([]string, 0, 2)
	if additional := root["additionalProperties"]; additional != nil {
		if b, ok := additional.(bool); ok && b {
			limitations = append(limitations, "additionalProperties=true is not representable as static Glazed fields")
		}
	}

	properties, _ := toStringAnyMap(root["properties"])
	propertyNames := make([]string, 0, len(properties))
	for key := range properties {
		propertyNames = append(propertyNames, key)
	}
	sort.Strings(propertyNames)

	fieldDefs := make([]*fields.Definition, 0, len(propertyNames))
	for _, propertyName := range propertyNames {
		propertySchema, ok := toStringAnyMap(properties[propertyName])
		if !ok {
			limitations = append(limitations, fmt.Sprintf("property %q is not an object schema; skipped", propertyName))
			continue
		}
		field, fieldLimitations, err := fieldFromPropertySchema(propertyName, propertySchema, isRequired(requiredSet, propertyName))
		if err != nil {
			return nil, err
		}
		limitations = append(limitations, fieldLimitations...)
		if field == nil {
			continue
		}
		fieldDefs = append(fieldDefs, field)
	}

	section, err := cmdschema.NewSection(
		slug,
		name,
		cmdschema.WithDescription("Generated from middleware JSON schema."),
		cmdschema.WithFields(fieldDefs...),
	)
	if err != nil {
		return nil, err
	}

	return &AdapterResult{
		Section:     section,
		Limitations: uniqueSorted(limitations),
	}, nil
}

func fieldFromPropertySchema(
	fieldName string,
	propertySchema map[string]any,
	required bool,
) (*fields.Definition, []string, error) {
	name := strings.TrimSpace(fieldName)
	if name == "" {
		return nil, nil, nil
	}

	fieldType, limitations := glazedFieldTypeForProperty(propertySchema)
	if fieldType == "" {
		return nil, limitations, nil
	}

	options := make([]fields.Option, 0, 5)
	help := propertyHelpText(propertySchema)
	if help != "" {
		options = append(options, fields.WithHelp(help))
	}
	if required {
		options = append(options, fields.WithRequired(true))
	}
	if defaultValue, ok := propertySchema["default"]; ok {
		coercedDefault, err := coerceAndValidate(propertySchema, defaultValue)
		if err != nil {
			return nil, nil, fmt.Errorf("glazed adapter: default for %q is invalid: %w", name, err)
		}
		options = append(options, fields.WithDefault(coercedDefault))
	}

	enumChoices, enumLimitations := enumChoicesForFieldType(propertySchema, fieldType)
	limitations = append(limitations, enumLimitations...)
	if len(enumChoices) > 0 {
		options = append(options, fields.WithChoices(enumChoices...))
	}

	return fields.New(name, fieldType, options...), limitations, nil
}

func glazedFieldTypeForProperty(propertySchema map[string]any) (fields.Type, []string) {
	typ := schemaType(propertySchema)
	switch typ {
	case "string":
		if hasStringEnum(propertySchema) {
			return fields.TypeChoice, nil
		}
		return fields.TypeString, nil
	case "integer":
		return fields.TypeInteger, nil
	case "number":
		return fields.TypeFloat, nil
	case "boolean":
		return fields.TypeBool, nil
	case "array":
		itemsSchema, ok := toStringAnyMap(propertySchema["items"])
		if !ok {
			return fields.TypeStringList, []string{"array schema without items mapped to string list"}
		}
		switch schemaType(itemsSchema) {
		case "string":
			if hasStringEnum(itemsSchema) {
				return fields.TypeChoiceList, nil
			}
			return fields.TypeStringList, nil
		case "integer":
			return fields.TypeIntegerList, nil
		case "number":
			return fields.TypeFloatList, nil
		default:
			return "", []string{fmt.Sprintf("array item type %q is not supported for Glazed mapping", schemaType(itemsSchema))}
		}
	case "object":
		return "", []string{"nested object fields are not flattened into Glazed fields"}
	default:
		return "", []string{fmt.Sprintf("schema type %q is not supported for Glazed mapping", typ)}
	}
}

func enumChoicesForFieldType(propertySchema map[string]any, fieldType fields.Type) ([]string, []string) {
	enumValues, ok := toAnySlice(propertySchema["enum"])
	if !ok || len(enumValues) == 0 {
		return nil, nil
	}
	if fieldType != fields.TypeChoice {
		return nil, []string{fmt.Sprintf("enum on non-choice field type %q is not mapped", fieldType)}
	}

	choices := make([]string, 0, len(enumValues))
	for _, raw := range enumValues {
		s, ok := raw.(string)
		if !ok {
			return nil, []string{"enum contains non-string values; enum choices not mapped"}
		}
		choices = append(choices, s)
	}
	sort.Strings(choices)
	return choices, nil
}

func hasStringEnum(schema map[string]any) bool {
	enumValues, ok := toAnySlice(schema["enum"])
	if !ok || len(enumValues) == 0 {
		return false
	}
	for _, raw := range enumValues {
		if _, ok := raw.(string); !ok {
			return false
		}
	}
	return true
}

func propertyHelpText(schema map[string]any) string {
	if description, ok := schema["description"].(string); ok && strings.TrimSpace(description) != "" {
		return strings.TrimSpace(description)
	}
	if title, ok := schema["title"].(string); ok && strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	return ""
}

func isRequired(required map[string]struct{}, key string) bool {
	_, ok := required[key]
	return ok
}

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, entry := range in {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		set[entry] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for entry := range set {
		out = append(out, entry)
	}
	sort.Strings(out)
	return out
}
