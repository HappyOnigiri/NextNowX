package store_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HappyOnigiri/nnx/internal/app"
	"github.com/HappyOnigiri/nnx/internal/config"
	"github.com/HappyOnigiri/nnx/internal/domain"
	"github.com/HappyOnigiri/nnx/internal/store"
)

func TestOpenReportsWhetherItCreatedTheDatabaseFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "created.db")
	created, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if !created.CreatedDatabaseFile() {
		t.Error("opening a missing path did not report a created database file")
	}
	if err := created.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	if reopened.CreatedDatabaseFile() {
		t.Error("reopening an existing database reported a created database file")
	}
}

// TestOpenDoesNotReportCreationForNonFileDatabases は、ファイルを持たない
// データベースとファイル DSN を「新規作成ではない」として扱うことを確かめる。
// サンプルの投入先はファイルのパスで開いたデータベースだけである。
func TestOpenDoesNotReportCreationForNonFileDatabases(t *testing.T) {
	ctx := context.Background()
	empty := filepath.Join(t.TempDir(), "empty.db")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		":memory:",
		"file:" + filepath.Join(t.TempDir(), "dsn.db"),
		empty,
	} {
		database, err := store.Open(ctx, path)
		if err != nil {
			t.Fatalf("open %q: %v", path, err)
		}
		if database.CreatedDatabaseFile() {
			t.Errorf("opening %q reported a created database file", path)
		}
		_ = database.Close()
	}
}

func TestEnsureSampleDataCreatesFirstRunGraph(t *testing.T) {
	for _, test := range []struct {
		name     string
		locale   string
		contains string
	}{
		{name: "english", locale: "en_US.UTF-8", contains: "Sample project"},
		{name: "japanese", locale: "ja_JP.UTF-8", contains: "サンプル project"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LC_ALL", test.locale)
			t.Setenv("LC_MESSAGES", test.locale)
			t.Setenv("LANG", test.locale)
			ctx := context.Background()
			service := openSampleDataService(t)
			added, err := service.EnsureSampleData(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !added {
				t.Fatal("a freshly created database did not receive the sample data")
			}
			snapshot, err := service.Snapshot(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Projects) != 1 || len(snapshot.Features) != 1 ||
				len(snapshot.Tasks) != 6 || len(snapshot.Documents) != 2 {
				t.Fatalf(
					"projects=%d features=%d tasks=%d documents=%d, want 1/1/6/2",
					len(snapshot.Projects), len(snapshot.Features),
					len(snapshot.Tasks), len(snapshot.Documents),
				)
			}
			if len(snapshot.PullRequests) != 0 {
				t.Errorf("pull requests=%d, want none", len(snapshot.PullRequests))
			}
			assertSampleDisplayStates(t, snapshot)
			if issues := service.Validate(ctx); len(issues) > 0 {
				t.Errorf("validate sample data: %v", issues)
			}
			if !strings.Contains(snapshot.Projects[0].Title, test.contains) {
				t.Errorf("project title=%q, want it to contain %q", snapshot.Projects[0].Title, test.contains)
			}
		})
	}
}

// assertSampleDisplayStates は、初回のグラフだけで designed・ready・依存未解決を
// 一度に見られることを確かめる。サンプルの目的がこの 3 つを見せることにある。
func assertSampleDisplayStates(t *testing.T, snapshot domain.Snapshot) {
	t.Helper()
	designed := 0
	blocked := 0
	for _, task := range snapshot.Tasks {
		if task.DisplayState == domain.TaskDisplayStateDesigned {
			designed++
		}
		for _, label := range task.BlockLabels {
			if label == domain.TaskBlockLabelDependencyUnresolved {
				blocked++
			}
		}
	}
	if designed < 1 {
		t.Error("no sample task reaches the designed display state")
	}
	if blocked < 1 {
		t.Error("no sample task carries the dependency_unresolved block label")
	}
	if len(snapshot.ReadyTasks) < 1 {
		t.Error("no sample task is ready")
	}
}

func TestEnsureSampleDataSkipsExistingDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "existing.db")
	first, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	configStore, err := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewWithConfig(database, nil, configStore)
	added, err := service.EnsureSampleData(ctx)
	if err != nil || added {
		t.Fatalf("added=%t err=%v, want no sample data on an existing database", added, err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projects) != 0 || len(snapshot.Tasks) != 0 || len(snapshot.Documents) != 0 {
		t.Fatalf("snapshot=%+v, want the existing database left untouched", snapshot)
	}
}

func openSampleDataService(t *testing.T) *app.Service {
	t.Helper()
	root := t.TempDir()
	database, err := store.Open(context.Background(), filepath.Join(root, "nnx.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	configStore, err := config.NewStore(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return app.NewWithConfig(database, nil, configStore)
}
