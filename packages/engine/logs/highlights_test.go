package logs

import (
	"reflect"
	"testing"
)

func TestNewHighlightFieldFormatsCanonicalPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path  string
		label string
	}{
		{path: "/global_event_name", label: "global_event_name"},
		{path: "/context/event/name", label: "context.event.name"},
		{path: "/a.b/c~1d~0e", label: `["a.b"]["c/d~e"]`},
	}
	for _, test := range tests {
		field, err := NewHighlightField(test.path)
		if err != nil {
			t.Fatalf("NewHighlightField(%q) returned error: %v", test.path, err)
		}
		if field.Label != test.label {
			t.Fatalf("NewHighlightField(%q).Label = %q, want %q", test.path, field.Label, test.label)
		}
	}
	if _, err := NewHighlightField("context/name"); err == nil {
		t.Fatal("NewHighlightField should reject a non-pointer path")
	}
}

func TestApplyHighlightSettingsMergesColumnsAndPreservesJSON(t *testing.T) {
	t.Parallel()
	page := LogPage{Items: []LogRecord{
		{ID: 1, Application: "api", Params: `{"context":{"event":{"name":"created"}}}`},
		{ID: 2, Application: "worker", Params: `{"global_name":9007199254740993}`},
		{ID: 3, Application: "scheduler", Params: `{"context":{}}`},
		{ID: 4, Application: "unconfigured", Params: `{"global_name":"ignored"}`},
	}}
	settings := []HighlightSetting{
		{Application: "api", FieldPath: "/context/event/name"},
		{Application: "worker", FieldPath: "/global_name"},
		{Application: "scheduler", FieldPath: "/context/event/name"},
	}

	got, err := applyHighlightSettings(page, settings)
	if err != nil {
		t.Fatalf("applyHighlightSettings returned error: %v", err)
	}
	wantColumns := []LogHighlightColumn{
		{Path: "/context/event/name", Label: "context.event.name"},
		{Path: "/global_name", Label: "global_name"},
	}
	if !reflect.DeepEqual(got.HighlightColumns, wantColumns) {
		t.Fatalf("columns = %#v, want %#v", got.HighlightColumns, wantColumns)
	}
	if got.Items[0].HighlightValues["/context/event/name"] != `"created"` {
		t.Fatalf("api highlight = %#v", got.Items[0].HighlightValues)
	}
	if got.Items[1].HighlightValues["/global_name"] != "9007199254740993" {
		t.Fatalf("worker highlight = %#v", got.Items[1].HighlightValues)
	}
	if len(got.Items[2].HighlightValues) != 0 || len(got.Items[3].HighlightValues) != 0 {
		t.Fatalf("missing and unconfigured values must remain empty: %#v", got.Items)
	}
}

func TestExtractHighlightValueKeepsListsNullAndLargeNumbers(t *testing.T) {
	t.Parallel()
	params := `{"items":["one",{"nested":true}],"nullable":null,"large":9007199254740993}`
	tests := []struct {
		path string
		want string
	}{
		{path: "/items", want: `["one",{"nested":true}]`},
		{path: "/nullable", want: "null"},
		{path: "/large", want: "9007199254740993"},
	}
	for _, test := range tests {
		got, found, err := extractHighlightValue(params, test.path)
		if err != nil || !found || got != test.want {
			t.Fatalf("extractHighlightValue(%q) = %q, %t, %v; want %q, true", test.path, got, found, err, test.want)
		}
	}
}

func TestNormalizeHighlightSettingsValidatesApplicationAndField(t *testing.T) {
	t.Parallel()
	configuration := []ApplicationHighlight{{
		Application: "api",
		Fields:      []HighlightField{{Path: "/event", Label: "event"}},
	}}
	if _, err := normalizeHighlightSettings(configuration, []HighlightSetting{{Application: "api", FieldPath: "/missing"}}); err == nil {
		t.Fatal("missing field should be rejected")
	}
	if _, err := normalizeHighlightSettings(configuration, []HighlightSetting{{Application: "unknown", FieldPath: ""}}); err == nil {
		t.Fatal("unknown application should be rejected")
	}
	if _, err := normalizeHighlightSettings(configuration, []HighlightSetting{{Application: "api"}, {Application: " api "}}); err == nil {
		t.Fatal("duplicate normalized application should be rejected")
	}
	settings, err := normalizeHighlightSettings(configuration, []HighlightSetting{{Application: " api ", FieldPath: "/event"}})
	if err != nil || !reflect.DeepEqual(settings, []HighlightSetting{{Application: "api", FieldPath: "/event"}}) {
		t.Fatalf("normalized settings = %#v, error = %v", settings, err)
	}
}
