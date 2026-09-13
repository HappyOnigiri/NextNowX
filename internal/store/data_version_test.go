package store_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/app"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func TestDataVersionChangesOnlyAfterAnotherConnectionWrites(t *testing.T) {
	ctx := context.Background()
	database, service := openTestService(t)
	before, err := database.DataVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := database.DataVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged != before {
		t.Fatalf("data version changed without a write: %d -> %d", before, unchanged)
	}
	if _, err := service.CreateProject(ctx, "Realtime", ""); err != nil {
		t.Fatal(err)
	}
	after, err := database.DataVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after == before {
		t.Fatalf("data version did not change after a write: %d", after)
	}
}

func TestDataVersionSeesWritesFromAnotherStore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "shared.db")
	watcher, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = watcher.Close() })
	writer, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	before, err := watcher.DataVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	provider, _ := githubprovider.NewFixtureProvider("demo")
	if _, err := app.New(writer, provider).CreateProject(ctx, "Другой", ""); err != nil {
		t.Fatal(err)
	}
	after, err := watcher.DataVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after == before {
		t.Fatalf("data version did not follow the other store: %d", after)
	}
}

// 上限 1 本のプールで専用接続を握ると、以降の全クエリが待ち続ける。利用不可を
// 返したうえで通常のクエリが通ることまで確かめる。
func TestDataVersionUnavailableInMemoryLeavesThePoolUsable(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.DataVersion(ctx); !errors.Is(err, store.ErrDataVersionUnavailable) {
		t.Fatalf("expected ErrDataVersionUnavailable, got %v", err)
	}
	query, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	provider, _ := githubprovider.NewFixtureProvider("demo")
	if _, err := app.New(database, provider).Snapshot(query); err != nil {
		t.Fatalf("the pool is exhausted after the unavailable read: %v", err)
	}
}

func TestDataVersionAfterCloseReturnsAnError(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "closed.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.DataVersion(ctx); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	timed, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := database.DataVersion(timed); !errors.Is(err, store.ErrStoreClosed) {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}
}

// *sql.Conn は壊れると以後すべて ErrConnDone を返す。張り直さないと変更検知だけが
// 永久に止まるので、閉じた接続からの復帰を確かめる。
func TestDataVersionReopensTheConnectionAfterItDies(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "reopen.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	before, err := database.DataVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CloseDataVersionConnForTest(); err != nil && !errors.Is(err, sql.ErrConnDone) {
		t.Fatal(err)
	}
	after, err := database.DataVersion(ctx)
	if err != nil {
		t.Fatalf("the connection was not reopened: %v", err)
	}
	if after != before {
		t.Fatalf("the reopened connection reported a different value: %d != %d", after, before)
	}
}
