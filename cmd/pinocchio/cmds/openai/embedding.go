package openai

//var EmbeddingsCmd = &cobra.Command{
//	Use:   "embeddings",
//	Short: "Compute embeddings for a series of files",
//	Args:  cobra.MinimumNArgs(1),
//	Run: func(cmd *cobra.Command, args []string) {
//		user, _ := cmd.PersistentFlags().GetString("user")
//
//		prompts := []string{}
//
//		for _, file := range args {
//
//			if file == "-" {
//				file = "/dev/stdin"
//			}
//
//			f, err := os.ReadFile(file)
//			cobra.CheckErr(err)
//
//			prompts = append(prompts, string(f))
//		}
//
//		// TODO(manuel, 2023-01-28) actually I don't think it's a good idea to go through the stepfactory here
//		// we just want to have the RAW api access with all its outputs
//
//		clientSettings, err := openai.NewClientSettingsFromCobra(cmd)
//		cobra.CheckErr(err)
//
//		err = completionStepFactory.UpdateFromParameters(cmd)
//		cobra.CheckErr(err)
//
//		client, err := clientSettings.CreateClient()
//		cobra.CheckErr(err)
//
//		engine, _ := cmd.Flags().GetString("engine")
//
//		ctx := context.Background()
//		resp, err := client.Embeddings(ctx, gpt3.EmbeddingsRequest{
//			Input: prompts,
//			Model: engine,
//			User:  user,
//		})
//		cobra.CheckErr(err)
//
//		printUsage, _ := cmd.Flags().GetBool("print-usage")
//		usage := resp.Usage
//		evt := log.Debug()
//		if printUsage {
//			evt = log.Info()
//		}
//		evt.
//			Int("prompt-tokens", usage.PromptTokens).
//			Int("total-tokens", usage.TotalTokens).
//			Msg("Usage")
//
//		gp, of, err := cli.SetupProcessor(cmd)
//		cobra.CheckErr(err)
//
//		printRawResponse, _ := cmd.Flags().GetBool("print-raw-response")
//
//		if printRawResponse {
//			// serialize resp to json
//			rawResponse, err := json.MarshalIndent(resp, "", "  ")
//			cobra.CheckErr(err)
//
//			// deserialize to map[string]interface{}
//			var rawResponseMap map[string]interface{}
//			err = json.Unmarshal(rawResponse, &rawResponseMap)
//			cobra.CheckErr(err)
//
//			err = gp.ProcessInputObject(rawResponseMap)
//			cobra.CheckErr(err)
//
//		} else {
//			for _, embedding := range resp.Data {
//				row := map[string]interface{}{
//					"object":    embedding.Object,
//					"embedding": embedding.Embedding,
//					"index":     embedding.Index,
//				}
//				err = gp.ProcessInputObject(row)
//				cobra.CheckErr(err)
//			}
//		}
//
//		s, err := of.Output()
//		if err != nil {
//			_, _ = fmt.Fprintf(os.Stderr, "Error rendering output: %s\n", err)
//			os.Exit(1)
//		}
//		fmt.Print(s)
//		cobra.CheckErr(err)
//	},
//}
