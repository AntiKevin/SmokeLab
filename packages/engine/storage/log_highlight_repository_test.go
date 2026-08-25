package storage

import (
	"context"
	"reflect"
	"testing"

	"SmokeLab/packages/engine/logs"
)

func TestLogHighlightConfigurationDetectsLeavesAndPersistsSelections(t *testing.T) {
	t.Parallel()
	db, repository := openReadRepository(t)
	insertStoredLog(t, db, storedLog{
		application: "api",
		params: `{
            "global_event_name":"created",
            "context":{"event":{"name":"checkout"}},
            "tags":["one",{"ignored":true}],
            "empty_object":{},
            "a.b":{"c/d~e":true}
        }`,
	})
	insertStoredLog(t, db, storedLog{
		application: "worker",
		params:      `{"global_event_name":"processed","global_name":"job"}`,
	})
	service := logs.NewReadService(repository)

	configuration, err := service.HighlightConfiguration(context.Background())
	if err != nil {
		t.Fatalf("HighlightConfiguration returned error: %v", err)
	}
	if len(configuration) != 2 {
		t.Fatalf("configuration = %#v", configuration)
	}
	wantAPIFields := []logs.HighlightField{
		{Path: "/a.b/c~1d~0e", Label: `["a.b"]["c/d~e"]`},
		{Path: "/context/event/name", Label: "context.event.name"},
		{Path: "/global_event_name", Label: "global_event_name"},
		{Path: "/tags", Label: "tags"},
	}
	if !reflect.DeepEqual(configuration[0].Fields, wantAPIFields) {
		t.Fatalf("api fields = %#v, want %#v", configuration[0].Fields, wantAPIFields)
	}

	settings := []logs.HighlightSetting{
		{Application: "api", FieldPath: "/global_event_name"},
		{Application: "worker", FieldPath: "/global_event_name"},
	}
	if err := service.SaveHighlightSettings(context.Background(), settings); err != nil {
		t.Fatalf("SaveHighlightSettings returned error: %v", err)
	}
	page, err := service.List(context.Background(), logs.ListLogsRequest{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if !reflect.DeepEqual(page.HighlightColumns, []logs.LogHighlightColumn{{Path: "/global_event_name", Label: "global_event_name"}}) {
		t.Fatalf("highlight columns = %#v", page.HighlightColumns)
	}
	for _, record := range page.Items {
		if record.HighlightValues["/global_event_name"] == "" {
			t.Fatalf("record %d has no highlighted value: %#v", record.ID, record.HighlightValues)
		}
	}

	if err := service.SaveHighlightSettings(context.Background(), []logs.HighlightSetting{{Application: "api"}}); err != nil {
		t.Fatalf("remove highlight returned error: %v", err)
	}
	stored, err := repository.HighlightSettings(context.Background(), []string{"api"})
	if err != nil {
		t.Fatalf("HighlightSettings returned error: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("api setting was not removed: %#v", stored)
	}
}
