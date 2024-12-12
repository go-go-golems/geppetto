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
	"github.com/spf13/cobra"
)

type testFlags struct {
	runStepTests        bool
	runConversationTest bool
}

func main() {
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
	}

	// Add run command that executes JS tests
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run JavaScript tests",
		Run: func(cmd *cobra.Command, args []string) {
			parsedLayers, err := parser.Parse(cmd, nil)
			cobra.CheckErr(err)

			err = stepSettings.UpdateFromParsedLayers(parsedLayers)
			cobra.CheckErr(err)

			// Create event loop
			loop := eventloop.NewEventLoop()
			loop.Start()
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
			})

			time.Sleep(10 * time.Second)
		},
	}

	runCmd.Flags().BoolVar(&flags.runStepTests, "step-tests", true, "Run step tests")
	runCmd.Flags().BoolVar(&flags.runConversationTest, "conversation-test", true, "Run conversation test")

	err = clay.InitViper("geppetto", runCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	err = parser.AddToCobraCommand(runCmd)
	cobra.CheckErr(err)

	err = runCmd.Execute()
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
			
			// Test adding messages
			const msgId1 = conv.addMessage("system", "You are a helpful assistant.");
			console.log("Added system message:", msgId1);
			
			const msgId2 = conv.addMessage("user", "Hello, can you help me?");
			console.log("Added user message:", msgId2);
			
			const msgId3 = conv.addMessage("assistant", "Of course! What can I help you with?");
			console.log("Added assistant message:", msgId3);
			
			// Test tool use
			const toolId = "search-123";
			const toolUseId = conv.addToolUse(toolId, "search", { query: "test query" });
			console.log("Added tool use:", toolUseId);
			
			const toolResultId = conv.addToolResult(toolId, "Found results for test query");
			console.log("Added tool result:", toolResultId);
			
			// Test getting messages
			const messages = conv.getMessages();
			console.log("All messages:", JSON.stringify(messages, null, 2));
			
			// Test getting single prompt
			const prompt = conv.getSinglePrompt();
			console.log("Single prompt:", prompt);
			
			// Test converting back to Go
			const goConv = conv.toGoConversation();
			console.log("Converted back to Go conversation");
			
			console.log("Conversation test complete");
		}
		
		runConversationTest().catch(console.error);
	`

	_, err = vm.RunString(conversationTestJS)
	cobra.CheckErr(err)
}
