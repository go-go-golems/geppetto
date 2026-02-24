package profiles

import "testing"

func TestProjectRuntimeMiddlewareConfigsToExtensions_RoundTripByID(t *testing.T) {
	runtime := &RuntimeSpec{
		Middlewares: []MiddlewareUse{
			{
				Name: "agentmode",
				ID:   "primary",
				Config: map[string]any{
					"default_mode": "chat",
				},
			},
		},
	}
	extensions := map[string]any{
		"app.note@v1": map[string]any{"enabled": true},
	}

	projected, err := ProjectRuntimeMiddlewareConfigsToExtensions(runtime, extensions)
	if err != nil {
		t.Fatalf("project runtime configs: %v", err)
	}
	if runtime.Middlewares[0].Config != nil {
		t.Fatalf("expected inline runtime middleware config to be cleared")
	}
	if _, ok := projected["app.note@v1"]; !ok {
		t.Fatalf("expected unrelated extension key to be preserved")
	}

	config, ok, err := MiddlewareConfigFromExtensions(projected, runtime.Middlewares[0], 0)
	if err != nil {
		t.Fatalf("lookup middleware config: %v", err)
	}
	if !ok {
		t.Fatalf("expected middleware config to be present in typed-key extensions")
	}
	if got, want := config["default_mode"], "chat"; got != want {
		t.Fatalf("middleware config mismatch: got=%#v want=%#v", got, want)
	}
}

func TestProjectRuntimeMiddlewareConfigsToExtensions_ByIndex(t *testing.T) {
	runtime := &RuntimeSpec{
		Middlewares: []MiddlewareUse{
			{
				Name: "agentmode",
				Config: map[string]any{
					"default_mode": "chat",
				},
			},
			{
				Name: "agentmode",
				Config: map[string]any{
					"default_mode": "safe",
				},
			},
		},
	}

	projected, err := ProjectRuntimeMiddlewareConfigsToExtensions(runtime, nil)
	if err != nil {
		t.Fatalf("project runtime configs: %v", err)
	}
	first, ok, err := MiddlewareConfigFromExtensions(projected, runtime.Middlewares[0], 0)
	if err != nil || !ok {
		t.Fatalf("lookup first middleware config: ok=%v err=%v", ok, err)
	}
	second, ok, err := MiddlewareConfigFromExtensions(projected, runtime.Middlewares[1], 1)
	if err != nil || !ok {
		t.Fatalf("lookup second middleware config: ok=%v err=%v", ok, err)
	}
	if got, want := first["default_mode"], "chat"; got != want {
		t.Fatalf("first middleware config mismatch: got=%#v want=%#v", got, want)
	}
	if got, want := second["default_mode"], "safe"; got != want {
		t.Fatalf("second middleware config mismatch: got=%#v want=%#v", got, want)
	}
}

func TestMiddlewareConfigExtensionKey_RejectsInvalidName(t *testing.T) {
	_, err := MiddlewareConfigExtensionKey("not valid!")
	if err == nil {
		t.Fatalf("expected invalid middleware name to fail")
	}
}
