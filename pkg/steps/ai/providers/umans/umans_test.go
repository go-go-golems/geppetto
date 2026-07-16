package umans

import "testing"

func TestClaudeOptionsRequireKey(t *testing.T) {
	if _, err := ClaudeOptions(""); err == nil {
		t.Fatal("empty Umans key unexpectedly accepted")
	}
	options, err := ClaudeOptions("test-key")
	if err != nil || len(options) != 1 {
		t.Fatalf("ClaudeOptions = %d, %v", len(options), err)
	}
}
