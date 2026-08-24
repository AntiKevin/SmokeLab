package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"SmokeLab/packages/engine/logs"
)

// These expressions are part of migration 3 and must remain immutable. A
// future change to persisted timestamp ordering needs a new migration.
const (
	logTimestampOrderExpressionV3  = `substr(timestamp, 1, 19) || '.' || CASE WHEN substr(timestamp, 20, 1) = '.' THEN substr(substr(timestamp, 21, instr(substr(timestamp, 21), 'Z') - 1) || '000000000', 1, 9) ELSE '000000000' END || 'Z'`
	logCapturedAtOrderExpressionV3 = `substr(captured_at, 1, 19) || '.' || CASE WHEN substr(captured_at, 20, 1) = '.' THEN substr(substr(captured_at, 21, instr(substr(captured_at, 21), 'Z') - 1) || '000000000', 1, 9) ELSE '000000000' END || 'Z'`
)

// LogReadRepository queries persisted logs without exposing SQL to interfaces.
type LogReadRepository struct {
	db *sql.DB
}

var _ logs.Reader = (*LogReadRepository)(nil)

// NewLogReadRepository applies pending migrations and creates a read repository.
func NewLogReadRepository(ctx context.Context, db *sql.DB) (*LogReadRepository, error) {
	if err := prepareLogDatabase(ctx, db, "log read repository"); err != nil {
		return nil, err
	}
	return &LogReadRepository{db: db}, nil
}

// List returns a bounded page and its matching total from one read transaction.
func (r *LogReadRepository) List(ctx context.Context, request logs.ListLogsRequest) (logs.LogPage, error) {
	if err := r.ready(ctx, "log list"); err != nil {
		return logs.LogPage{}, err
	}

	normalized, err := logs.NormalizeListLogsRequest(request)
	if err != nil {
		return logs.LogPage{}, err
	}
	predicates, arguments := buildLogPredicates(normalized.Filter)
	where := ""
	if len(predicates) > 0 {
		where = " WHERE " + strings.Join(predicates, " AND ")
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return logs.LogPage{}, fmt.Errorf("begin log list: %w", err)
	}
	defer tx.Rollback()

	var total int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM logs"+where, arguments...).Scan(&total); err != nil {
		return logs.LogPage{}, fmt.Errorf("count logs: %w", err)
	}

	page := logs.LogPage{
		Items:    make([]logs.LogRecord, 0, normalized.PageSize),
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}
	if total > 0 {
		page.TotalPages = int((total-1)/int64(normalized.PageSize) + 1)
	}

	sortExpression, direction := logOrder(normalized.SortBy, normalized.SortDirection)
	query := `SELECT
        id, timestamp, level, message, application, source_kind, source_name, source_id,
        line_number, captured_at, params
    FROM logs` + where + " ORDER BY " + sortExpression + " " + direction + ", id " + direction + " LIMIT ? OFFSET ?"
	queryArguments := append(append([]any(nil), arguments...), normalized.PageSize, (normalized.Page-1)*normalized.PageSize)
	rows, err := tx.QueryContext(ctx, query, queryArguments...)
	if err != nil {
		return logs.LogPage{}, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		record, err := scanLogRecord(rows)
		if err != nil {
			return logs.LogPage{}, err
		}
		page.Items = append(page.Items, record)
	}
	if err := rows.Err(); err != nil {
		return logs.LogPage{}, fmt.Errorf("iterate logs: %w", err)
	}
	if err := rows.Close(); err != nil {
		return logs.LogPage{}, fmt.Errorf("close log rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return logs.LogPage{}, fmt.Errorf("finish log list: %w", err)
	}
	return page, nil
}

// Overview summarizes the entire persisted log collection.
func (r *LogReadRepository) Overview(ctx context.Context) (logs.LogOverview, error) {
	if err := r.ready(ctx, "log overview"); err != nil {
		return logs.LogOverview{}, err
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return logs.LogOverview{}, fmt.Errorf("begin log overview: %w", err)
	}
	defer tx.Rollback()

	overview := logs.LogOverview{
		ByLevel:      make([]logs.LevelCount, 0),
		Applications: make([]string, 0),
		Sources:      make([]logs.LogSource, 0),
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM logs").Scan(&overview.Total); err != nil {
		return logs.LogOverview{}, fmt.Errorf("count all logs: %w", err)
	}

	levelRows, err := tx.QueryContext(ctx, "SELECT level, COUNT(*) FROM logs GROUP BY level ORDER BY level COLLATE NOCASE, level")
	if err != nil {
		return logs.LogOverview{}, fmt.Errorf("query log levels: %w", err)
	}
	for levelRows.Next() {
		var count logs.LevelCount
		if err := levelRows.Scan(&count.Level, &count.Count); err != nil {
			levelRows.Close()
			return logs.LogOverview{}, fmt.Errorf("scan log level: %w", err)
		}
		overview.ByLevel = append(overview.ByLevel, count)
	}
	if err := levelRows.Err(); err != nil {
		levelRows.Close()
		return logs.LogOverview{}, fmt.Errorf("iterate log levels: %w", err)
	}
	if err := levelRows.Close(); err != nil {
		return logs.LogOverview{}, fmt.Errorf("close log levels: %w", err)
	}

	applicationRows, err := tx.QueryContext(ctx, "SELECT DISTINCT application FROM logs ORDER BY application COLLATE NOCASE, application")
	if err != nil {
		return logs.LogOverview{}, fmt.Errorf("query log applications: %w", err)
	}
	for applicationRows.Next() {
		var application string
		if err := applicationRows.Scan(&application); err != nil {
			applicationRows.Close()
			return logs.LogOverview{}, fmt.Errorf("scan log application: %w", err)
		}
		overview.Applications = append(overview.Applications, application)
	}
	if err := applicationRows.Err(); err != nil {
		applicationRows.Close()
		return logs.LogOverview{}, fmt.Errorf("iterate log applications: %w", err)
	}
	if err := applicationRows.Close(); err != nil {
		return logs.LogOverview{}, fmt.Errorf("close log applications: %w", err)
	}

	sourceRows, err := tx.QueryContext(ctx, `SELECT DISTINCT source_kind, source_name, source_id
        FROM logs ORDER BY source_kind COLLATE NOCASE, source_name COLLATE NOCASE, source_id COLLATE NOCASE,
        source_kind, source_name, source_id`)
	if err != nil {
		return logs.LogOverview{}, fmt.Errorf("query log sources: %w", err)
	}
	for sourceRows.Next() {
		var source logs.LogSource
		if err := sourceRows.Scan(&source.Kind, &source.Name, &source.ID); err != nil {
			sourceRows.Close()
			return logs.LogOverview{}, fmt.Errorf("scan log source: %w", err)
		}
		overview.Sources = append(overview.Sources, source)
	}
	if err := sourceRows.Err(); err != nil {
		sourceRows.Close()
		return logs.LogOverview{}, fmt.Errorf("iterate log sources: %w", err)
	}
	if err := sourceRows.Close(); err != nil {
		return logs.LogOverview{}, fmt.Errorf("close log sources: %w", err)
	}

	if overview.Total > 0 {
		oldest, err := selectTimestampBoundary(ctx, tx, logs.SortAscending)
		if err != nil {
			return logs.LogOverview{}, err
		}
		newest, err := selectTimestampBoundary(ctx, tx, logs.SortDescending)
		if err != nil {
			return logs.LogOverview{}, err
		}
		overview.OldestTimestamp = &oldest
		overview.NewestTimestamp = &newest
	}

	if err := tx.Commit(); err != nil {
		return logs.LogOverview{}, fmt.Errorf("finish log overview: %w", err)
	}
	return overview, nil
}

func (r *LogReadRepository) ready(ctx context.Context, operation string) error {
	if ctx == nil {
		return fmt.Errorf("%s context is required", operation)
	}
	if r == nil || r.db == nil {
		return errors.New("log read repository is not initialized")
	}
	return ctx.Err()
}

func buildLogPredicates(filter logs.LogFilter) ([]string, []any) {
	predicates := make([]string, 0, 6)
	arguments := make([]any, 0, len(filter.Levels)+len(filter.Applications)+len(filter.Sources)*3+3)

	if filter.Search != "" {
		predicates = append(predicates, `message LIKE ? ESCAPE '\' COLLATE NOCASE`)
		arguments = append(arguments, "%"+escapeLike(filter.Search)+"%")
	}
	if len(filter.Levels) > 0 {
		placeholders := make([]string, len(filter.Levels))
		for index, level := range filter.Levels {
			placeholders[index] = "?"
			arguments = append(arguments, level)
		}
		predicates = append(predicates, "level IN ("+strings.Join(placeholders, ", ")+")")
	}
	if len(filter.Applications) > 0 {
		placeholders := make([]string, len(filter.Applications))
		for index, application := range filter.Applications {
			placeholders[index] = "?"
			arguments = append(arguments, application)
		}
		predicates = append(predicates, "application IN ("+strings.Join(placeholders, ", ")+")")
	}
	if len(filter.Sources) > 0 {
		sources := make([]string, 0, len(filter.Sources))
		for _, source := range filter.Sources {
			sources = append(sources, "(source_kind = ? AND source_name = ? AND source_id = ?)")
			arguments = append(arguments, source.Kind, source.Name, source.ID)
		}
		predicates = append(predicates, "("+strings.Join(sources, " OR ")+")")
	}
	if filter.From != nil {
		predicates = append(predicates, timestampKey("timestamp")+" >= ?")
		arguments = append(arguments, timestampArgument(*filter.From))
	}
	if filter.To != nil {
		predicates = append(predicates, timestampKey("timestamp")+" <= ?")
		arguments = append(arguments, timestampArgument(*filter.To))
	}
	return predicates, arguments
}

func logOrder(field logs.SortField, direction logs.SortDirection) (string, string) {
	column := "timestamp"
	if field == logs.SortByCapturedAt {
		column = "captured_at"
	}
	directionSQL := "DESC"
	if direction == logs.SortAscending {
		directionSQL = "ASC"
	}
	return timestampKey(column), directionSQL
}

// timestampKey converts the UTC RFC3339Nano text emitted by Store into a fixed
// width key. This preserves nanosecond order when one value omits a fraction and
// another uses a variable-length fraction. column is selected only by logOrder.
func timestampKey(column string) string {
	if column == "captured_at" {
		return logCapturedAtOrderExpressionV3
	}
	return logTimestampOrderExpressionV3
}

func timestampArgument(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.000000000Z")
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

type rowScanner interface {
	Scan(...any) error
}

func scanLogRecord(scanner rowScanner) (logs.LogRecord, error) {
	var record logs.LogRecord
	var timestampText string
	var capturedAtText string
	var paramsText string
	if err := scanner.Scan(
		&record.ID,
		&timestampText,
		&record.Level,
		&record.Message,
		&record.Application,
		&record.Source.Kind,
		&record.Source.Name,
		&record.Source.ID,
		&record.LineNumber,
		&capturedAtText,
		&paramsText,
	); err != nil {
		return logs.LogRecord{}, fmt.Errorf("scan log record: %w", err)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, timestampText)
	if err != nil {
		return logs.LogRecord{}, fmt.Errorf("parse timestamp for log %d: %w", record.ID, err)
	}
	capturedAt, err := time.Parse(time.RFC3339Nano, capturedAtText)
	if err != nil {
		return logs.LogRecord{}, fmt.Errorf("parse captured time for log %d: %w", record.ID, err)
	}
	params, err := normalizeParamsJSON(paramsText)
	if err != nil {
		return logs.LogRecord{}, fmt.Errorf("decode params for log %d: %w", record.ID, err)
	}
	record.Timestamp = timestamp
	record.CapturedAt = capturedAt
	record.Params = params
	return record, nil
}

func normalizeParamsJSON(value string) (string, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(value))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return "", err
	}
	if _, ok := decoded.(map[string]any); !ok {
		return "", errors.New("params must be a JSON object")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(value)); err != nil {
		return "", fmt.Errorf("compact params JSON: %w", err)
	}
	return compact.String(), nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return errors.New("params contains multiple JSON values")
	}
	return fmt.Errorf("invalid JSON after params: %w", err)
}

func selectTimestampBoundary(ctx context.Context, tx *sql.Tx, direction logs.SortDirection) (time.Time, error) {
	directionSQL := "ASC"
	if direction == logs.SortDescending {
		directionSQL = "DESC"
	}
	var value string
	query := "SELECT timestamp FROM logs ORDER BY " + timestampKey("timestamp") + " " + directionSQL + ", id " + directionSQL + " LIMIT 1"
	if err := tx.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return time.Time{}, fmt.Errorf("query log time range: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse log time range: %w", err)
	}
	return parsed, nil
}
