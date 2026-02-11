package toolloop

import (
	"context"
	"testing"
	"time"
)

func TestStepController_WaitUnblocksOnContinue(t *testing.T) {
	t.Parallel()

	sc := NewStepController()
	sc.Enable(StepScope{SessionID: "s1"})

	pm, ok := sc.Pause(PauseMeta{SessionID: "s1", Phase: StepPhaseAfterInference, Summary: "x"})
	if !ok || pm.PauseID == "" {
		t.Fatalf("expected pause to be registered")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = sc.Wait(context.Background(), pm.PauseID, 5*time.Second)
	}()

	time.Sleep(10 * time.Millisecond)
	_, _ = sc.Continue(pm.PauseID)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("wait did not unblock")
	}
}

func TestStepController_WaitUnblocksOnCancel(t *testing.T) {
	t.Parallel()

	sc := NewStepController()
	sc.Enable(StepScope{SessionID: "s1"})

	pm, ok := sc.Pause(PauseMeta{SessionID: "s1", Phase: StepPhaseAfterInference, Summary: "x"})
	if !ok || pm.PauseID == "" {
		t.Fatalf("expected pause to be registered")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sc.Wait(ctx, pm.PauseID, 5*time.Second)
	if err == nil {
		t.Fatalf("expected ctx error")
	}

	// Ensure pause is cleaned up.
	if _, ok := sc.Lookup(pm.PauseID); ok {
		t.Fatalf("expected pause to be removed on cancel")
	}
}

func TestStepController_DisableSessionDrainsWaiters(t *testing.T) {
	t.Parallel()

	sc := NewStepController()
	sc.Enable(StepScope{SessionID: "s1"})

	pm, ok := sc.Pause(PauseMeta{SessionID: "s1", Phase: StepPhaseAfterInference, Summary: "x"})
	if !ok || pm.PauseID == "" {
		t.Fatalf("expected pause to be registered")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = sc.Wait(context.Background(), pm.PauseID, 5*time.Second)
	}()

	time.Sleep(10 * time.Millisecond)
	sc.DisableSession("s1")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("wait did not unblock after disable")
	}
}
