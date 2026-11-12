package store_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
	"github.com/HappyOnigiri/NextNowX/internal/store"
)

func TestTaskScopeMigratesToAnAttachmentOnce(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scope.db")
	legacy, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = legacy.Close() }()
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(
		ctx,
		`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
	); err != nil {
		t.Fatal(err)
	}
	for index, file := range files {
		if strings.HasPrefix(filepath.Base(file), "021_") {
			break
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := legacy.ExecContext(ctx, string(body)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if _, err := legacy.ExecContext(
			ctx,
			`INSERT INTO schema_migrations VALUES(?, '2026-01-01T00:00:00Z')`,
			index+1,
		); err != nil {
			t.Fatal(err)
		}
	}
	const created = "2026-01-01T00:00:00Z"
	const updated = "2026-01-02T00:00:00Z"
	const content = "  # 要件\n\n元の本文を保持する。\n"
	for _, statement := range []string{
		`INSERT INTO projects(id,public_id,title,archived,created_at,updated_at)
      VALUES('p','P-1','Project',1,'` + created + `','` + updated + `')`,
		`INSERT INTO features(id,public_id,project_id,title,created_at,updated_at)
      VALUES('f','F-1','p','Feature','` + created + `','` + updated + `')`,
		`INSERT INTO tasks(id,public_id,feature_id,title,scope,created_at,updated_at)
      VALUES('t','T-1','f','Task','','` + created + `','` + updated + `'),
            ('empty','T-2','f','Empty','   ','` + created + `','` + updated + `')`,
		`INSERT INTO documents(id,task_id,kind,title,content,is_implementation_plan,created_at,updated_at)
      VALUES('plan','t','markdown','Plan','# Plan',1,'` + created + `','` + updated + `'),
            ('notes','t','markdown','Notes','# Notes',0,'` + created + `','` + updated + `')`,
		`INSERT INTO dependencies(blocker_task_id,blocked_task_id,created_at)
      VALUES('t','empty','` + created + `')`,
		`INSERT INTO pull_requests(task_id,host,owner,repository,number,url)
      VALUES('t','github.com','acme','app',1,'https://github.com/acme/app/pull/1')`,
	} {
		if _, err := legacy.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := legacy.ExecContext(ctx, `UPDATE tasks SET scope=? WHERE id='t'`, content); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		database, err := store.Open(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := database.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Documents) != 3 || len(snapshot.Tasks) != 2 || len(snapshot.Dependencies) != 1 ||
			len(snapshot.PullRequests) != 1 {
			t.Fatalf("migration changed related data: %+v", snapshot)
		}
		found := false
		for _, document := range snapshot.Documents {
			full, err := database.GetDocument(ctx, document.ID)
			if err != nil {
				t.Fatal(err)
			}
			if document.Title == "Scope" {
				found = true
				if full.TaskID != "T-1" || full.Kind != domain.DocumentKindMarkdown || full.Content != content ||
					full.IsImplementationPlan ||
					full.CreatedAt.Format("2006-01-02T15:04:05Z07:00") != created ||
					full.UpdatedAt.Format("2006-01-02T15:04:05Z07:00") != updated {
					t.Fatalf("migrated attachment=%+v", full)
				}
			}
		}
		if !found {
			t.Fatal("scope attachment missing")
		}
		plan, err := database.GetImplementationPlan(ctx, "T-1")
		if err != nil || plan.ID != "plan" || plan.Content != "# Plan" {
			t.Fatalf("plan=%+v err=%v", plan, err)
		}
		if _, err := database.DB().ExecContext(ctx, `SELECT scope FROM tasks`); err == nil {
			t.Fatal("scope column still exists")
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
