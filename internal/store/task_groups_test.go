package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/NextNowX/internal/app"
	"github.com/HappyOnigiri/NextNowX/internal/domain"
	"github.com/HappyOnigiri/NextNowX/internal/store"
)

func TestTaskGroupsMigrationPreservesExistingTasksAndSequences(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "before-task-groups.db")
	legacy := openLegacyDatabase(t, ctx, path, "023_")
	if _, err := legacy.ExecContext(ctx, `
		INSERT INTO projects(id, public_id, title, created_at, updated_at)
		VALUES ('p', 'P-1', 'Project', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO features(id, public_id, project_id, title, created_at, updated_at)
		VALUES ('f', 'F-1', 'p', 'Feature', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO tasks(id, public_id, feature_id, title, kind, created_at, updated_at)
		VALUES ('t', 'T-1', 'f', 'Existing task', 'work', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		UPDATE id_sequences SET next_value = 2 WHERE entity = 'task';
	`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	snapshot, err := database.Snapshot(ctx)
	if err != nil || len(snapshot.TaskGroups) != 0 || len(snapshot.Tasks) != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	task := snapshot.Tasks[0]
	if task.ID != "T-1" || task.FeatureID != "F-1" || task.Title != "Existing task" ||
		task.Kind != domain.TaskKindWork || task.GroupID != "" {
		t.Fatalf("migrated task=%+v", task)
	}
	service := app.New(database, nil)
	group, err := service.CreateTaskGroup(ctx, task.FeatureID, "Work")
	if err != nil || group.ID != "G-1" {
		t.Fatalf("group=%+v err=%v", group, err)
	}
	created, err := service.CreateTask(ctx, task.FeatureID, "New task", "", "", group.ID)
	if err != nil || created.ID != "T-2" || created.GroupID != group.ID {
		t.Fatalf("created task=%+v err=%v", created, err)
	}
}

func TestTaskGroupColorMigrationPreservesExistingGroup(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "before-task-group-color.db")
	legacy := openLegacyDatabase(t, ctx, path, "025_")
	if _, err := legacy.ExecContext(ctx, `
		INSERT INTO projects(id, public_id, title, created_at, updated_at)
		VALUES ('p', 'P-1', 'Project', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO features(id, public_id, project_id, title, created_at, updated_at)
		VALUES ('f', 'F-1', 'p', 'Feature', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO task_groups(public_id, feature_id, name) VALUES ('G-1', 'f', 'Existing group');
		INSERT INTO tasks(id, public_id, feature_id, title, created_at, updated_at, group_id)
		VALUES ('t', 'T-1', 'f', 'Existing task', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', 'G-1');
	`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	snapshot, err := database.Snapshot(ctx)
	want := domain.TaskGroup{ID: "G-1", FeatureID: "F-1", Name: "Existing group"}
	if err != nil || len(snapshot.TaskGroups) != 1 || snapshot.TaskGroups[0] != want ||
		len(snapshot.Tasks) != 1 || snapshot.Tasks[0].GroupID != want.ID {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
}

func TestTaskGroupsFollowFeatureAndProjectCascade(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	for _, deleteProject := range []bool{false, true} {
		feature, err := createFeature(ctx, service, "Groups", "")
		if err != nil {
			t.Fatal(err)
		}
		group, err := service.CreateTaskGroup(ctx, feature.ID, "An empty group")
		if err != nil {
			t.Fatal(err)
		}
		if err := service.DeleteFeature(
			ctx,
			feature.ID,
			false,
		); domain.ErrorCode(
			err,
		) != domain.DomainErrorCodeReferencesExist {
			t.Fatalf("feature delete without cascade: %v", err)
		}
		if deleteProject {
			err = service.DeleteProject(ctx, feature.ProjectID, true)
		} else {
			err = service.DeleteFeature(ctx, feature.ID, true)
		}
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := service.Snapshot(ctx)
		if err != nil || len(snapshot.TaskGroups) != 0 {
			t.Fatalf("groups after cascade=%+v err=%v", snapshot.TaskGroups, err)
		}
		err = service.DeleteTaskGroup(ctx, group.ID, false)
		if domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
			t.Fatalf("group delete after cascade: %v", err)
		}
	}
}

func TestTaskUpdateWithoutGroupPreservesCurrentMembership(t *testing.T) {
	database, service := openTestService(t)
	ctx := context.Background()
	feature, err := createFeature(ctx, service, "Feature", "")
	if err != nil {
		t.Fatal(err)
	}
	group, err := service.CreateTaskGroup(ctx, feature.ID, "First")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Task", "", "", group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteTaskGroup(ctx, group.ID, false); err != nil {
		t.Fatal(err)
	}
	task.Title = "Renamed task"
	updated, err := database.UpdateTask(ctx, task, nil)
	if err != nil || updated.GroupID != "" || updated.Title != task.Title {
		t.Fatalf("updated task=%+v err=%v", updated, err)
	}
}
