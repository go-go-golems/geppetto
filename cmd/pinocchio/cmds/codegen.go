package cmds

import (
	"bytes"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/codegen"
	"github.com/spf13/cobra"
	"os"
	"path"
	"strings"
)

func NewCodegenCommand() *cobra.Command {
	ret := &cobra.Command{
		Use:   "codegen [file...]",
		Short: "A program to convert Geppetto YAML commands into Go code",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := cmd.Flag("package-name").Value.String()
			outputDir := cmd.Flag("output-dir").Value.String()

			s := &codegen.GeppettoCommandCodeGenerator{
				PackageName: packageName,
			}

			for _, fileName := range args {
				psYaml, err := os.ReadFile(fileName)
				if err != nil {
					return err
				}

				loader := &cmds.GeppettoCommandLoader{}

				// create reader from psYaml
				r := bytes.NewReader(psYaml)
				cmds_, err := loader.LoadCommandFromYAML(r)
				if err != nil {
					return err
				}
				if len(cmds_) != 1 {
					return fmt.Errorf("expected exactly one command, got %d", len(cmds_))
				}
				cmd := cmds_[0].(*cmds.GeppettoCommand)

				f, err := s.GenerateCommandCode(cmd)
				if err != nil {
					return err
				}

				s := f.GoString()
				// store in path.go after removing .yaml
				p, _ := strings.CutSuffix(path.Base(fileName), ".yaml")
				p = p + ".go"
				p = path.Join(outputDir, p)

				fmt.Printf("Converting %s to %s\n", fileName, p)
				_ = os.WriteFile(p, []byte(s), 0644)
			}

			return nil
		},
	}

	ret.PersistentFlags().StringP("output-dir", "o", ".", "Output directory for generated code")
	ret.PersistentFlags().StringP("package-name", "p", "main", "Package name for generated code")
	return ret
}
