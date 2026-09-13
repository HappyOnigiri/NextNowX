package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func openUpdateCheckStore(t *testing.T) *store.Store {
	t.Helper()
	database, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "update.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

// 実行権は 1 つだけに渡る。複数のタブと CLI が同時に来ても確認は 1 回である。
func TestAcquireUpdateCheckHandsTheRunToOneCallerUntilTheIntervalPasses(t *testing.T) {
	ctx := context.Background()
	database := openUpdateCheckStore(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	due := now.Add(-24 * time.Hour).Unix()

	acquired, err := database.AcquireUpdateCheck(ctx, now, due)
	if err != nil || !acquired {
		t.Fatalf("acquired=%v err=%v", acquired, err)
	}
	again, err := database.AcquireUpdateCheck(ctx, now.Add(time.Minute), due)
	if err != nil || again {
		t.Fatalf("second acquire=%v err=%v", again, err)
	}
	later := now.Add(25 * time.Hour)
	acquired, err = database.AcquireUpdateCheck(ctx, later, later.Add(-24*time.Hour).Unix())
	if err != nil || !acquired {
		t.Fatalf("acquired after the interval=%v err=%v", acquired, err)
	}
}

func TestUpdateCheckStateKeepsTheStoredReleases(t *testing.T) {
	ctx := context.Background()
	database := openUpdateCheckStore(t)
	published := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checked := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	releases := []domain.ReleaseNote{
		{Version: "v0.4.0", PublishedAt: &published, Body: "Notes", URL: "https://example.test"},
	}
	if err := database.CompleteUpdateCheck(ctx, checked, releases, ""); err != nil {
		t.Fatal(err)
	}
	state, err := database.UpdateCheckState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state.LastCheckedAt == nil || !state.LastCheckedAt.Equal(checked) {
		t.Fatalf("checked=%v", state.LastCheckedAt)
	}
	if len(state.Releases) != 1 || state.Releases[0].Body != "Notes" {
		t.Fatalf("releases=%+v", state.Releases)
	}
	if state.Releases[0].PublishedAt == nil || !state.Releases[0].PublishedAt.Equal(published) {
		t.Fatalf("published=%v", state.Releases[0].PublishedAt)
	}
}

// 失敗しても直前の一覧は残す。オフラインでもモーダルは開ける。
func TestCompleteUpdateCheckRecordsTheFailureAlongsideTheCache(t *testing.T) {
	ctx := context.Background()
	database := openUpdateCheckStore(t)
	checked := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	cached := []domain.ReleaseNote{{Version: "v0.4.0"}}
	if err := database.CompleteUpdateCheck(ctx, checked, cached, "the feed is unreachable"); err != nil {
		t.Fatal(err)
	}
	state, err := database.UpdateCheckState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state.Error != "the feed is unreachable" || len(state.Releases) != 1 {
		t.Fatalf("state=%+v", state)
	}
}

func TestUpdateCheckStateStartsEmpty(t *testing.T) {
	state, err := openUpdateCheckStore(t).UpdateCheckState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.LastCheckedAt != nil || state.Error != "" || len(state.Releases) != 0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestCompleteUpdateCheckStoresAnEmptyListWithoutReleases(t *testing.T) {
	ctx := context.Background()
	database := openUpdateCheckStore(t)
	checked := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	if err := database.CompleteUpdateCheck(ctx, checked, nil, ""); err != nil {
		t.Fatal(err)
	}
	state, err := database.UpdateCheckState(ctx)
	if err != nil || len(state.Releases) != 0 {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}
