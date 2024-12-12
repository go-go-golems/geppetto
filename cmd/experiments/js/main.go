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

	// Add run command that executes a JS test
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run JavaScript Steps test",
		Run: func(cmd *cobra.Command, args []string) {
			parsedLayers, err := parser.Parse(cmd, nil)
			cobra.CheckErr(err)

			err = stepSettings.UpdateFromParsedLayers(parsedLayers)
			cobra.CheckErr(err)

			// Create event loop
			loop := eventloop.NewEventLoop()

			// Create a simple test step that doubles numbers with delay
			doubleStep := &utils.LambdaStep[float64, float64]{
				Function: func(input float64) helpers.Result[float64] {
					fmt.Println("Starting doubleStep")
					// Simulate async work
					time.Sleep(500 * time.Millisecond)
					fmt.Println("Finished doubleStep")
					return helpers.NewValueResult(input * 2)
				},
			}

			// Run test JavaScript code
			testJS := `
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
				async function runTests() {
					await testPromise();
					testBlocking();
					testCallbacks();
					console.log("All tests complete");
				}

				runTests().catch(console.error);
			`

			loop.Start()
			defer loop.Stop()
			// Register step and console in event loop
			loop.RunOnLoop(func(vm *goja.Runtime) {
				// Register the step in JS
				js.RegisterStep(
					vm,
					loop,
					"doubleStep",
					doubleStep,
					func(v goja.Value) float64 {
						return v.ToFloat()
					},
					func(v float64) goja.Value {
						return vm.ToValue(v)
					})

				// Create console object for logging
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
				vm.Set("console", console)

				// Run the test program
				_, err = vm.RunString(testJS)
				cobra.CheckErr(err)

			})
			time.Sleep(10 * time.Second)
		},
	}

	err = clay.InitViper("geppetto", runCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	err = parser.AddToCobraCommand(runCmd)
	cobra.CheckErr(err)

	err = runCmd.Execute()
	cobra.CheckErr(err)
}
