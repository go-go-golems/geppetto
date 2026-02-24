package profiles

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._-]{0,126}[a-z0-9])?$`)

type RegistrySlug string

type ProfileSlug string

type RuntimeKey string

func ParseRegistrySlug(raw string) (RegistrySlug, error) {
	normalized, err := parseSlug("registry slug", raw)
	if err != nil {
		return "", err
	}
	return RegistrySlug(normalized), nil
}

func ParseProfileSlug(raw string) (ProfileSlug, error) {
	normalized, err := parseSlug("profile slug", raw)
	if err != nil {
		return "", err
	}
	return ProfileSlug(normalized), nil
}

func ParseRuntimeKey(raw string) (RuntimeKey, error) {
	normalized, err := parseSlug("runtime key", raw)
	if err != nil {
		return "", err
	}
	return RuntimeKey(normalized), nil
}

func MustRegistrySlug(raw string) RegistrySlug {
	slug, err := ParseRegistrySlug(raw)
	if err != nil {
		panic(err)
	}
	return slug
}

func MustProfileSlug(raw string) ProfileSlug {
	slug, err := ParseProfileSlug(raw)
	if err != nil {
		panic(err)
	}
	return slug
}

func MustRuntimeKey(raw string) RuntimeKey {
	key, err := ParseRuntimeKey(raw)
	if err != nil {
		panic(err)
	}
	return key
}

func (s RegistrySlug) String() string { return string(s) }

func (s ProfileSlug) String() string { return string(s) }

func (s RuntimeKey) String() string { return string(s) }

func (s RegistrySlug) IsZero() bool { return strings.TrimSpace(string(s)) == "" }

func (s ProfileSlug) IsZero() bool { return strings.TrimSpace(string(s)) == "" }

func (s RuntimeKey) IsZero() bool { return strings.TrimSpace(string(s)) == "" }

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

func (s ProfileSlug) MarshalText() ([]byte, error) {
	if s.IsZero() {
		return []byte(""), nil
	}
	if _, err := ParseProfileSlug(string(s)); err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func (s *ProfileSlug) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*s = ""
		return nil
	}
	parsed, err := ParseProfileSlug(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (s RuntimeKey) MarshalText() ([]byte, error) {
	if s.IsZero() {
		return []byte(""), nil
	}
	if _, err := ParseRuntimeKey(string(s)); err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func (s *RuntimeKey) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*s = ""
		return nil
	}
	parsed, err := ParseRuntimeKey(string(text))
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

func (s ProfileSlug) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return []byte(`""`), nil
	}
	if _, err := ParseProfileSlug(string(s)); err != nil {
		return nil, err
	}
	return json.Marshal(string(s))
}

func (s *ProfileSlug) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseProfileSlug(raw)
	if err != nil && strings.TrimSpace(raw) != "" {
		return err
	}
	*s = parsed
	return nil
}

func (s RuntimeKey) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return []byte(`""`), nil
	}
	if _, err := ParseRuntimeKey(string(s)); err != nil {
		return nil, err
	}
	return json.Marshal(string(s))
}

func (s *RuntimeKey) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseRuntimeKey(raw)
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

func (s ProfileSlug) MarshalYAML() (interface{}, error) {
	if s.IsZero() {
		return "", nil
	}
	if _, err := ParseProfileSlug(string(s)); err != nil {
		return nil, err
	}
	return string(s), nil
}

func (s *ProfileSlug) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || strings.TrimSpace(value.Value) == "" {
		*s = ""
		return nil
	}
	parsed, err := ParseProfileSlug(value.Value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (s RuntimeKey) MarshalYAML() (interface{}, error) {
	if s.IsZero() {
		return "", nil
	}
	if _, err := ParseRuntimeKey(string(s)); err != nil {
		return nil, err
	}
	return string(s), nil
}

func (s *RuntimeKey) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || strings.TrimSpace(value.Value) == "" {
		*s = ""
		return nil
	}
	parsed, err := ParseRuntimeKey(value.Value)
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
