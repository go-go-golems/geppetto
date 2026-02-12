//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

func main() {
	loop := eventloop.NewEventLoop()
	go loop.Start()
	defer loop.StopNoWait()

	ready := make(chan struct{})
	var setupErr error

	loop.RunOnLoop(func(vm *goja.Runtime) {
		vm.Set("inferAsync", func(prompt string) *goja.Promise {
			p, resolve, reject := vm.NewPromise()
			go func() {
				time.Sleep(20 * time.Millisecond)
				loop.RunOnLoop(func(_ *goja.Runtime) {
					if err := resolve(vm.ToValue("ok:" + prompt)); err != nil {
						_ = reject(vm.ToValue(err.Error()))
					}
				})
			}()
			return p
		})

		vm.Set("inferWithCancel", func(prompt string) map[string]any {
			ctx, cancel := context.WithCancel(context.Background())
			p, resolve, reject := vm.NewPromise()
			go func() {
				select {
				case <-ctx.Done():
					loop.RunOnLoop(func(_ *goja.Runtime) {
						_ = reject(vm.ToValue("canceled:" + prompt))
					})
				case <-time.After(40 * time.Millisecond):
					loop.RunOnLoop(func(_ *goja.Runtime) {
						if err := resolve(vm.ToValue("ok:" + prompt)); err != nil {
							_ = reject(vm.ToValue(err.Error()))
						}
					})
				}
			}()

			return map[string]any{
				"promise": p,
				"cancel": func() {
					cancel()
				},
			}
		})

		_, setupErr = vm.RunString(`
var events = [];
inferAsync("alpha")
  .then(v => events.push("resolve:" + v))
  .catch(e => events.push("reject:" + e));

const pending = inferWithCancel("beta");
pending.promise
  .then(v => events.push("resolve:" + v))
  .catch(e => events.push("reject:" + e));
pending.cancel();
`)
		close(ready)
	})

	<-ready
	if setupErr != nil {
		panic(setupErr)
	}

	time.Sleep(120 * time.Millisecond)

	type result struct {
		events []any
		err    error
	}
	ch := make(chan result, 1)
	loop.RunOnLoop(func(vm *goja.Runtime) {
		v := vm.Get("events")
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			ch <- result{err: fmt.Errorf("events not set")}
			return
		}
		arr, ok := v.Export().([]any)
		if !ok {
			ch <- result{err: fmt.Errorf("events is not []any (%T)", v.Export())}
			return
		}
		ch <- result{events: arr}
	})

	res := <-ch
	if res.err != nil {
		panic(res.err)
	}

	fmt.Println("=== Goja Eventloop Async Experiment ===")
	for i, e := range res.events {
		fmt.Printf("event[%d]=%v\n", i, e)
	}
}
