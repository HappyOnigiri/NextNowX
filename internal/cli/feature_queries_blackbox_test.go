package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/HappyOnigiri/NextNowX/internal/app"
	"github.com/HappyOnigiri/NextNowX/internal/domain"
	"github.com/HappyOnigiri/NextNowX/internal/store"
)

func TestBlackBoxFeatureScopedLists(t *testing.T) {
	binary := buildCLI(t)
	path := filepath.Join(t.TempDir(), "feature-lists.db")
	run := func(args ...string) json.RawMessage {
		t.Helper()
		return runCLIData(t, binary, path, args...)
	}
	project := decodeID(t, run("project", "create", "Payments"))
	first := decodeID(t, run("feature", "create", "Checkout", "--project", project))
	second := decodeID(t, run("feature", "create", "Refunds", "--project", project))
	empty := decodeID(t, run("feature", "create", "Receipts", "--project", project))
	var blockers []string
	for index, feature := range []string{first, second} {
		blocker := decodeID(t, run("task", "create", feature, "Implement API"))
		blocked := decodeID(t, run("task", "create", feature, "Document API"))
		pullURL := []string{
			"https://github.com/acme/api/pull/11",
			"https://github.com/acme/api/pull/22",
		}[index]
		run("pr", "attach", blocker, pullURL)
		run("dependency", "add", blocker, blocked)
		blockers = append(blockers, blocker)
	}
	run("task", "create", empty, "Implement receipts")

	for _, list := range []struct {
		command, key, idKey, header, emptyMessage string
	}{
		{"pr", "pull_requests", "task_id", "TASK HOST REPOSITORY NUMBER STATE STALE URL", "No pull requests attached.\n"},
		{"dependency", "dependencies", "blocker_task_id", "BLOCKER BLOCKED", "No dependencies found.\n"},
	} {
		t.Run(list.command, func(t *testing.T) {
			for _, scope := range []struct {
				feature string
				want    []string
			}{
				{want: blockers},
				{feature: first, want: blockers[:1]},
				{feature: empty, want: []string{}},
			} {
				args := []string{list.command}
				if scope.feature != "" {
					args = append(args, "--feature", scope.feature)
				}
				var result map[string][]map[string]json.RawMessage
				if err := json.Unmarshal(run(args...), &result); err != nil {
					t.Fatal(err)
				}
				rows := result[list.key]
				if len(result) != 1 || rows == nil || len(rows) != len(scope.want) {
					t.Fatalf("%v JSON = %v, want %s array with %d rows", args, result, list.key, len(scope.want))
				}
				want := slices.Clone(scope.want)
				var actual []string
				for _, row := range rows {
					var id string
					if err := json.Unmarshal(row[list.idKey], &id); err != nil {
						t.Fatal(err)
					}
					actual = append(actual, id)
				}
				slices.Sort(actual)
				slices.Sort(want)
				if !slices.Equal(actual, want) {
					t.Fatalf("%v JSON IDs = %v, want %v", args, actual, want)
				}

				out := executeCLI(t, binary, "", append([]string{"--db", path}, args...)...)
				if out.exit != 0 || out.stderr != "" {
					t.Fatalf("%v text = %+v", args, out)
				}
				if len(want) == 0 {
					if out.stdout != list.emptyMessage {
						t.Fatalf("%v text = %q, want %q", args, out.stdout, list.emptyMessage)
					}
					continue
				}
				header, table, _ := strings.Cut(strings.TrimSuffix(out.stdout, "\n"), "\n")
				if strings.Join(strings.Fields(header), " ") != list.header {
					t.Fatalf("%v header = %q, want %q", args, header, list.header)
				}
				actual = actual[:0]
				for _, row := range strings.Split(table, "\n") {
					id, _, _ := strings.Cut(row, " ")
					actual = append(actual, id)
				}
				slices.Sort(actual)
				if !slices.Equal(actual, want) {
					t.Fatalf("%v text IDs = %v, want %v", args, actual, want)
				}
			}
			assertQueryNotFound(t, binary, path, list.command, "--feature", "F-999")
		})
	}
}

func TestBlackBoxShowTaskGroup(t *testing.T) {
	binary := buildCLI(t)
	path := filepath.Join(t.TempDir(), "show-group.db")
	run := func(args ...string) json.RawMessage {
		t.Helper()
		return runCLIData(t, binary, path, args...)
	}
	createdProject := run("project", "create", "Payments")
	project := decodeID(t, createdProject)
	feature := decodeID(t, run("feature", "create", "Checkout", "--project", project))
	task := decodeID(t, run("task", "create", feature, "Implement API"))
	group := decodeID(t, run("task", "group", "create", feature, "API delivery"))
	dataStore, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	color := "#123abc"
	_, updateErr := app.New(dataStore, nil).UpdateTaskGroup(t.Context(), group, nil, &color)
	if err := dataStore.Close(); err != nil {
		t.Fatal(err)
	}
	if updateErr != nil {
		t.Fatal(updateErr)
	}
	expected, err := json.Marshal(domain.TaskGroup{ID: group, FeatureID: feature, Name: "API delivery", Color: color})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, run("show", group), expected, "show task group")
	out := executeCLI(t, binary, "", "--db", path, "show", group)
	want := "ID: " + group + " Feature: " + feature + " Name: API delivery Color: " + color
	if out.exit != 0 || out.stderr != "" || strings.Join(strings.Fields(out.stdout), " ") != want {
		t.Fatalf("show group text = %+v, want %q", out, want)
	}
	assertJSONEqual(t, run("show", project), createdProject, "show project")
	for _, existing := range []struct{ command, id string }{{"feature", feature}, {"task", task}} {
		assertJSONEqual(t, run("show", existing.id), run(existing.command, existing.id), "show "+existing.command)
	}
	assertQueryNotFound(t, binary, path, "show", "G-999")
}

func TestBlackBoxTaskListImplementationPlans(t *testing.T) {
	binary := buildCLI(t)
	path := filepath.Join(t.TempDir(), "plan-lists.db")
	run := func(args ...string) json.RawMessage {
		t.Helper()
		return runCLIData(t, binary, path, args...)
	}
	project := decodeID(t, run("project", "create", "Payments"))
	feature := decodeID(t, run("feature", "create", "Checkout", "--project", project))
	planned := decodeID(t, run("task", "create", feature, "Planned"))
	unplanned := decodeID(t, run("task", "create", feature, "Unplanned"))
	planPath := filepath.Join(t.TempDir(), "plan.md")
	if err := os.WriteFile(planPath, []byte("# Plan\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("plan", "set", planned, "--file", planPath)
	for _, args := range [][]string{{"task"}, {"task", "--feature", feature}} {
		out := executeCLI(t, binary, "", append([]string{"--db", path}, args...)...)
		lines := strings.Split(strings.TrimSuffix(out.stdout, "\n"), "\n")
		if out.exit != 0 || out.stderr != "" || len(lines) != 3 ||
			strings.Join(strings.Fields(lines[0]), " ") != "ID STATUS READY ASSIGNEE IMPLEMENTATION PLAN TITLE" {
			t.Fatalf("%v text = %+v", args, out)
		}
		wantPlans := map[string]string{planned: "yes", unplanned: "no"}
		for _, row := range lines[1:] {
			fields := strings.Fields(row)
			if len(fields) != 6 || wantPlans[fields[0]] != fields[4] {
				t.Fatalf("%v task row = %q", args, row)
			}
			delete(wantPlans, fields[0])
		}
		var result struct {
			Tasks []json.RawMessage `json:"tasks"`
		}
		if err := json.Unmarshal(run(args...), &result); err != nil || len(result.Tasks) != 2 {
			t.Fatalf("%v JSON = %+v, err = %v", args, result, err)
		}
		for _, row := range result.Tasks {
			id := decodeID(t, row)
			detail := run("task", id)
			assertJSONEqual(t, row, detail, "task list row")
			var value domain.Task
			if err := json.Unmarshal(row, &value); err != nil || value.HasImplementationPlan != (id == planned) {
				t.Fatalf("%v task %s plan = %t, err = %v", args, id, value.HasImplementationPlan, err)
			}
		}
	}
	for _, args := range [][]string{{"graph", feature}, {"snapshot"}, {"ready"}} {
		out := executeCLI(t, binary, "", append([]string{"--db", path}, args...)...)
		header := "ID STATUS READY ASSIGNEE TITLE"
		normalized := strings.Join(strings.Fields(out.stdout), " ")
		if out.exit != 0 || out.stderr != "" || !strings.Contains(normalized, header) ||
			strings.Contains(out.stdout, "IMPLEMENTATION PLAN") {
			t.Fatalf("%v text = %+v", args, out)
		}
	}
}

func assertQueryNotFound(t *testing.T, binary, path string, args ...string) {
	t.Helper()
	out := executeCLI(t, binary, "", append([]string{"--db", path, "--json"}, args...)...)
	if out.exit == 0 || out.stdout != "" {
		t.Fatalf("%v = %+v, want failure without stdout", args, out)
	}
	failure := decodeFailure(t, []byte(out.stderr), out.stderr)
	if failure.ErrorCode != "not_found" {
		t.Fatalf("%v code = %q, want not_found", args, failure.ErrorCode)
	}
}
