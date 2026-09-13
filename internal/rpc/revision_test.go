package rpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
	"github.com/HappyOnigiri/PRX/internal/app"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/revision"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/store"
)

// newRevisionClient は WatchRevision だけを差し替えた構成を作る。newTestClient を
// 壊さずに購読元と heartbeat 間隔を注入するための補助。
func newRevisionClient(
	t *testing.T, subscriber rpc.RevisionSubscriber, heartbeat time.Duration,
) prxv1connect.PRXServiceClient {
	t.Helper()
	path, handler := rpc.NewWithOptions(internalErrorService{}, rpc.Options{
		Revisions:         subscriber,
		HeartbeatInterval: heartbeat,
	})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
}

// stubSubscriber は購読 1 件分のチャネルを試験から直接動かす。
type stubSubscriber struct {
	current  uint64
	updates  chan uint64
	canceled chan struct{}
	watching atomic.Bool
}

func newStubSubscriber(current uint64) *stubSubscriber {
	subscriber := &stubSubscriber{current: current, updates: make(chan uint64, 1), canceled: make(chan struct{}, 1)}
	subscriber.watching.Store(true)
	return subscriber
}

func (s *stubSubscriber) Watching() bool { return s.watching.Load() }

func (s *stubSubscriber) Subscribe() (uint64, <-chan uint64, func()) {
	return s.current, s.updates, func() {
		select {
		case s.canceled <- struct{}{}:
		default:
		}
	}
}

func receiveRevision(t *testing.T, stream *connect.ServerStreamForClient[prxv1.WatchRevisionResponse]) uint64 {
	t.Helper()
	if !stream.Receive() {
		t.Fatalf("the stream ended: %v", stream.Err())
	}
	return stream.Msg().GetRevision()
}

func openRevisionStream(
	t *testing.T, ctx context.Context, client prxv1connect.PRXServiceClient,
) *connect.ServerStreamForClient[prxv1.WatchRevisionResponse] {
	t.Helper()
	stream, err := client.WatchRevision(ctx, connect.NewRequest(&prxv1.WatchRevisionRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

// 1 通目は heartbeat を待たずに届く。これがクライアント側の「接続のたびに
// 再取得する」契機になる。
func TestWatchRevisionSendsTheCurrentRevisionImmediately(t *testing.T) {
	client := newRevisionClient(t, newStubSubscriber(5), time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := openRevisionStream(t, ctx, client)
	if got := receiveRevision(t, stream); got != 5 {
		t.Fatalf("the first revision is %d, want 5", got)
	}
}

func TestWatchRevisionSendsUpdatesAndHeartbeats(t *testing.T) {
	subscriber := newStubSubscriber(1)
	client := newRevisionClient(t, subscriber, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := openRevisionStream(t, ctx, client)
	if got := receiveRevision(t, stream); got != 1 {
		t.Fatalf("the first revision is %d, want 1", got)
	}
	subscriber.updates <- 2
	for {
		if got := receiveRevision(t, stream); got == 2 {
			break
		}
	}
	// 同じ値の再送が heartbeat である。変化がない間も届くことで、無変化と死亡を
	// クライアントが区別できる。
	if got := receiveRevision(t, stream); got != 2 {
		t.Fatalf("the heartbeat carried %d, want 2", got)
	}
}

func TestWatchRevisionFansOutToEverySubscriber(t *testing.T) {
	watcher := revision.NewWatcher(constantReader{}, time.Hour, nil)
	client := newRevisionClient(t, watcher.Revisions(), time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := openRevisionStream(t, ctx, client)
	second := openRevisionStream(t, ctx, client)
	for _, stream := range []*connect.ServerStreamForClient[prxv1.WatchRevisionResponse]{first, second} {
		if got := receiveRevision(t, stream); got != 1 {
			t.Fatalf("the first revision is %d, want 1", got)
		}
	}
	watcher.Revisions().Close()
	for _, stream := range []*connect.ServerStreamForClient[prxv1.WatchRevisionResponse]{first, second} {
		if stream.Receive() {
			t.Fatalf("the stream continued after Close with %d", stream.Msg().GetRevision())
		}
		if err := stream.Err(); err != nil {
			t.Fatalf("Close ended the stream with %v", err)
		}
	}
}

func TestWatchRevisionCancelsTheSubscriptionWhenTheClientLeaves(t *testing.T) {
	subscriber := newStubSubscriber(3)
	client := newRevisionClient(t, subscriber, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	stream := openRevisionStream(t, ctx, client)
	if got := receiveRevision(t, stream); got != 3 {
		t.Fatalf("the first revision is %d, want 3", got)
	}
	cancel()
	select {
	case <-subscriber.canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("the subscription was never released")
	}
}

// 購読元のない構成でもエラーにしない。クライアントは接続時の再取得とフォーカス
// 再取得で従来どおり動く。
func TestWatchRevisionWithoutASubscriberOnlyHeartbeats(t *testing.T) {
	client := newRevisionClient(t, nil, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := openRevisionStream(t, ctx, client)
	for range 3 {
		if got := receiveRevision(t, stream); got != 1 {
			t.Fatalf("the revision is %d, want 1", got)
		}
	}
}

// 監視が止まっているあいだ heartbeat を送ると、変更が届かないまま接続だけが健全に
// 見える。クライアントが更新の停止を表示できるよう、エラーで終える。
func TestWatchRevisionFailsWhileTheDatabaseIsNotWatched(t *testing.T) {
	subscriber := newStubSubscriber(4)
	subscriber.watching.Store(false)
	client := newRevisionClient(t, subscriber, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.WatchRevision(ctx, connect.NewRequest(&prxv1.WatchRevisionRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stream.Close() })
	if stream.Receive() {
		t.Fatalf("the stream sent revision %d, want an error", stream.Msg().GetRevision())
	}
	if got := connect.CodeOf(stream.Err()); got != connect.CodeUnavailable {
		t.Fatalf("the stream failed with %v, want %v", got, connect.CodeUnavailable)
	}
}

// 監視が止まったら、開いたままのストリームも次の heartbeat で終える。
func TestWatchRevisionEndsWhenWatchingStops(t *testing.T) {
	subscriber := newStubSubscriber(2)
	client := newRevisionClient(t, subscriber, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := openRevisionStream(t, ctx, client)
	if got := receiveRevision(t, stream); got != 2 {
		t.Fatalf("the first revision is %d, want 2", got)
	}
	subscriber.watching.Store(false)
	for stream.Receive() {
	}
	if got := connect.CodeOf(stream.Err()); got != connect.CodeUnavailable {
		t.Fatalf("the stream failed with %v, want %v", got, connect.CodeUnavailable)
	}
}

// サーバー自身の書き込みも watcher の専用接続から見れば他人の書き込みである。
func TestWatchRevisionFollowsWritesMadeThroughTheServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "revision.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	provider, err := githubprovider.NewFixtureProvider("demo")
	if err != nil {
		t.Fatal(err)
	}
	watcher := revision.NewWatcher(database, 5*time.Millisecond, nil)
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		watcher.Run(ctx)
	}()
	t.Cleanup(func() { cancel(); <-watcherDone })
	path, handler := rpc.NewWithOptions(app.New(database, provider), rpc.Options{
		Revisions:         watcher.Revisions(),
		HeartbeatInterval: time.Hour,
	})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
	stream := openRevisionStream(t, ctx, client)
	first := receiveRevision(t, stream)
	if _, err := client.CreateProject(ctx, connect.NewRequest(&prxv1.CreateProjectRequest{
		Title: "Realtime",
	})); err != nil {
		t.Fatal(err)
	}
	if got := receiveRevision(t, stream); got <= first {
		t.Fatalf("the revision did not advance past %d", first)
	}
}

// constantReader は変化しない data_version を返す。リビジョンの配布だけを見る試験で使う。
type constantReader struct{}

func (constantReader) DataVersion(context.Context) (int64, error) { return 1, nil }
