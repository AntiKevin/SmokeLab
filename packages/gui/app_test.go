package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"SmokeLab/packages/engine/logs"
)

type cancelAwareReader struct {
	mu       sync.Mutex
	calls    int
	started  chan struct{}
	canceled chan struct{}
	blockAll bool
}

func (r *cancelAwareReader) List(ctx context.Context, request logs.ListLogsRequest) (logs.LogPage, error) {
	r.mu.Lock()
	r.calls++
	call := r.calls
	r.mu.Unlock()

	if call == 1 {
		close(r.started)
	}
	if r.blockAll || call == 1 {
		<-ctx.Done()
		close(r.canceled)
		return logs.LogPage{}, ctx.Err()
	}

	return logs.LogPage{Items: []logs.LogRecord{}, Page: 1, PageSize: 50}, nil
}

func (*cancelAwareReader) Overview(context.Context) (logs.LogOverview, error) {
	return logs.LogOverview{}, nil
}

func TestListLogsCancelsPreviousRequest(t *testing.T) {
	reader := &cancelAwareReader{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
	app := &App{
		ctx:            context.Background(),
		logReadService: logs.NewReadService(reader),
	}
	firstResult := make(chan error, 1)

	go func() {
		_, err := app.ListLogs(logs.ListLogsRequest{})
		firstResult <- err
	}()

	waitForSignal(t, reader.started, "first list request to start")
	if _, err := app.ListLogs(logs.ListLogsRequest{}); err != nil {
		t.Fatalf("second ListLogs() error = %v", err)
	}
	waitForSignal(t, reader.canceled, "first list request to be canceled")

	select {
	case err := <-firstResult:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("first ListLogs() error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first ListLogs() did not return after cancellation")
	}
}

func TestShutdownCancelsActiveListAndDisablesService(t *testing.T) {
	reader := &cancelAwareReader{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
		blockAll: true,
	}
	app := &App{
		ctx:            context.Background(),
		logReadService: logs.NewReadService(reader),
	}
	result := make(chan error, 1)

	go func() {
		_, err := app.ListLogs(logs.ListLogsRequest{})
		result <- err
	}()

	waitForSignal(t, reader.started, "list request to start")
	app.shutdown(context.Background())
	waitForSignal(t, reader.canceled, "list request to be canceled by shutdown")

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ListLogs() error after shutdown = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ListLogs() did not return after shutdown")
	}

	if _, err := app.ListLogs(logs.ListLogsRequest{}); err == nil {
		t.Fatal("ListLogs() after shutdown returned no error")
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}
