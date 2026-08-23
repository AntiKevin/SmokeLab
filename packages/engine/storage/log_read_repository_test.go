package storage

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"SmokeLab/packages/engine/logs"
)

func TestLogReadRepositoryPagesWithStableNanosecondOrdering(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:00Z", message: "whole second"})
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:00.1Z", message: "first tie"})
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:00.100000000Z", message: "second tie"})
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:00.9Z", message: "latest fraction"})

	first, err := repository.List(context.Background(), logs.ListLogsRequest{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("List first page returned error: %v", err)
	}
	assertMessages(t, first.Items, "latest fraction", "second tie")
	if first.Total != 4 || first.TotalPages != 2 || first.Page != 1 || first.PageSize != 2 {
		t.Fatalf("unexpected first page metadata: %#v", first)
	}

	second, err := repository.List(context.Background(), logs.ListLogsRequest{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("List second page returned error: %v", err)
	}
	assertMessages(t, second.Items, "first tie", "whole second")
	if second.Items[0].ID <= second.Items[1].ID {
		t.Fatalf("tie must use descending id: ids %d, %d", second.Items[0].ID, second.Items[1].ID)
	}

	ascending, err := repository.List(context.Background(), logs.ListLogsRequest{
		PageSize:      10,
		SortDirection: logs.SortAscending,
	})
	if err != nil {
		t.Fatalf("List ascending returned error: %v", err)
	}
	assertMessages(t, ascending.Items, "whole second", "first tie", "second tie", "latest fraction")
	if ascending.Items[1].ID >= ascending.Items[2].ID {
		t.Fatalf("ascending tie must use ascending id: ids %d, %d", ascending.Items[1].ID, ascending.Items[2].ID)
	}
}

func TestLogReadRepositorySortsByCapturedAt(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:03Z", capturedAt: "2026-08-23T13:00:00Z", message: "captured first"})
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:01Z", capturedAt: "2026-08-23T13:00:00.5Z", message: "captured last"})

	page, err := repository.List(context.Background(), logs.ListLogsRequest{SortBy: logs.SortByCapturedAt})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	assertMessages(t, page.Items, "captured last", "captured first")
}

func TestLogOrderingQueriesUseExpressionIndexes(t *testing.T) {
	t.Parallel()

	db, _ := openReadRepository(t)
	tests := []struct {
		column string
		index  string
	}{
		{column: "timestamp", index: "logs_timestamp_order_idx"},
		{column: "captured_at", index: "logs_captured_at_order_idx"},
	}
	for _, test := range tests {
		t.Run(test.column, func(t *testing.T) {
			query := `EXPLAIN QUERY PLAN SELECT
                id, timestamp, level, message, source_kind, source_name, source_id,
                line_number, captured_at, params
                FROM logs ORDER BY ` + timestampKey(test.column) + ` DESC, id DESC LIMIT ? OFFSET ?`
			rows, err := db.Query(query, logs.DefaultLogPageSize, 0)
			if err != nil {
				t.Fatalf("explain ordering query: %v", err)
			}
			defer rows.Close()

			used := false
			for rows.Next() {
				var id, parent, unused int
				var detail string
				if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
					t.Fatalf("scan query plan: %v", err)
				}
				if strings.Contains(detail, test.index) {
					used = true
				}
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("iterate query plan: %v", err)
			}
			if !used {
				t.Fatalf("query plan did not use %s", test.index)
			}
		})
	}
}

func TestLogReadRepositoryAppliesCombinedFilters(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{
		timestamp: "2026-08-23T12:00:00Z", level: "error", message: "DATABASE unavailable",
		source: logs.LogSource{Kind: "file", Name: "app.log", ID: "one"},
	})
	insertStoredLog(t, db, storedLog{
		timestamp: "2026-08-23T12:00:01Z", level: "info", message: "database recovered",
		source: logs.LogSource{Kind: "file", Name: "app.log", ID: "one"},
	})
	insertStoredLog(t, db, storedLog{
		timestamp: "2026-08-23T12:00:00Z", level: "error", message: "database unavailable",
		source: logs.LogSource{Kind: "stdin", Name: "terminal", ID: "two"},
	})

	from := mustTime(t, "2026-08-23T12:00:00Z")
	to := mustTime(t, "2026-08-23T12:00:00Z")
	page, err := repository.List(context.Background(), logs.ListLogsRequest{
		Filter: logs.LogFilter{
			Search:  "database UNAVAILABLE",
			Levels:  []string{"error"},
			Sources: []logs.LogSource{{Kind: "file", Name: "app.log", ID: "one"}},
			From:    &from,
			To:      &to,
		},
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	assertMessages(t, page.Items, "DATABASE unavailable")
	if page.Total != 1 {
		t.Fatalf("filtered total = %d, want 1", page.Total)
	}
}

func TestLogReadRepositoryFiltersByMultipleLevels(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{level: "debug", message: "debug"})
	insertStoredLog(t, db, storedLog{level: "info", message: "info"})
	insertStoredLog(t, db, storedLog{level: "warn", message: "warn"})
	insertStoredLog(t, db, storedLog{level: "error", message: "error"})

	page, err := repository.List(context.Background(), logs.ListLogsRequest{
		Filter:        logs.LogFilter{Levels: []string{"warn", "error"}},
		SortDirection: logs.SortAscending,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	assertMessages(t, page.Items, "warn", "error")
	if page.Total != 2 {
		t.Fatalf("filtered total = %d, want 2", page.Total)
	}
}

func TestLogReadRepositoryTreatsSearchWildcardsLiterally(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{message: "progress 100%_done"})
	insertStoredLog(t, db, storedLog{message: "progress 100Xdone"})

	page, err := repository.List(context.Background(), logs.ListLogsRequest{Filter: logs.LogFilter{Search: "%_"}})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	assertMessages(t, page.Items, "progress 100%_done")
}

func TestLogReadRepositoryMatchesExactSourceTuplesWithOR(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{message: "first", source: logs.LogSource{Kind: "file", Name: "a", ID: "1"}})
	insertStoredLog(t, db, storedLog{message: "second", source: logs.LogSource{Kind: "file", Name: "b", ID: "2"}})
	insertStoredLog(t, db, storedLog{message: "cross product", source: logs.LogSource{Kind: "file", Name: "a", ID: "2"}})

	page, err := repository.List(context.Background(), logs.ListLogsRequest{
		Filter: logs.LogFilter{Sources: []logs.LogSource{
			{Kind: "file", Name: "a", ID: "1"},
			{Kind: "file", Name: "b", ID: "2"},
		}},
		SortDirection: logs.SortAscending,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	assertMessages(t, page.Items, "first", "second")
}

func TestLogReadRepositoryDateRangeIsInclusiveAtNanosecondPrecision(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:00Z", message: "before"})
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:00.1Z", message: "lower boundary"})
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:00.100000001Z", message: "upper boundary"})
	insertStoredLog(t, db, storedLog{timestamp: "2026-08-23T12:00:00.2Z", message: "after"})

	from := mustTime(t, "2026-08-23T09:00:00.1-03:00")
	to := mustTime(t, "2026-08-23T12:00:00.100000001Z")
	page, err := repository.List(context.Background(), logs.ListLogsRequest{
		Filter:        logs.LogFilter{From: &from, To: &to},
		SortDirection: logs.SortAscending,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	assertMessages(t, page.Items, "lower boundary", "upper boundary")
}

func TestLogReadRepositoryReadsCompleteMetadataAndJSONNumbers(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{
		timestamp:  "2026-08-23T12:34:56.123456789Z",
		capturedAt: "2026-08-23T12:35:00.987654321Z",
		level:      "warn",
		message:    "metadata",
		source:     logs.LogSource{Kind: "file", Name: "service.log", ID: "source-7"},
		lineNumber: 42,
		params:     `{ "large": 9007199254740993, "nested": { "retry": true }, "tags": ["a"] }`,
	})

	page, err := repository.List(context.Background(), logs.ListLogsRequest{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(page.Items))
	}
	record := page.Items[0]
	if record.ID == 0 || record.Level != "warn" || record.Message != "metadata" || record.LineNumber != 42 ||
		record.Source != (logs.LogSource{Kind: "file", Name: "service.log", ID: "source-7"}) {
		t.Fatalf("unexpected metadata: %#v", record)
	}
	if !record.Timestamp.Equal(mustTime(t, "2026-08-23T12:34:56.123456789Z")) ||
		!record.CapturedAt.Equal(mustTime(t, "2026-08-23T12:35:00.987654321Z")) {
		t.Fatalf("unexpected record times: %s, %s", record.Timestamp, record.CapturedAt)
	}
	wantParams := `{"large":9007199254740993,"nested":{"retry":true},"tags":["a"]}`
	if record.Params != wantParams {
		t.Fatalf("params = %q, want lossless JSON %q", record.Params, wantParams)
	}
}

func TestLogReadRepositoryReturnsCompactEmptyParams(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{params: `{  }`})
	page, err := repository.List(context.Background(), logs.ListLogsRequest{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Params != `{}` {
		t.Fatalf("empty params = %#v, want compact JSON object", page.Items)
	}
}

func TestLogReadRepositoryRejectsInvalidOrNonObjectParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params string
	}{
		{name: "invalid", params: `{"broken":`},
		{name: "array", params: `[]`},
		{name: "null", params: `null`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			db, repository := openReadRepository(t)
			if test.name == "invalid" {
				if _, err := db.Exec("PRAGMA ignore_check_constraints = ON"); err != nil {
					t.Fatalf("disable check constraints: %v", err)
				}
			}
			insertStoredLog(t, db, storedLog{params: test.params})
			if _, err := repository.List(context.Background(), logs.ListLogsRequest{}); err == nil || !strings.Contains(err.Error(), "params") {
				t.Fatalf("List error = %v, want params decoding error", err)
			}
		})
	}
}

func TestLogReadRepositoryReturnsEmptyNonNilCollections(t *testing.T) {
	t.Parallel()

	_, repository := openReadRepository(t)
	page, err := repository.List(context.Background(), logs.ListLogsRequest{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if page.Items == nil || len(page.Items) != 0 || page.Total != 0 || page.TotalPages != 0 || page.Page != 1 || page.PageSize != logs.DefaultLogPageSize {
		t.Fatalf("unexpected empty page: %#v", page)
	}

	overview, err := repository.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}
	if overview.Total != 0 || overview.ByLevel == nil || overview.Sources == nil ||
		len(overview.ByLevel) != 0 || len(overview.Sources) != 0 ||
		overview.OldestTimestamp != nil || overview.NewestTimestamp != nil {
		t.Fatalf("unexpected empty overview: %#v", overview)
	}
}

func TestLogReadRepositoryOverview(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{
		timestamp: "2026-08-23T12:00:00Z", level: "info", message: "oldest",
		source: logs.LogSource{Kind: "file", Name: "app.log", ID: "one"},
	})
	insertStoredLog(t, db, storedLog{
		timestamp: "2026-08-23T12:00:00.5Z", level: "error", message: "newest",
		source: logs.LogSource{Kind: "stdin", Name: "terminal", ID: "two"},
	})
	insertStoredLog(t, db, storedLog{
		timestamp: "2026-08-23T12:00:00.1Z", level: "info", message: "middle",
		source: logs.LogSource{Kind: "file", Name: "app.log", ID: "one"},
	})

	overview, err := repository.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}
	if overview.Total != 3 {
		t.Fatalf("overview total = %d, want 3", overview.Total)
	}
	wantLevels := []logs.LevelCount{{Level: "error", Count: 1}, {Level: "info", Count: 2}}
	if !reflect.DeepEqual(overview.ByLevel, wantLevels) {
		t.Fatalf("overview levels = %#v, want %#v", overview.ByLevel, wantLevels)
	}
	wantSources := []logs.LogSource{
		{Kind: "file", Name: "app.log", ID: "one"},
		{Kind: "stdin", Name: "terminal", ID: "two"},
	}
	if !reflect.DeepEqual(overview.Sources, wantSources) {
		t.Fatalf("overview sources = %#v, want %#v", overview.Sources, wantSources)
	}
	if overview.OldestTimestamp == nil || overview.NewestTimestamp == nil ||
		!overview.OldestTimestamp.Equal(mustTime(t, "2026-08-23T12:00:00Z")) ||
		!overview.NewestTimestamp.Equal(mustTime(t, "2026-08-23T12:00:00.5Z")) {
		t.Fatalf("overview range = (%v, %v)", overview.OldestTimestamp, overview.NewestTimestamp)
	}
}

func TestLogReadRepositoryHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := repository.List(ctx, logs.ListLogsRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("List error = %v, want context.Canceled", err)
	}
	if _, err := repository.Overview(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Overview error = %v, want context.Canceled", err)
	}
	if _, err := NewLogReadRepository(ctx, db); !errors.Is(err, context.Canceled) {
		t.Fatalf("NewLogReadRepository error = %v, want context.Canceled", err)
	}
}

func openReadRepository(t *testing.T) (*sql.DB, *LogReadRepository) {
	t.Helper()
	ctx := context.Background()
	db := openTestDatabase(t, ctx)
	repository, err := NewLogReadRepository(ctx, db)
	if err != nil {
		t.Fatalf("NewLogReadRepository returned error: %v", err)
	}
	return db, repository
}

type storedLog struct {
	timestamp  string
	level      string
	message    string
	source     logs.LogSource
	lineNumber int
	capturedAt string
	params     string
}

func insertStoredLog(t *testing.T, db *sql.DB, record storedLog) {
	t.Helper()
	if record.timestamp == "" {
		record.timestamp = "2026-08-23T12:00:00Z"
	}
	if record.level == "" {
		record.level = "info"
	}
	if record.message == "" {
		record.message = "message"
	}
	if record.source == (logs.LogSource{}) {
		record.source = logs.LogSource{Kind: "file", Name: "default.log", ID: "default"}
	}
	if record.lineNumber == 0 {
		record.lineNumber = 1
	}
	if record.capturedAt == "" {
		record.capturedAt = "2026-08-23T13:00:00Z"
	}
	if record.params == "" {
		record.params = `{}`
	}
	if _, err := db.Exec(`INSERT INTO logs (
        timestamp, level, message, source_kind, source_name, source_id,
        line_number, captured_at, params
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.timestamp,
		record.level,
		record.message,
		record.source.Kind,
		record.source.Name,
		record.source.ID,
		record.lineNumber,
		record.capturedAt,
		record.params,
	); err != nil {
		t.Fatalf("insert stored log: %v", err)
	}
}

func assertMessages(t *testing.T, records []logs.LogRecord, want ...string) {
	t.Helper()
	got := make([]string, len(records))
	for index, record := range records {
		got[index] = record.Message
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("messages = %#v, want %#v", got, want)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
