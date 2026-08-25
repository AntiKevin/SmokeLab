package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"SmokeLab/packages/engine/logs"
)

var _ logs.HighlightRepository = (*LogReadRepository)(nil)

const detectedHighlightFieldsQuery = `WITH RECURSIVE fields(application, field_path, value, value_type) AS (
    SELECT logs.application,
           '/' || replace(replace(CAST(root.key AS TEXT), '~', '~0'), '/', '~1'),
           root.value,
           root.type
    FROM logs, json_each(logs.params) AS root
    UNION ALL
    SELECT fields.application,
           fields.field_path || '/' || replace(replace(CAST(child.key AS TEXT), '~', '~0'), '/', '~1'),
           child.value,
           child.type
    FROM fields, json_each(CASE WHEN fields.value_type = 'object' THEN fields.value ELSE '{}' END) AS child
)
SELECT DISTINCT application, field_path
FROM fields
WHERE value_type <> 'object'
ORDER BY application COLLATE NOCASE, application, field_path COLLATE NOCASE, field_path`

// HighlightConfiguration returns stored applications, their detected leaf
// fields, and the selected field when one has been configured.
func (r *LogReadRepository) HighlightConfiguration(ctx context.Context) ([]logs.ApplicationHighlight, error) {
	if err := r.ready(ctx, "log highlight configuration"); err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin log highlight configuration: %w", err)
	}
	defer tx.Rollback()

	configuration := make([]logs.ApplicationHighlight, 0)
	byApplication := make(map[string]int)
	rows, err := tx.QueryContext(ctx, `SELECT applications.application, COALESCE(highlights.field_path, '')
        FROM (SELECT DISTINCT application FROM logs) AS applications
        LEFT JOIN log_application_highlights AS highlights USING (application)
        ORDER BY applications.application COLLATE NOCASE, applications.application`)
	if err != nil {
		return nil, fmt.Errorf("query log highlight applications: %w", err)
	}
	for rows.Next() {
		var application logs.ApplicationHighlight
		if err := rows.Scan(&application.Application, &application.FieldPath); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan log highlight application: %w", err)
		}
		application.Fields = make([]logs.HighlightField, 0)
		byApplication[application.Application] = len(configuration)
		configuration = append(configuration, application)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate log highlight applications: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close log highlight applications: %w", err)
	}

	fieldRows, err := tx.QueryContext(ctx, detectedHighlightFieldsQuery)
	if err != nil {
		return nil, fmt.Errorf("query detected log highlight fields: %w", err)
	}
	for fieldRows.Next() {
		var application, path string
		if err := fieldRows.Scan(&application, &path); err != nil {
			fieldRows.Close()
			return nil, fmt.Errorf("scan detected log highlight field: %w", err)
		}
		index, exists := byApplication[application]
		if !exists {
			continue
		}
		field, err := logs.NewHighlightField(path)
		if err != nil {
			fieldRows.Close()
			return nil, fmt.Errorf("decode detected log highlight field %q: %w", path, err)
		}
		configuration[index].Fields = append(configuration[index].Fields, field)
	}
	if err := fieldRows.Err(); err != nil {
		fieldRows.Close()
		return nil, fmt.Errorf("iterate detected log highlight fields: %w", err)
	}
	if err := fieldRows.Close(); err != nil {
		return nil, fmt.Errorf("close detected log highlight fields: %w", err)
	}
	for index := range configuration {
		sort.Slice(configuration[index].Fields, func(left, right int) bool {
			leftLabel := strings.ToLower(configuration[index].Fields[left].Label)
			rightLabel := strings.ToLower(configuration[index].Fields[right].Label)
			if leftLabel != rightLabel {
				return leftLabel < rightLabel
			}
			return configuration[index].Fields[left].Path < configuration[index].Fields[right].Path
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("finish log highlight configuration: %w", err)
	}
	return configuration, nil
}

// HighlightSettings returns configured fields for every application or only
// the supplied application filter.
func (r *LogReadRepository) HighlightSettings(ctx context.Context, applications []string) ([]logs.HighlightSetting, error) {
	if err := r.ready(ctx, "log highlight settings"); err != nil {
		return nil, err
	}
	query := `SELECT highlights.application, highlights.field_path
        FROM log_application_highlights AS highlights
        JOIN (SELECT DISTINCT application FROM logs) AS applications USING (application)`
	arguments := make([]any, 0, len(applications))
	if len(applications) > 0 {
		placeholders := make([]string, len(applications))
		for index, application := range applications {
			placeholders[index] = "?"
			arguments = append(arguments, application)
		}
		query += " WHERE highlights.application IN (" + strings.Join(placeholders, ", ") + ")"
	}
	query += " ORDER BY highlights.application COLLATE NOCASE, highlights.application"
	rows, err := r.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("query log highlight settings: %w", err)
	}
	defer rows.Close()
	settings := make([]logs.HighlightSetting, 0)
	for rows.Next() {
		var setting logs.HighlightSetting
		if err := rows.Scan(&setting.Application, &setting.FieldPath); err != nil {
			return nil, fmt.Errorf("scan log highlight setting: %w", err)
		}
		settings = append(settings, setting)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate log highlight settings: %w", err)
	}
	return settings, nil
}

// SaveHighlightSettings atomically upserts or removes every supplied
// application setting.
func (r *LogReadRepository) SaveHighlightSettings(ctx context.Context, settings []logs.HighlightSetting) error {
	if err := r.ready(ctx, "save log highlight settings"); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin saving log highlight settings: %w", err)
	}
	defer tx.Rollback()
	for _, setting := range settings {
		if setting.FieldPath == "" {
			if _, err := tx.ExecContext(ctx, "DELETE FROM log_application_highlights WHERE application = ?", setting.Application); err != nil {
				return fmt.Errorf("delete highlight for application %q: %w", setting.Application, err)
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO log_application_highlights (application, field_path)
            VALUES (?, ?)
            ON CONFLICT(application) DO UPDATE SET field_path = excluded.field_path`, setting.Application, setting.FieldPath); err != nil {
			return fmt.Errorf("store highlight for application %q: %w", setting.Application, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("finish saving log highlight settings: %w", err)
	}
	return nil
}
