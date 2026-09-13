package revision_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/revision"
)

// fakeReader は読み取りのたびに通知を出す。偽クロックを持たずに 1 周期の完了を
// 待てるようにし、-race でも安定させる。
type fakeReader struct {
	mu    sync.Mutex
	value int64
	err   error
	reads chan struct{}
}

func newFakeReader(value int64) *fakeReader {
	return &fakeReader{value: value, reads: make(chan struct{}, 64)}
}

func (r *fakeReader) DataVersion(context.Context) (int64, error) {
	r.mu.Lock()
	value, err := r.value, r.err
	r.mu.Unlock()
	select {
	case r.reads <- struct{}{}:
	default:
	}
	return value, err
}

func (r *fakeReader) set(value int64, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.value, r.err = value, err
}

// awaitReads は指定した回数の読み取りが終わるまで待つ。
func (r *fakeReader) awaitReads(t *testing.T, count int) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for range count {
		select {
		case <-r.reads:
		case <-deadline:
			t.Fatal("the watcher stopped reading")
		}
	}
}

// startWatcher は基準となる 1 回目の読み取りが終わるまで待って返す。待たずに値を
// 変えると、その値が基準になってしまい変化が起きない。
func startWatcher(t *testing.T, reader *fakeReader, onError func(error)) *revision.Watcher {
	t.Helper()
	watcher := revision.NewWatcher(reader, time.Millisecond, onError)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		watcher.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	reader.awaitReads(t, 1)
	return watcher
}

// awaitRevision は購読チャネルから期待する値が届くまで読む。
func awaitRevision(t *testing.T, updates <-chan uint64, want uint64) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case value, ok := <-updates:
			if !ok {
				t.Fatalf("the subscription closed before revision %d", want)
			}
			if value >= want {
				return
			}
		case <-deadline:
			t.Fatalf("revision %d never arrived", want)
		}
	}
}

func TestWatcherRaisesTheRevisionWhenTheDataVersionChanges(t *testing.T) {
	reader := newFakeReader(7)
	watcher := startWatcher(t, reader, nil)
	current, updates, cancel := watcher.Revisions().Subscribe()
	defer cancel()
	if current != 1 {
		t.Fatalf("the first revision is %d, want 1", current)
	}
	reader.set(8, nil)
	awaitRevision(t, updates, 2)
}

func TestWatcherKeepsTheRevisionWhileTheDataVersionHolds(t *testing.T) {
	reader := newFakeReader(7)
	watcher := startWatcher(t, reader, nil)
	_, updates, cancel := watcher.Revisions().Subscribe()
	defer cancel()
	reader.awaitReads(t, 5)
	select {
	case value := <-updates:
		t.Fatalf("the revision advanced to %d without a change", value)
	default:
	}
}

// data_version は 32 ビットで折り返す。`>` で比べると折り返し後に検知が永久停止する。
func TestWatcherFollowsADecreasingDataVersion(t *testing.T) {
	reader := newFakeReader(2_147_483_647)
	watcher := startWatcher(t, reader, nil)
	_, updates, cancel := watcher.Revisions().Subscribe()
	defer cancel()
	reader.set(-2_147_483_648, nil)
	awaitRevision(t, updates, 2)
}

func TestWatcherReportsReadErrorsAndKeepsRunning(t *testing.T) {
	reader := newFakeReader(7)
	failure := errors.New("read failed")
	errs := make(chan error, 16)
	watcher := startWatcher(t, reader, func(err error) {
		select {
		case errs <- err:
		default:
		}
	})
	_, updates, cancel := watcher.Revisions().Subscribe()
	defer cancel()
	reader.set(0, failure)
	select {
	case err := <-errs:
		if !errors.Is(err, failure) {
			t.Fatalf("unexpected error %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the read error was never reported")
	}
	reader.set(9, nil)
	awaitRevision(t, updates, 2)
}

func TestWatcherClosesEverySubscriptionWhenTheContextEnds(t *testing.T) {
	reader := newFakeReader(7)
	watcher := revision.NewWatcher(reader, time.Millisecond, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		watcher.Run(ctx)
	}()
	_, first, cancelFirst := watcher.Revisions().Subscribe()
	defer cancelFirst()
	_, second, cancelSecond := watcher.Revisions().Subscribe()
	defer cancelSecond()
	cancel()
	<-done
	// 閉じるまで読み捨てる。閉じていなければここで止まる。
	for _, updates := range []<-chan uint64{first, second} {
		for range updates {
		}
	}
}

func TestSubscribeAfterCloseYieldsAClosedChannel(t *testing.T) {
	broadcaster := revision.NewWatcher(newFakeReader(1), time.Hour, nil).Revisions()
	broadcaster.Close()
	current, updates, cancel := broadcaster.Subscribe()
	defer cancel()
	if current != 1 {
		t.Fatalf("the current revision is %d, want 1", current)
	}
	if _, ok := <-updates; ok {
		t.Fatal("the subscription is open after Close")
	}
	broadcaster.Close()
}

// 遅い購読者が監視を止めないよう、チャネルは cap 1 の latest-wins にしてある。
func TestSubscriptionKeepsOnlyTheLatestRevision(t *testing.T) {
	reader := newFakeReader(1)
	watcher := startWatcher(t, reader, nil)
	_, updates, cancel := watcher.Revisions().Subscribe()
	defer cancel()
	for value := int64(2); value <= 4; value++ {
		reader.set(value, nil)
		reader.awaitReads(t, 2)
	}
	awaitRevision(t, updates, 4)
	select {
	case value := <-updates:
		t.Fatalf("a stale revision %d was buffered", value)
	default:
	}
}

func TestCancelStopsDeliveryAndIsIdempotent(t *testing.T) {
	reader := newFakeReader(1)
	watcher := startWatcher(t, reader, nil)
	_, updates, cancel := watcher.Revisions().Subscribe()
	cancel()
	cancel()
	reader.set(2, nil)
	reader.awaitReads(t, 3)
	select {
	case value, ok := <-updates:
		if ok {
			t.Fatalf("revision %d arrived after the cancel", value)
		}
	default:
	}
}

// 読み取りが失敗しているあいだは変更を検知できない。購読側がそれを知れないと、
// 更新が止まったまま接続だけが健全に見える。
func TestWatcherReportsWhetherItIsWatching(t *testing.T) {
	reader := newFakeReader(3)
	watcher := startWatcher(t, reader, func(error) {})
	if !watcher.Revisions().Watching() {
		t.Fatal("the watcher reports that it is not watching after a successful read")
	}
	reader.set(3, errors.New("the database cannot be read"))
	reader.awaitReads(t, 2)
	waitFor(t, func() bool { return !watcher.Revisions().Watching() })

	reader.set(4, nil)
	reader.awaitReads(t, 2)
	waitFor(t, func() bool { return watcher.Revisions().Watching() })
}

// 起動直後の 1 回の読み取り失敗で変化とみなすと、接続が集中する時間帯に全
// クライアントが一度きりの無駄な取り直しをする。
func TestWatcherHoldsTheRevisionWhenTheBaselineReadFailed(t *testing.T) {
	reader := newFakeReader(9)
	reader.set(9, errors.New("the database cannot be read"))
	watcher := startWatcher(t, reader, func(error) {})
	reader.set(9, nil)
	reader.awaitReads(t, 3)
	current, updates, cancel := watcher.Revisions().Subscribe()
	t.Cleanup(cancel)
	if current != 1 {
		t.Fatalf("the revision is %d, want 1", current)
	}
	select {
	case value := <-updates:
		t.Fatalf("the watcher published revision %d without a change", value)
	default:
	}
}

// waitFor は条件が成立するまで短い間隔で確かめる。読み取りと状態の更新は監視の
// goroutine が行うので、読み取りの完了だけでは同期できない。
func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		if condition() {
			return
		}
		select {
		case <-deadline:
			t.Fatal("the watcher never reached the expected watching state")
		case <-time.After(time.Millisecond):
		}
	}
}
