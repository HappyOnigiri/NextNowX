package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/NextNowX/internal/app"
	"github.com/HappyOnigiri/NextNowX/internal/domain"
	"github.com/HappyOnigiri/NextNowX/internal/store"
)

func TestTaskKindMigrationDefaultsExistingTasksToDevelopment(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "migration.db")
	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	service := app.New(database, nil)
	feature, err := createFeature(ctx, service, "Kinds", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Existing task", "Scope", "Bob", "")
	if err != nil {
		t.Fatal(err)
	}
	// 種類を持たない移行前のスキーマとメタデータを再現する。
	if _, err := database.DB().
		ExecContext(ctx, `ALTER TABLE tasks DROP COLUMN kind; DELETE FROM schema_migrations WHERE version=21;`); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	migrated, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = migrated.Close() }()
	got, err := migrated.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != domain.TaskKindDevelopment || got.Title != task.Title || got.Scope != task.Scope ||
		got.Assignee != task.Assignee ||
		got.Status != task.Status ||
		!got.CreatedAt.Equal(task.CreatedAt) {
		t.Fatalf("migrated=%+v original=%+v", got, task)
	}
}

func TestWorkTaskManualLifecycleAndKindChanges(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := createFeature(ctx, service, "Kinds", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Work", "", "", domain.TaskKindWork)
	if err != nil || task.Kind != domain.TaskKindWork {
		t.Fatalf("task=%+v err=%v", task, err)
	}
	_, err = service.UpsertImplementationPlan(
		ctx,
		task.ID,
		domain.Document{Kind: domain.DocumentKindMarkdown, Content: "Plan"},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []domain.TaskStatus{
		domain.TaskStatusNotStarted,
		domain.TaskStatusInProgress,
		domain.TaskStatusCompleted,
		domain.TaskStatusClosed,
	} {
		updated, err := service.UpdateTask(ctx, task.ID, nil, nil, &status, nil, nil)
		if err != nil || updated.Status != status || updated.Kind != domain.TaskKindWork {
			t.Fatalf("updated=%+v err=%v", updated, err)
		}
		snapshot, err := service.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if string(snapshot.Tasks[0].DisplayState) != string(status) {
			t.Fatalf("display=%s status=%s", snapshot.Tasks[0].DisplayState, status)
		}
	}
	designing := domain.TaskStatusDesigning
	if _, err := service.UpdateTask(
		ctx,
		task.ID,
		nil,
		nil,
		&designing,
		nil,
		nil,
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeInvalidStatus {
		t.Fatalf("designing err=%v", err)
	}
	if _, err := service.AttachPullRequest(
		ctx,
		task.ID,
		"https://github.com/acme/api/pull/1",
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodePullRequestOnWorkTask {
		t.Fatalf("attach err=%v", err)
	}
	development := domain.TaskKindDevelopment
	if _, err := service.UpdateTask(ctx, task.ID, nil, nil, &designing, nil, &development); err != nil {
		t.Fatal(err)
	}
	work := domain.TaskKindWork
	if _, err := service.UpdateTask(
		ctx,
		task.ID,
		nil,
		nil,
		nil,
		nil,
		&work,
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeInvalidStatus {
		t.Fatalf("kind err=%v", err)
	}
	notStarted := domain.TaskStatusNotStarted
	if _, err := service.UpdateTask(ctx, task.ID, nil, nil, &notStarted, nil, &work); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateTask(ctx, task.ID, nil, nil, nil, nil, &development); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(ctx, task.ID, "https://github.com/acme/api/pull/1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateTask(
		ctx,
		task.ID,
		nil,
		nil,
		nil,
		nil,
		&work,
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodePullRequestOnWorkTask {
		t.Fatalf("linked kind err=%v", err)
	}
	if err := service.DetachPullRequest(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateTask(ctx, task.ID, nil, nil, nil, nil, &work); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidTaskKindsAreRejected(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := createFeature(ctx, service, "Kinds", "")
	if err != nil {
		t.Fatal(err)
	}
	invalid := domain.TaskKind("invalid")
	if _, err := service.CreateTask(
		ctx,
		feature.ID,
		"Invalid",
		"",
		"",
		invalid,
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeInvalidTaskKind {
		t.Fatalf("create err=%v", err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Default", "", "", "")
	if err != nil || task.Kind != domain.TaskKindDevelopment {
		t.Fatalf("default=%+v err=%v", task, err)
	}
	if _, err := service.UpdateTask(
		ctx,
		task.ID,
		nil,
		nil,
		nil,
		nil,
		&invalid,
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeInvalidTaskKind {
		t.Fatalf("update err=%v", err)
	}
}
