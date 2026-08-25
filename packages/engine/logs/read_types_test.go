package logs

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNormalizeListLogsRequestAppliesDefaultsAndCleansFilters(t *testing.T) {
	t.Parallel()

	localFrom := time.Date(2026, time.August, 23, 9, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	request, err := NormalizeListLogsRequest(ListLogsRequest{
		Filter: LogFilter{
			Search:       "  database unavailable  ",
			Levels:       []string{" info ", "", "info", " error "},
			Applications: []string{" api ", "", "api", "worker"},
			Sources: []LogSource{
				{Kind: " file ", Name: " app.log ", ID: " one "},
				{},
				{Kind: "file", Name: "app.log", ID: "one"},
			},
			From: &localFrom,
		},
		Page:          -2,
		PageSize:      MaxLogPageSize + 20,
		SortBy:        " TIMESTAMP ",
		SortDirection: " DESC ",
	})
	if err != nil {
		t.Fatalf("NormalizeListLogsRequest returned error: %v", err)
	}

	if request.Page != 1 || request.PageSize != MaxLogPageSize {
		t.Fatalf("normalized pagination = (%d, %d), want (1, %d)", request.Page, request.PageSize, MaxLogPageSize)
	}
	if request.SortBy != SortByTimestamp || request.SortDirection != SortDescending {
		t.Fatalf("normalized ordering = (%q, %q)", request.SortBy, request.SortDirection)
	}
	if request.Filter.Search != "database unavailable" {
		t.Fatalf("normalized search = %q", request.Filter.Search)
	}
	if !reflect.DeepEqual(request.Filter.Levels, []string{"info", "error"}) {
		t.Fatalf("normalized levels = %#v", request.Filter.Levels)
	}
	if !reflect.DeepEqual(request.Filter.Applications, []string{"api", "worker"}) {
		t.Fatalf("normalized applications = %#v", request.Filter.Applications)
	}
	wantSources := []LogSource{{Kind: "file", Name: "app.log", ID: "one"}}
	if !reflect.DeepEqual(request.Filter.Sources, wantSources) {
		t.Fatalf("normalized sources = %#v, want %#v", request.Filter.Sources, wantSources)
	}
	if request.Filter.From == nil || request.Filter.From.Location() != time.UTC || !request.Filter.From.Equal(localFrom) {
		t.Fatalf("normalized from = %v, want UTC equivalent", request.Filter.From)
	}
}

func TestNormalizeListLogsRequestUsesAllDefaults(t *testing.T) {
	t.Parallel()

	request, err := NormalizeListLogsRequest(ListLogsRequest{})
	if err != nil {
		t.Fatalf("NormalizeListLogsRequest returned error: %v", err)
	}
	if request.Page != 1 || request.PageSize != DefaultLogPageSize || request.SortBy != SortByTimestamp || request.SortDirection != SortDescending {
		t.Fatalf("defaults not applied: %#v", request)
	}
	if request.Filter.Levels == nil || request.Filter.Applications == nil || request.Filter.Sources == nil {
		t.Fatalf("normalized filter arrays must not be nil: %#v", request.Filter)
	}
}

func TestNormalizeListLogsRequestRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC)
	to := from.Add(-time.Nanosecond)
	tests := []struct {
		name    string
		request ListLogsRequest
	}{
		{name: "reversed dates", request: ListLogsRequest{Filter: LogFilter{From: &from, To: &to}}},
		{name: "sort field", request: ListLogsRequest{SortBy: "message"}},
		{name: "sort direction", request: ListLogsRequest{SortDirection: "sideways"}},
		{name: "page overflow", request: ListLogsRequest{Page: int(^uint(0) >> 1), PageSize: 2}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NormalizeListLogsRequest(test.request); err == nil {
				t.Fatal("NormalizeListLogsRequest should reject invalid request")
			}
		})
	}
}

type readStub struct {
	request  ListLogsRequest
	page     LogPage
	overview LogOverview
	err      error
}

func (*readStub) HighlightConfiguration(context.Context) ([]ApplicationHighlight, error) {
	return []ApplicationHighlight{}, nil
}

func (*readStub) HighlightSettings(context.Context, []string) ([]HighlightSetting, error) {
	return []HighlightSetting{}, nil
}

func (*readStub) SaveHighlightSettings(context.Context, []HighlightSetting) error {
	return nil
}

func (s *readStub) List(_ context.Context, request ListLogsRequest) (LogPage, error) {
	s.request = request
	return s.page, s.err
}

func (s *readStub) Overview(context.Context) (LogOverview, error) {
	return s.overview, s.err
}

func TestReadServiceNormalizesAndDelegates(t *testing.T) {
	t.Parallel()

	repository := &readStub{page: LogPage{Items: []LogRecord{}}}
	service := NewReadService(repository)
	if _, err := service.List(context.Background(), ListLogsRequest{Filter: LogFilter{Levels: []string{" info ", ""}}}); err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if repository.request.Page != 1 || repository.request.PageSize != DefaultLogPageSize ||
		!reflect.DeepEqual(repository.request.Filter.Levels, []string{"info"}) {
		t.Fatalf("delegated request was not normalized: %#v", repository.request)
	}
}

func TestReadServicePreservesContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewReadService(&readStub{})
	if _, err := service.List(ctx, ListLogsRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("List error = %v, want context.Canceled", err)
	}
	if _, err := service.Overview(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Overview error = %v, want context.Canceled", err)
	}
}
