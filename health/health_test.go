package health

import (
	"context"
	"sync"
	"testing"
	"time"
)

func resetTimestamps() {
	timestampMutex.Lock()
	latestNonErrorTimestamp = time.Time{}
	latestErrorTimestamp = time.Time{}
	latestBusyTimestamp = time.Time{}
	timestampMutex.Unlock()
}

func TestIsAlive_InitialState(t *testing.T) {
	resetTimestamps()
	alive, fields := IsAlive()
	// Both timestamps are zero, so nonError.Before(error) is false → alive.
	if !alive {
		t.Fatal("expected alive on initial state")
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
}

func TestIsAlive_AfterNonError(t *testing.T) {
	resetTimestamps()
	now := time.Now()
	UpdateLatestTimestamps(&now, nil, nil)
	alive, _ := IsAlive()
	if !alive {
		t.Fatal("expected alive after non-error")
	}
}

func TestIsAlive_AfterError(t *testing.T) {
	resetTimestamps()
	nonErr := time.Now()
	UpdateLatestTimestamps(&nonErr, nil, nil)
	time.Sleep(time.Millisecond)
	errTime := time.Now()
	UpdateLatestTimestamps(nil, &errTime, nil)
	alive, _ := IsAlive()
	if alive {
		t.Fatal("expected not alive when error is newer than non-error")
	}
}

func TestIsAlive_RecoverAfterError(t *testing.T) {
	resetTimestamps()
	errTime := time.Now()
	UpdateLatestTimestamps(nil, &errTime, nil)
	time.Sleep(time.Millisecond)
	nonErr := time.Now()
	UpdateLatestTimestamps(&nonErr, nil, nil)
	alive, _ := IsAlive()
	if !alive {
		t.Fatal("expected alive after recovery")
	}
}

func TestIsReady_InitialState(t *testing.T) {
	resetTimestamps()
	ready, fields := IsReady()
	// Zero busy timestamp + rememberBusyTimeout is still in the past → ready.
	if !ready {
		t.Fatal("expected ready on initial state")
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
}

func TestIsReady_AfterBusy(t *testing.T) {
	resetTimestamps()
	now := time.Now()
	UpdateLatestTimestamps(nil, nil, &now)
	ready, _ := IsReady()
	// Just became busy; rememberBusyTimeout hasn't elapsed → not ready.
	if ready {
		t.Fatal("expected not ready immediately after busy")
	}
}

func TestUpdateLatestTimestamps_OnlyAdvances(t *testing.T) {
	resetTimestamps()
	t1 := time.Now()
	t2 := t1.Add(-time.Hour) // Older timestamp.
	UpdateLatestTimestamps(&t1, nil, nil)
	UpdateLatestTimestamps(&t2, nil, nil) // Should not go backwards.
	timestampMutex.RLock()
	got := latestNonErrorTimestamp
	timestampMutex.RUnlock()
	if !got.Equal(t1) {
		t.Fatalf("timestamp went backwards: got %v, want %v", got, t1)
	}
}

func TestUpdateLatestTimestamps_AllNil(t *testing.T) {
	resetTimestamps()
	UpdateLatestTimestamps(nil, nil, nil) // Should not panic.
}

func TestUpdateLatestTimestamps_Concurrent(t *testing.T) {
	resetTimestamps()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			now := time.Now()
			UpdateLatestTimestamps(&now, &now, &now)
		}()
	}
	wg.Wait()
	alive, _ := IsAlive()
	if !alive {
		t.Fatal("expected alive after concurrent identical updates")
	}
}

func TestCompleteRequest_DoneBeforeDeadline(t *testing.T) {
	resetTimestamps()
	doneChan := make(chan int, 1)
	doneChan <- 42
	// Use a context that won't expire.
	ctx, cancel := newTestContext(time.Hour)
	defer cancel()
	status := CompleteRequest(ctx, doneChan)
	if status != 42 {
		t.Fatalf("got %d, want 42", status)
	}
}

func TestCompleteRequest_DeadlineBeforeDone(t *testing.T) {
	resetTimestamps()
	doneChan := make(chan int, 1)
	// Use a context that's already expired.
	ctx, cancel := newTestContext(-time.Millisecond)
	defer cancel()
	// Yield to let the context expire.
	time.Sleep(time.Millisecond)
	status := CompleteRequest(ctx, doneChan)
	if status != -1 {
		t.Fatalf("got %d, want -1", status)
	}
}

func newTestContext(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), time.Now().Add(d))
}
