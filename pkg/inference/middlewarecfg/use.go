package middlewarecfg

// Use describes a named middleware instance and optional config payload.
type Use struct {
	Name    string `json:"name" yaml:"name"`
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
	Enabled *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Config  any    `json:"config,omitempty" yaml:"config,omitempty"`
}
