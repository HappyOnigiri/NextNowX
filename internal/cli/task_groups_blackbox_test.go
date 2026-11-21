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
