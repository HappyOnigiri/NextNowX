// Package revision はローカルデータベースの変更をリビジョンの増加に変換し、
// 購読者へ配る。SQLite も RPC も知らない。
package revision

import (
	"context"
	"sync"
	"time"
)

// Reader は data_version の供給元。store を import しないための最小境界。
type Reader interface {
	DataVersion(ctx context.Context) (int64, error)
}

// Broadcaster は最新リビジョンを保持し、購読者へ latest-wins で配る。
type Broadcaster struct {
	mu          sync.Mutex
	current     uint64
	closed      bool
	watching    bool
	subscribers map[int]chan uint64
	nextID      int
}

func newBroadcaster() *Broadcaster {
	return &Broadcaster{current: 1, watching: true, subscribers: map[int]chan uint64{}}
}

// Watching は直近の読み取りが成功しているかを返す。false のあいだ変更は検知でき
// ないので、購読を生かしたままにするとクライアントは更新の停止に気づけない。
func (b *Broadcaster) Watching() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.watching
}

func (b *Broadcaster) setWatching(value bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.watching = value
}

// Subscribe は現在のリビジョンと更新チャネルを返す。チャネルは cap 1 の
// latest-wins で、遅い購読者が監視を止めない。cancel は冪等。
func (b *Broadcaster) Subscribe() (uint64, <-chan uint64, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	updates := make(chan uint64, 1)
	if b.closed {
		close(updates)
		return b.current, updates, func() {}
	}
	id := b.nextID
	b.nextID++
	b.subscribers[id] = updates
	var once sync.Once
	return b.current, updates, func() { once.Do(func() { b.unsubscribe(id) }) }
}

func (b *Broadcaster) unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subscribers, id)
}

// publish は新しいリビジョンを保存し、購読者へ配る。溜まっている古い値は捨てる。
// リビジョンはイベント列ではなく冪等な最新値なので、取りこぼしても害はない。
func (b *Broadcaster) publish(value uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.current = value
	for _, updates := range b.subscribers {
		select {
		case <-updates:
		default:
		}
		select {
		case updates <- value:
		default:
		}
	}
}

// Close は全購読チャネルを閉じる。ハンドラはこれを見て正常終了する。
func (b *Broadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, updates := range b.subscribers {
		close(updates)
		delete(b.subscribers, id)
	}
}

// Watcher は data_version を周期的に読み、変化をリビジョンの増加に変換する。
type Watcher struct {
	reader      Reader
	interval    time.Duration
	onError     func(error)
	broadcaster *Broadcaster
}

// NewWatcher は監視間隔と読み取り失敗の報告先を受け取る。間隔を引数にするのは
// テストが実時間を待たずに 1 周期を回せるようにするためである。
func NewWatcher(reader Reader, interval time.Duration, onError func(error)) *Watcher {
	if onError == nil {
		onError = func(error) {}
	}
	return &Watcher{reader: reader, interval: interval, onError: onError, broadcaster: newBroadcaster()}
}

// Revisions は購読窓口を返す。
func (w *Watcher) Revisions() *Broadcaster { return w.broadcaster }

// Run は ctx が終わるまで監視し、終了時に Broadcaster を閉じる。
func (w *Watcher) Run(ctx context.Context) {
	defer w.broadcaster.Close()
	baseline, ok := w.read(ctx)
	w.broadcaster.setWatching(ok)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	revision := uint64(1)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		current, read := w.read(ctx)
		w.broadcaster.setWatching(read)
		if !read {
			continue
		}
		// 基準値を持てていないときは比べる相手がないので、変化とみなさず基準値に
		// する。ここで進めると、起動直後の 1 回の読み取り失敗が全クライアントの
		// 取り直しに化ける。
		if !ok {
			baseline, ok = current, true
			continue
		}
		// data_version は 32 ビットで折り返す。`>` で比較すると折り返し後に
		// 検知が永久停止する。
		if current == baseline {
			continue
		}
		baseline = current
		revision++
		w.broadcaster.publish(revision)
	}
}

func (w *Watcher) read(ctx context.Context) (int64, bool) {
	value, err := w.reader.DataVersion(ctx)
	if err != nil {
		if ctx.Err() == nil {
			w.onError(err)
		}
		return 0, false
	}
	return value, true
}
