package conversation

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

// LoadFromFile reads messages from a JSON or YAML file, facilitating conversation initialization
// from saved states.
func LoadFromFile(filename string) ([]*Message, error) {
	if strings.HasSuffix(filename, ".json") {
		return loadFromJSONFile(filename)
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		return loadFromYAMLFile(filename)
	} else {
		return nil, nil
	}
}

func loadFromYAMLFile(filename string) ([]*Message, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	var messages []*Message
	err = yaml.NewDecoder(f).Decode(&messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func loadFromJSONFile(filename string) ([]*Message, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	var messages []*Message
	err = json.NewDecoder(f).Decode(&messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (c *ManagerImpl) AppendMessages(messages ...*Message) {
	c.Tree.AppendMessages(messages)
}

func (c *ManagerImpl) AttachMessages(parentID NodeID, messages ...*Message) {
	c.Tree.AttachThread(parentID, messages)
}

func (c *ManagerImpl) PrependMessages(messages ...*Message) {
	c.Tree.PrependThread(messages)
}

// SaveToFile persists the current conversation state to a JSON file, enabling
// conversation continuity across sessions.
func (c *ManagerImpl) SaveToFile(s string) error {
	// TODO(manuel, 2023-11-14) For now only json
	msgs := c.GetConversation()
	f, err := os.Create(s)
	if err != nil {
		return err
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	// TODO(manuel, 2024-04-07) Encode as tree structure?? we skip the Children field on purpose to avoid circular references
	err = encoder.Encode(msgs)
	if err != nil {
		return err
	}

	return nil
}
