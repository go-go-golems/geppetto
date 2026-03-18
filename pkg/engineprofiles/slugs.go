package engineprofiles

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._-]{0,126}[a-z0-9])?$`)

type RegistrySlug string

type EngineProfileSlug string

func ParseRegistrySlug(raw string) (RegistrySlug, error) {
	normalized, err := parseSlug("registry slug", raw)
	if err != nil {
		return "", err
	}
	return RegistrySlug(normalized), nil
}

func ParseEngineProfileSlug(raw string) (EngineProfileSlug, error) {
	normalized, err := parseSlug("profile slug", raw)
	if err != nil {
		return "", err
	}
	return EngineProfileSlug(normalized), nil
}

func MustRegistrySlug(raw string) RegistrySlug {
	slug, err := ParseRegistrySlug(raw)
	if err != nil {
		panic(err)
	}
	return slug
}

func MustEngineProfileSlug(raw string) EngineProfileSlug {
	slug, err := ParseEngineProfileSlug(raw)
	if err != nil {
		panic(err)
	}
	return slug
}

func (s RegistrySlug) String() string { return string(s) }

func (s EngineProfileSlug) String() string { return string(s) }

func (s RegistrySlug) IsZero() bool { return strings.TrimSpace(string(s)) == "" }

func (s EngineProfileSlug) IsZero() bool { return strings.TrimSpace(string(s)) == "" }

func (s RegistrySlug) MarshalText() ([]byte, error) {
	if s.IsZero() {
		return []byte(""), nil
	}
	if _, err := ParseRegistrySlug(string(s)); err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func (s *RegistrySlug) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*s = ""
		return nil
	}
	parsed, err := ParseRegistrySlug(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (s EngineProfileSlug) MarshalText() ([]byte, error) {
	if s.IsZero() {
		return []byte(""), nil
	}
	if _, err := ParseEngineProfileSlug(string(s)); err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func (s *EngineProfileSlug) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*s = ""
		return nil
	}
	parsed, err := ParseEngineProfileSlug(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (s RegistrySlug) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return []byte(`""`), nil
	}
	if _, err := ParseRegistrySlug(string(s)); err != nil {
		return nil, err
	}
	return json.Marshal(string(s))
}

func (s *RegistrySlug) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseRegistrySlug(raw)
	if err != nil && strings.TrimSpace(raw) != "" {
		return err
	}
	*s = parsed
	return nil
}

func (s EngineProfileSlug) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return []byte(`""`), nil
	}
	if _, err := ParseEngineProfileSlug(string(s)); err != nil {
		return nil, err
	}
	return json.Marshal(string(s))
}

func (s *EngineProfileSlug) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseEngineProfileSlug(raw)
	if err != nil && strings.TrimSpace(raw) != "" {
		return err
	}
	*s = parsed
	return nil
}

func (s RegistrySlug) MarshalYAML() (interface{}, error) {
	if s.IsZero() {
		return "", nil
	}
	if _, err := ParseRegistrySlug(string(s)); err != nil {
		return nil, err
	}
	return string(s), nil
}

func (s *RegistrySlug) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || strings.TrimSpace(value.Value) == "" {
		*s = ""
		return nil
	}
	parsed, err := ParseRegistrySlug(value.Value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (s EngineProfileSlug) MarshalYAML() (interface{}, error) {
	if s.IsZero() {
		return "", nil
	}
	if _, err := ParseEngineProfileSlug(string(s)); err != nil {
		return nil, err
	}
	return string(s), nil
}

func (s *EngineProfileSlug) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || strings.TrimSpace(value.Value) == "" {
		*s = ""
		return nil
	}
	parsed, err := ParseEngineProfileSlug(value.Value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func parseSlug(label, raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", fmt.Errorf("%s cannot be empty", label)
	}
	if !slugPattern.MatchString(normalized) {
		return "", fmt.Errorf("%s %q is invalid", label, raw)
	}
	return normalized, nil
}
