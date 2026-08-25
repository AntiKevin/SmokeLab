package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// HighlightField is one selectable leaf path detected in an application's
// structured log parameters. Path uses RFC 6901 JSON Pointer encoding.
type HighlightField struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// ApplicationHighlight describes the available and selected highlighted field
// for one application.
type ApplicationHighlight struct {
	Application string           `json:"application"`
	FieldPath   string           `json:"fieldPath"`
	Fields      []HighlightField `json:"fields"`
}

// HighlightSetting associates one application with at most one field. An empty
// FieldPath removes the application's existing setting.
type HighlightSetting struct {
	Application string `json:"application"`
	FieldPath   string `json:"fieldPath"`
}

// HighlightRepository persists settings and discovers fields from stored logs.
type HighlightRepository interface {
	HighlightConfiguration(context.Context) ([]ApplicationHighlight, error)
	HighlightSettings(context.Context, []string) ([]HighlightSetting, error)
	SaveHighlightSettings(context.Context, []HighlightSetting) error
}

// NewHighlightField creates a displayable field from a canonical JSON Pointer.
func NewHighlightField(path string) (HighlightField, error) {
	segments, err := decodeHighlightPath(path)
	if err != nil {
		return HighlightField{}, err
	}
	return HighlightField{Path: path, Label: highlightPathLabel(segments)}, nil
}

func normalizeHighlightSettings(configuration []ApplicationHighlight, settings []HighlightSetting) ([]HighlightSetting, error) {
	available := make(map[string]map[string]struct{}, len(configuration))
	for _, application := range configuration {
		fields := make(map[string]struct{}, len(application.Fields))
		for _, field := range application.Fields {
			fields[field.Path] = struct{}{}
		}
		available[application.Application] = fields
	}

	normalized := make([]HighlightSetting, 0, len(settings))
	seen := make(map[string]struct{}, len(settings))
	for _, setting := range settings {
		setting.Application = strings.TrimSpace(setting.Application)
		if setting.Application == "" {
			return nil, errors.New("highlight application is required")
		}
		if _, exists := seen[setting.Application]; exists {
			return nil, fmt.Errorf("duplicate highlight application %q", setting.Application)
		}
		seen[setting.Application] = struct{}{}
		fields, exists := available[setting.Application]
		if !exists {
			return nil, fmt.Errorf("unknown highlight application %q", setting.Application)
		}
		if setting.FieldPath != "" {
			if _, exists := fields[setting.FieldPath]; !exists {
				return nil, fmt.Errorf("highlight field %q is not available for application %q", setting.FieldPath, setting.Application)
			}
		}
		normalized = append(normalized, setting)
	}
	return normalized, nil
}

func applyHighlightSettings(page LogPage, settings []HighlightSetting) (LogPage, error) {
	byApplication := make(map[string]HighlightSetting, len(settings))
	columns := make(map[string]LogHighlightColumn, len(settings))
	for _, setting := range settings {
		field, err := NewHighlightField(setting.FieldPath)
		if err != nil {
			return LogPage{}, fmt.Errorf("decode highlight for application %q: %w", setting.Application, err)
		}
		byApplication[setting.Application] = setting
		columns[field.Path] = LogHighlightColumn(field)
	}

	page.HighlightColumns = make([]LogHighlightColumn, 0, len(columns))
	for _, column := range columns {
		page.HighlightColumns = append(page.HighlightColumns, column)
	}
	sort.Slice(page.HighlightColumns, func(i, j int) bool {
		left := strings.ToLower(page.HighlightColumns[i].Label)
		right := strings.ToLower(page.HighlightColumns[j].Label)
		if left != right {
			return left < right
		}
		if page.HighlightColumns[i].Label != page.HighlightColumns[j].Label {
			return page.HighlightColumns[i].Label < page.HighlightColumns[j].Label
		}
		return page.HighlightColumns[i].Path < page.HighlightColumns[j].Path
	})

	for index := range page.Items {
		page.Items[index].HighlightValues = make(map[string]string)
		setting, configured := byApplication[page.Items[index].Application]
		if !configured {
			continue
		}
		value, found, err := extractHighlightValue(page.Items[index].Params, setting.FieldPath)
		if err != nil {
			return LogPage{}, fmt.Errorf("extract highlight for log %d: %w", page.Items[index].ID, err)
		}
		if found {
			page.Items[index].HighlightValues[setting.FieldPath] = value
		}
	}
	return page, nil
}

func extractHighlightValue(params, path string) (string, bool, error) {
	segments, err := decodeHighlightPath(path)
	if err != nil {
		return "", false, err
	}
	current := json.RawMessage(params)
	for _, segment := range segments {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(current, &object); err != nil {
			return "", false, fmt.Errorf("decode object at %q: %w", segment, err)
		}
		var found bool
		current, found = object[segment]
		if !found {
			return "", false, nil
		}
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, current); err != nil {
		return "", false, fmt.Errorf("compact value: %w", err)
	}
	return compact.String(), true, nil
}

func decodeHighlightPath(path string) ([]string, error) {
	if path == "" || path[0] != '/' {
		return nil, fmt.Errorf("invalid highlight path %q", path)
	}
	rawSegments := strings.Split(path[1:], "/")
	segments := make([]string, len(rawSegments))
	for index, raw := range rawSegments {
		var decoded strings.Builder
		for offset := 0; offset < len(raw); offset++ {
			if raw[offset] != '~' {
				decoded.WriteByte(raw[offset])
				continue
			}
			if offset+1 >= len(raw) || (raw[offset+1] != '0' && raw[offset+1] != '1') {
				return nil, fmt.Errorf("invalid highlight path escape in %q", path)
			}
			offset++
			if raw[offset] == '0' {
				decoded.WriteByte('~')
			} else {
				decoded.WriteByte('/')
			}
		}
		segments[index] = decoded.String()
	}
	return segments, nil
}

func highlightPathLabel(segments []string) string {
	var label strings.Builder
	for index, segment := range segments {
		if isSimpleHighlightSegment(segment) {
			if index > 0 {
				label.WriteByte('.')
			}
			label.WriteString(segment)
			continue
		}
		encoded, _ := json.Marshal(segment)
		label.WriteByte('[')
		label.Write(encoded)
		label.WriteByte(']')
	}
	return label.String()
}

func isSimpleHighlightSegment(segment string) bool {
	for index, character := range segment {
		if index == 0 {
			if character != '_' && !unicode.IsLetter(character) {
				return false
			}
		} else if character != '_' && !unicode.IsLetter(character) && !unicode.IsDigit(character) {
			return false
		}
	}
	return segment != ""
}
