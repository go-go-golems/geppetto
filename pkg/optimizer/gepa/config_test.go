package gepa

import "testing"

func TestConfigWithDefaults(t *testing.T) {
	got := (Config{}).withDefaults()

	if got.MaxEvalCalls != 200 {
		t.Fatalf("expected MaxEvalCalls=200, got %d", got.MaxEvalCalls)
	}
	if got.BatchSize != 8 {
		t.Fatalf("expected BatchSize=8, got %d", got.BatchSize)
	}
	if got.FrontierSize != 10 {
		t.Fatalf("expected FrontierSize=10, got %d", got.FrontierSize)
	}
	if got.RandomSeed == 0 {
		t.Fatalf("expected RandomSeed to be set")
	}
	if got.ReflectionSystemPrompt == "" {
		t.Fatalf("expected ReflectionSystemPrompt to be set")
	}
	if got.ReflectionPromptTemplate == "" {
		t.Fatalf("expected ReflectionPromptTemplate to be set")
	}
	if got.Now == nil {
		t.Fatalf("expected Now function to be set")
	}
}
