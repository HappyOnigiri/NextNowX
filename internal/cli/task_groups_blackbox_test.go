package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

func TestBlackBoxTaskGroups(t *testing.T) {
	binary := buildCLI(t)
	path := filepath.Join(t.TempDir(), "task-groups.db")
	run := func(args ...string) json.RawMessage {
		t.Helper()
		return runCLIData(t, binary, path, args...)
	}
	project := decodeID(t, run("project", "create", "Project"))
	feature := decodeID(t, run("feature", "create", "Feature", "--project", project))
	otherFeature := decodeID(t, run("feature", "create", "Other feature", "--project", project))
	group := decodeID(t, run("task", "group", "create", feature, "  Checkout  "))
	otherGroup := decodeID(t, run("task", "group", "create", otherFeature, "Checkout"))
	emptyGroup := decodeID(t, run("task", "group", "create", feature, "Empty"))
	task := decodeID(t, run("task", "create", feature, "Assigned", "--group", group))
	blocked := decodeID(t, run("task", "create", feature, "Blocked"))
	run("dep", "add", task, blocked)

	decodeSnapshot := func(args ...string) domain.Snapshot {
		t.Helper()
		var snapshot domain.Snapshot
		if err := json.Unmarshal(run(args...), &snapshot); err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	assertGroup := func(raw json.RawMessage, want string) {
		t.Helper()
		var task domain.Task
		if err := json.Unmarshal(raw, &task); err != nil || task.GroupID != want {
			t.Fatalf("task group = %q, want %q, err=%v", task.GroupID, want, err)
		}
	}
	before := decodeSnapshot("ready")
	if len(before.ReadyTasks) != 1 || before.ReadyTasks[0].ID != task {
		t.Fatalf("ready=%+v", before.ReadyTasks)
	}
	wantGroups := []domain.TaskGroup{
		{ID: group, FeatureID: feature, Name: "Checkout"},
		{ID: emptyGroup, FeatureID: feature, Name: "Empty"},
	}
	for _, args := range [][]string{{"task", "group", "list", feature}, {"graph", feature}} {
		if got := decodeSnapshot(args...).TaskGroups; !reflect.DeepEqual(got, wantGroups) {
			t.Fatalf("%v groups=%+v want=%+v", args, got, wantGroups)
		}
	}
	if got := decodeSnapshot("snapshot").TaskGroups; len(got) != 3 ||
		!slices.Contains(got, domain.TaskGroup{ID: otherGroup, FeatureID: otherFeature, Name: "Checkout"}) {
		t.Fatalf("snapshot groups=%+v", got)
	}
	assertGroup(run("task", "update", task, "--title", "Renamed task"), group)
	assertGroup(run("task", "update", task, "--group", emptyGroup), emptyGroup)
	assertGroup(run("task", "update", task, "--group="), "")
	assertGroup(run("task", "update", task, "--group", group), group)
	run("task", "group", "rename", group, "Delivery")
	if got := decodeSnapshot("graph", feature).TaskGroups[0]; got.ID != group || got.Name != "Delivery" {
		t.Fatalf("renamed group=%+v", got)
	}
	run("task", "group", "delete", group)
	assertGroup(run("task", task), "")
	after := decodeSnapshot("ready")
	if len(after.ReadyTasks) != 1 || after.ReadyTasks[0].ID != task {
		t.Fatalf("ready after delete=%+v", after.ReadyTasks)
	}
	graph := decodeSnapshot("graph", feature)
	if len(graph.Tasks) != 2 || len(graph.Dependencies) != 1 ||
		graph.Dependencies[0].BlockerTaskID != task || graph.Dependencies[0].BlockedTaskID != blocked {
		t.Fatalf("graph after delete=%+v", graph)
	}
}

func TestBlackBoxGroupDocumentCommands(t *testing.T) {
	binary := buildCLI(t)
	path := filepath.Join(t.TempDir(), "group-documents.db")
	run := func(args ...string) json.RawMessage {
		t.Helper()
		return runCLIData(t, binary, path, args...)
	}
	project := decodeID(t, run("project", "create", "Project"))
	feature := decodeID(t, run("feature", "create", "Feature", "--project", project))
	group := decodeID(t, run("task", "group", "create", feature, "Delivery"))
	emptyGroup := decodeID(t, run("task", "group", "create", feature, "Empty"))
	task := decodeID(t, run("task", "create", feature, "Assigned", "--group", group))
	const url = "https://example.com/runbook"
	var added domain.Document
	if err := json.Unmarshal(run("document", "add", group, "--url", url), &added); err != nil {
		t.Fatal(err)
	}
	if added.ID == "" || added.GroupID != group || added.Locator != url ||
		added.ProjectID != "" || added.FeatureID != "" || added.TaskID != "" {
		t.Fatalf("group document = %+v", added)
	}
	run("document", "add", task, "--url", url)
	run("document", "add", feature, "--url", url)
	list := func(id string) []domain.Document {
		t.Helper()
		var body struct {
			Documents []domain.Document `json:"documents"`
		}
		if err := json.Unmarshal(run("document", "--group", id), &body); err != nil {
			t.Fatal(err)
		}
		return body.Documents
	}
	if actual := list(group); len(actual) != 1 || actual[0].ID != added.ID || actual[0].GroupID != group {
		t.Fatalf("filtered documents = %+v, want %s in %s", actual, added.ID, group)
	}
	if actual := list(emptyGroup); actual == nil || len(actual) != 0 {
		t.Fatalf("filtered documents for empty group = %#v, want empty array", actual)
	}
	requireError := func(code string, args ...string) {
		t.Helper()
		result, _, exit := runCLI(t, binary, path, args...)
		if exit == 0 || result.ErrorCode != code {
			t.Fatalf("%v exit=%d, result=%+v, want %s", args, exit, result, code)
		}
	}
	requireError("not_found", "document", "add", "G-999", "--url", url)
	requireError("not_found", "document", "--group", "G-999")
	run("feature", "archive", feature)
	requireError("archived_read_only", "document", "add", group, "--url", url)
	if actual := list(group); len(actual) != 1 || actual[0].ID != added.ID {
		t.Fatalf("filtered documents after archiving = %+v, want %s", actual, added.ID)
	}
}
