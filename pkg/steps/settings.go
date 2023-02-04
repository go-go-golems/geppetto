package steps

type StepDescription struct {
	Type string `yaml:"type"`

	// TODO(manuel, 2023-02-04) This is all just a hack right now to get a sense of what we can
	// achieve in a YAML, and also just to get article.yaml working
	//
	// MultiInput is just the name of the input parameter used to iterate over the prompt
	MultiInput string `yaml:"multi_input,omitempty"`
}
