package cmds

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// commands for manipulating the config
//
// - add a repository entry
// - set ai keys
// - add profile / remove profile / update profile
//
// layers that are loaded from the config file:
// (from cobra.go)
// - ai-chat
// - ai-client
// - openai-chat
// - claude-chat

func NewRepositoriesGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repositories",
		Short: "Manage repositories in the configuration",
	}

	cmd.AddCommand(NewAddRepositoryCommand())
	cmd.AddCommand(NewRemoveRepositoryCommand())
	cmd.AddCommand(NewPrintRepositoriesCommand())

	return cmd
}

func NewAddRepositoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [directories...]",
		Short: "Add directories to the repository entry in the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("at least one directory must be provided")
			}

			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				return fmt.Errorf("no config file found")
			}

			// Read the existing config file
			data, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("error reading config file: %w", err)
			}

			// Parse the YAML
			var root yaml.Node
			err = yaml.Unmarshal(data, &root)
			if err != nil {
				return fmt.Errorf("error parsing config file: %w", err)
			}

			// Find or create the repository node
			repoNode := findOrCreateNode(&root, "repositories")

			added := false

			// Append new directories
			for _, dir := range args {
				absDir, err := filepath.Abs(dir)
				if err != nil {
					return fmt.Errorf("error getting absolute path for %s: %w", dir, err)
				}

				// Check if the repository already exists
				if repoExists(repoNode, absDir) {
					fmt.Printf("Repository %s already exists in the list. Skipping.\n", absDir)
					continue
				}

				fmt.Printf("Adding %s to repository list.\n", absDir)
				repoNode.Content = append(repoNode.Content, &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: absDir,
				})
				added = true
			}

			// Write the updated config back to file
			f, err := os.Create(configFile)
			if err != nil {
				return fmt.Errorf("error opening config file for writing: %w", err)
			}
			defer f.Close()

			encoder := yaml.NewEncoder(f)
			encoder.SetIndent(2)
			err = encoder.Encode(&root)
			if err != nil {
				return fmt.Errorf("error writing config file: %w", err)
			}

			// Print out the total list of repositories
			if added {
				fmt.Println("\nCurrent repository list:")
				for _, node := range repoNode.Content {
					fmt.Printf("- %s\n", node.Value)
				}
			}

			return nil
		},
	}

	return cmd
}

func NewRemoveRepositoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [directories...]",
		Short: "Remove directories from the repository entry in the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("at least one directory must be provided")
			}

			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				return fmt.Errorf("no config file found")
			}

			// Read and parse the config file
			root, err := readAndParseConfig(configFile)
			if err != nil {
				return err
			}

			repoNode := findOrCreateNode(root, "repositories")

			removed := false
			for _, dir := range args {
				absDir, err := filepath.Abs(dir)
				if err != nil {
					return fmt.Errorf("error getting absolute path for %s: %w", dir, err)
				}

				if removeRepo(repoNode, absDir) {
					fmt.Printf("Removed %s from repository list.\n", absDir)
					removed = true
				} else {
					fmt.Printf("Repository %s not found in the list. Skipping.\n", absDir)
				}
			}

			if removed {
				// Write the updated config back to file
				if err := writeConfig(configFile, root); err != nil {
					return err
				}

				fmt.Println("\nUpdated repository list:")
				printRepos(repoNode)
			}

			return nil
		},
	}
	return cmd
}

func NewPrintRepositoriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Print the list of repositories in the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				return fmt.Errorf("no config file found")
			}

			// Read and parse the config file
			root, err := readAndParseConfig(configFile)
			if err != nil {
				return err
			}

			repoNode := findOrCreateNode(root, "repositories")

			printRepos(repoNode)

			return nil
		},
	}
	return cmd
}

func findOrCreateNode(root *yaml.Node, key string) *yaml.Node {
	if root.Kind != yaml.DocumentNode {
		root = &yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: []*yaml.Node{root},
		}
	}

	var mapNode *yaml.Node
	if len(root.Content) > 0 && root.Content[0].Kind == yaml.MappingNode {
		mapNode = root.Content[0]
	} else {
		mapNode = &yaml.Node{Kind: yaml.MappingNode}
		root.Content = []*yaml.Node{mapNode}
	}

	for i := 0; i < len(mapNode.Content); i += 2 {
		if mapNode.Content[i].Value == key {
			if mapNode.Content[i+1].Kind != yaml.SequenceNode {
				mapNode.Content[i+1] = &yaml.Node{Kind: yaml.SequenceNode}
			}
			return mapNode.Content[i+1]
		}
	}

	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: key,
	}
	valueNode := &yaml.Node{
		Kind: yaml.SequenceNode,
	}
	mapNode.Content = append(mapNode.Content, keyNode, valueNode)
	return valueNode
}

func repoExists(repoNode *yaml.Node, dir string) bool {
	for _, node := range repoNode.Content {
		if node.Value == dir {
			return true
		}
	}
	return false
}

func removeRepo(repoNode *yaml.Node, dir string) bool {
	for i, node := range repoNode.Content {
		if node.Value == dir {
			repoNode.Content = append(repoNode.Content[:i], repoNode.Content[i+1:]...)
			return true
		}
	}
	return false
}

func printRepos(repoNode *yaml.Node) {
	for _, node := range repoNode.Content {
		fmt.Printf("- %s\n", node.Value)
	}
}

func readAndParseConfig(configFile string) (*yaml.Node, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var root yaml.Node
	err = yaml.Unmarshal(data, &root)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &root, nil
}

func writeConfig(configFile string, root *yaml.Node) error {
	f, err := os.Create(configFile)
	if err != nil {
		return fmt.Errorf("error opening config file for writing: %w", err)
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	err = encoder.Encode(root)
	if err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

func NewConfigGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Commands for manipulating configuration and profiles",
	}

	cmd.AddCommand(NewRepositoriesGroupCommand())

	return cmd
}
