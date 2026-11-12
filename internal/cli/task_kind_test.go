package cli_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestBlackBoxTaskKinds(t *testing.T) {
	binary := buildCLI(t)
	path := filepath.Join(t.TempDir(), "kinds.db")
	projectID := decodeID(t, runCLIData(t, binary, path, "project", "create", "Kinds"))
	featureID := decodeID(t, runCLIData(t, binary, path, "feature", "create", "Kinds", "--project", projectID))
	task := runCLIData(t, binary, path, "task", "create", featureID, "Default")
	var value struct {
		ID     string `json:"id"`
		Kind   string `json:"kind"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(task, &value); err != nil {
		t.Fatal(err)
	}
	if value.Kind != "development" {
		t.Fatalf("default kind=%q", value.Kind)
	}
	taskID := value.ID
	for _, status := range []string{"not_started", "in_progress", "completed", "closed"} {
		body := runCLIData(t, binary, path, "task", "update", taskID, "--kind", "work", "--status", status)
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatal(err)
		}
		if value.Kind != "work" || value.Status != status {
			t.Fatalf("updated=%+v", value)
		}
	}
	for _, args := range [][]string{
		{"task", "create", featureID, "Invalid", "--kind", "invalid"},
		{"task", "update", taskID, "--kind", "invalid"},
		{"task", "update", taskID, "--status", "designing"},
	} {
		if _, _, exit := runCLI(t, binary, path, args...); exit == 0 {
			t.Fatalf("accepted %v", args)
		}
	}
	body := runCLIData(t, binary, path, "task", "create", featureID, "Work", "--kind", "work")
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	if value.Kind != "work" {
		t.Fatalf("created=%+v", value)
	}
	body = runCLIData(t, binary, path, "task", "update", taskID, "--title", "Renamed")
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	if value.Kind != "work" {
		t.Fatalf("kind changed on rename: %+v", value)
	}
}
