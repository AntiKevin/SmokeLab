package logs

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	// DefaultLogPageSize is used when callers omit a page size.
	DefaultLogPageSize = 50
	// MaxLogPageSize bounds database work performed by one list request.
	MaxLogPageSize = 100
)

// SortField identifies a persisted date field that can order log results.
type SortField string

const (
	SortByTimestamp  SortField = "timestamp"
	SortByCapturedAt SortField = "captured_at"
)

// SortDirection controls whether older or newer records are returned first.
type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

// LogSource is the exact persisted identity of a log source.
type LogSource struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	ID   string `json:"id"`
}

// LogFilter selects records for a list operation. Sources are combined with OR;
// all other populated fields are combined with AND.
type LogFilter struct {
	Search       string      `json:"search,omitempty"`
	Levels       []string    `json:"levels"`
	Applications []string    `json:"applications"`
	Sources      []LogSource `json:"sources"`
	From         *time.Time  `json:"from,omitempty"`
	To           *time.Time  `json:"to,omitempty"`
}

// ListLogsRequest configures one deterministic page of persisted logs.
// Page numbering starts at one.
type ListLogsRequest struct {
	Filter        LogFilter     `json:"filter"`
	Page          int           `json:"page"`
	PageSize      int           `json:"pageSize"`
	SortBy        SortField     `json:"sortBy"`
	SortDirection SortDirection `json:"sortDirection"`
}

// LogRecord contains all metadata persisted for a structured log.
type LogRecord struct {
	ID          int64     `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Application string    `json:"application"`
	Source      LogSource `json:"source"`
	LineNumber  int       `json:"lineNumber"`
	CapturedAt  time.Time `json:"capturedAt"`
	// Params is a validated, compact JSON object encoded as text. Keeping the
	// JSON text intact prevents JavaScript bindings from silently rounding
	// integer values that exceed the language's safe numeric range.
	Params string `json:"params"`
}

// LogPage is one page together with the total matching the same filters.
type LogPage struct {
	Items      []LogRecord `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

// LevelCount is the number of persisted records for one exact level.
type LevelCount struct {
	Level string `json:"level"`
	Count int64  `json:"count"`
}

// LogOverview summarizes all persisted logs without loading them into memory.
type LogOverview struct {
	Total           int64        `json:"total"`
	ByLevel         []LevelCount `json:"byLevel"`
	Applications    []string     `json:"applications"`
	Sources         []LogSource  `json:"sources"`
	OldestTimestamp *time.Time   `json:"oldestTimestamp,omitempty"`
	NewestTimestamp *time.Time   `json:"newestTimestamp,omitempty"`
}

// NormalizeListLogsRequest cleans empty filters and applies safe pagination and
// ordering defaults. Invalid dates and ordering options are rejected.
func NormalizeListLogsRequest(request ListLogsRequest) (ListLogsRequest, error) {
	request.Filter.Search = strings.TrimSpace(request.Filter.Search)
	request.Filter.Levels = normalizeLevels(request.Filter.Levels)
	request.Filter.Applications = normalizeApplications(request.Filter.Applications)
	request.Filter.Sources = normalizeSources(request.Filter.Sources)

	if request.Filter.From != nil {
		from := request.Filter.From.UTC()
		request.Filter.From = &from
	}
	if request.Filter.To != nil {
		to := request.Filter.To.UTC()
		request.Filter.To = &to
	}
	if request.Filter.From != nil && request.Filter.To != nil && request.Filter.From.After(*request.Filter.To) {
		return ListLogsRequest{}, errors.New("log filter start must not be after end")
	}

	if request.Page <= 0 {
		request.Page = 1
	}
	if request.PageSize <= 0 {
		request.PageSize = DefaultLogPageSize
	} else if request.PageSize > MaxLogPageSize {
		request.PageSize = MaxLogPageSize
	}
	if request.Page-1 > math.MaxInt/request.PageSize {
		return ListLogsRequest{}, errors.New("log page is too large")
	}

	request.SortBy = SortField(strings.ToLower(strings.TrimSpace(string(request.SortBy))))
	if request.SortBy == "" {
		request.SortBy = SortByTimestamp
	}
	if request.SortBy != SortByTimestamp && request.SortBy != SortByCapturedAt {
		return ListLogsRequest{}, fmt.Errorf("unsupported log sort field %q", request.SortBy)
	}

	request.SortDirection = SortDirection(strings.ToLower(strings.TrimSpace(string(request.SortDirection))))
	if request.SortDirection == "" {
		request.SortDirection = SortDescending
	}
	if request.SortDirection != SortAscending && request.SortDirection != SortDescending {
		return ListLogsRequest{}, fmt.Errorf("unsupported log sort direction %q", request.SortDirection)
	}

	return request, nil
}

func normalizeLevels(levels []string) []string {
	result := make([]string, 0, len(levels))
	seen := make(map[string]struct{}, len(levels))
	for _, level := range levels {
		level = strings.TrimSpace(level)
		if level == "" {
			continue
		}
		if _, exists := seen[level]; exists {
			continue
		}
		seen[level] = struct{}{}
		result = append(result, level)
	}
	return result
}

func normalizeApplications(applications []string) []string {
	result := make([]string, 0, len(applications))
	seen := make(map[string]struct{}, len(applications))
	for _, application := range applications {
		application = strings.TrimSpace(application)
		if application == "" {
			continue
		}
		if _, exists := seen[application]; exists {
			continue
		}
		seen[application] = struct{}{}
		result = append(result, application)
	}
	return result
}

func normalizeSources(sources []LogSource) []LogSource {
	result := make([]LogSource, 0, len(sources))
	seen := make(map[LogSource]struct{}, len(sources))
	for _, source := range sources {
		source.Kind = strings.TrimSpace(source.Kind)
		source.Name = strings.TrimSpace(source.Name)
		source.ID = strings.TrimSpace(source.ID)
		if source == (LogSource{}) {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		result = append(result, source)
	}
	return result
}
