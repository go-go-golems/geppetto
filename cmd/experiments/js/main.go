package main

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/js"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/pinocchio/pkg/cmds"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type testFlags struct {
	runStepTests        bool
	runConversationTest bool
	runChatStepTest     bool
}

var rootCmd = &cobra.Command{
	Use:   "js-experiments",
	Short: "JavaScript experiments for Geppetto",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// reinitialize the logger because we can now parse --log-level and co
		err := clay.InitLogger()
		cobra.CheckErr(err)
	},
}

var runCmd *cobra.Command

func main() {
	err := clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	stepSettings, err := settings.NewStepSettings()
	cobra.CheckErr(err)
	geppettoLayers, err := cmds.CreateGeppettoLayers(stepSettings)
	cobra.CheckErr(err)
	layers_ := layers.NewParameterLayers(layers.WithLayers(geppettoLayers...))

	parser, err := cli.NewCobraParserFromLayers(
		layers_,
		cli.WithCobraMiddlewaresFunc(cmds.GetCobraCommandGeppettoMiddlewares))
	cobra.CheckErr(err)

	flags := &testFlags{
		runStepTests:        true,
		runConversationTest: true,
		runChatStepTest:     true,
	}

	// Add run command that executes JS tests
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run JavaScript tests",
		Run: func(cmd *cobra.Command, args []string) {
			parsedLayers, err := parser.Parse(cmd, nil)
			cobra.CheckErr(err)

			err = stepSettings.UpdateFromParsedLayers(parsedLayers)
			cobra.CheckErr(err)

			// Create event loop
			loop := eventloop.NewEventLoop()
			// loop.Start()
			defer loop.Stop()

			// Run tests in event loop
			loop.RunOnLoop(func(vm *goja.Runtime) {
				setupConsole(vm)

				if flags.runStepTests {
					runStepTests(vm, loop)
				}

				if flags.runConversationTest {
					runConversationTest(vm)
				}

				if flags.runChatStepTest {
					runChatStepTest(vm, loop, stepSettings)
				}
			})

			log.Info().Msg("Starting loop")
			loop.StartInForeground()

			log.Info().Msg("Waiting for 10 seconds")
			time.Sleep(10 * time.Second)
			log.Info().Msg("Done")
		},
	}

	runCmd.Flags().BoolVar(&flags.runStepTests, "step-tests", false, "Run step tests")
	runCmd.Flags().BoolVar(&flags.runConversationTest, "conversation-test", true, "Run conversation test")
	runCmd.Flags().BoolVar(&flags.runChatStepTest, "chat-step-test", true, "Run chat step test")

	err = parser.AddToCobraCommand(runCmd)
	cobra.CheckErr(err)

	rootCmd.AddCommand(runCmd)

	err = rootCmd.Execute()
	cobra.CheckErr(err)
}

func setupConsole(vm *goja.Runtime) {
	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Println(args...)
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Printf("ERROR: %v\n", args...)
		return goja.Undefined()
	})
	_ = vm.Set("console", console)
}

func runStepTests(vm *goja.Runtime, loop *eventloop.EventLoop) {
	// Create a simple test step that doubles numbers with delay
	doubleStep := &utils.LambdaStep[float64, float64]{
		Function: func(input float64) helpers.Result[float64] {
			fmt.Println("Starting doubleStep")
			time.Sleep(500 * time.Millisecond)
			fmt.Println("Finished doubleStep")
			return helpers.NewValueResult(input * 2)
		},
	}

	// Register step in JS
	err := js.RegisterStep(
		vm,
		loop,
		"doubleStep",
		doubleStep,
		func(v goja.Value) float64 { return v.ToFloat() },
		func(v float64) goja.Value { return vm.ToValue(v) },
	)
	cobra.CheckErr(err)

	stepTestJS := `
		// Test Promise-based API
		async function testPromise() {
			console.log("Testing Promise API...");
			try {
				const promise = doubleStep.startAsync(21)
				console.log("Promise created")
				const result = await promise;
				console.log("Promise result:", result[0]);
			} catch (err) {
				console.error("Promise error:", err);
			}
		}

		// Test blocking API
		function testBlocking() {
			console.log("Testing Blocking API...");
			try {
				const result = doubleStep.startBlocking(32);
				console.log("Blocking result:", result[0]);
			} catch (err) {
				console.error("Blocking error:", err);
			}
		}

		// Test callback-based API
		function testCallbacks() {
			console.log("Testing Callbacks API...");
			const cancel = doubleStep.startWithCallbacks(43, {
				onResult: (result) => console.log("Callback result:", result),
				onError: (err) => console.error("Callback error:", err),
				onDone: () => console.log("Callbacks complete"),
			});
		}

		// Run tests sequentially
		async function runStepTests() {
			console.log("=== Running Step Tests ===");
			await testPromise();
			testBlocking();
			testCallbacks();
			console.log("Step tests complete");
		}

		runStepTests().catch(console.error);
	`

	_, err = vm.RunString(stepTestJS)
	cobra.CheckErr(err)
}

func runConversationTest(vm *goja.Runtime) {
	// Register conversation constructor
	err := js.RegisterConversation(vm)
	cobra.CheckErr(err)

	conversationTestJS := `
		async function runConversationTest() {
			console.log("=== Running Conversation Test ===");
			
			const conv = new Conversation();
			console.log("Conversation created:", conv);
			
			try {
				// Test adding messages
				const msgId1 = await conv.AddMessage("system", "You are a helpful assistant.");
				console.log("Added system message:", msgId1);
				
				const msgId2 = await conv.AddMessage("user", "Hello, can you help me?");
				console.log("Added user message:", msgId2);
				
				const msgId3 = await conv.AddMessage("assistant", "Of course! What can I help you with?");
				console.log("Added assistant message:", msgId3);
				
				// Test tool use
				const toolId = "search-123";
				const toolUseId = await conv.AddToolUse(toolId, "search", { query: "test query" });
				console.log("Added tool use:", toolUseId);
				
				const toolResultId = await conv.AddToolResult(toolId, "Found results for test query");
				console.log("Added tool result:", toolResultId);
				
				// Test getting messages
				const messages = conv.GetMessages();
				console.log("Messages count:", messages.length);
				console.log("First message role:", messages[0].Content.Role);
				
				// Test getting single prompt
				const prompt = conv.GetSinglePrompt();
				console.log("Single prompt:", prompt);
				
				// Test message view
				const view = await conv.GetMessageView(msgId1);
				console.log("Message view:", view);
				
				// Test metadata update
				const updated = await conv.UpdateMetadata(msgId1, { processed: true });
				console.log("Metadata updated:", updated);
				
				console.log("Conversation test complete");
			} catch (err) {
				console.error("Test error:", err);
			}
		}
		
		runConversationTest().catch(console.error);
	`

	_, err = vm.RunString(conversationTestJS)
	cobra.CheckErr(err)
}

func runChatStepTest(vm *goja.Runtime, loop *eventloop.EventLoop, stepSettings *settings.StepSettings) {
	// Register conversation constructor
	err := js.RegisterConversation(vm)
	cobra.CheckErr(err)

	// Register chat step factory
	err = js.RegisterFactory(vm, loop, stepSettings)
	cobra.CheckErr(err)

	chatStepTestJS := `
		async function runChatStepTest() {
			console.log("=== Running Chat Step Test ===");
			
			// Create factory and step
			const factory = new ChatStepFactory();
			const chatStep = factory.newStep();
			
			// Create conversation
			const conv = new Conversation();
			conv.AddMessage("system", "You are a helpful AI assistant. Be concise.");
			conv.AddMessage("user", "What is the capital of France?");
			
			// Test Promise API
			console.log("Testing Promise API...");
			try {
				const response = await chatStep.startAsync(conv);
				console.log("Promise response:", response);
				
				// Add assistant's response to conversation
				conv.AddMessage("assistant", response);
			} catch (err) {
				console.error("Promise API error:", err);
			}
			
			// Test Streaming API
			console.log("\nTesting Streaming API...");
			conv.AddMessage("user", "And what is France's population?");
			
			let streamingResponse = "";
			const cancel = chatStep.startWithCallbacks(conv, {
				onResult: (chunk) => {
					streamingResponse += chunk;
					console.log("Chunk received:", chunk);
				},
				onError: (err) => {
					console.error("Streaming error:", err);
				},
				onDone: () => {
					console.log("\nFinal streaming response:", streamingResponse);
					conv.AddMessage("assistant", streamingResponse);
					console.log("Chat step test complete");
				}
			});
			console.log("Streaming started")
			
			// Cancel after 10 seconds if not done
			setTimeout(() => {
				console.log("Cancelling streaming response")
				cancel();
				console.log("Cancelled streaming response");
			}, 10000);
		}
		
		runChatStepTest().catch(console.error);
	`

	_, err = vm.RunString(chatStepTestJS)
	cobra.CheckErr(err)
}
