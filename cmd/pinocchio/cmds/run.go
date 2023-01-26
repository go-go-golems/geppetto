package cmds

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wesen/geppetto/pkg/steps"
	"os"
)

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "run a LLM application",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]

		if file == "-" {
			file = "/dev/stdin"
		}
		f, err := os.ReadFile(file)
		cobra.CheckErr(err)

		step := steps.NewOpenAICompletionStep(viper.GetString("openai-api-key"))
		ctx := context.Background()
		err = step.Start(ctx, string(f))
		cobra.CheckErr(err)

		result := <-step.GetOutput()
		v, err := result.Value()
		cobra.CheckErr(err)

		_, err = os.Stdout.Write([]byte(v))
		cobra.CheckErr(err)
	},
}

func init() {
}
