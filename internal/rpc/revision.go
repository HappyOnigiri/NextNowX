package rpc

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	nnxv1 "github.com/HappyOnigiri/NextNowX/gen/nnx/v1"
)

// WatchRevision は現在のリビジョンを即座に 1 通目として送り、以後は変化と heartbeat を
// 流す。切断中の変更は再接続時の再取得へ畳まれるので、取りこぼしという概念がない。
func (h *Handler) WatchRevision(
	ctx context.Context,
	_ *connect.Request[nnxv1.WatchRevisionRequest],
	stream *connect.ServerStream[nnxv1.WatchRevisionResponse],
) error {
	if h.openStreams.Add(1) > maxRevisionStreams {
		h.openStreams.Add(-1)
		return connect.NewError(connect.CodeResourceExhausted, errors.New("too many revision streams are open"))
	}
	defer h.openStreams.Add(-1)
	if !h.watchingRevisions() {
		return errNotWatching()
	}
	current, updates, cancel := h.subscribeRevisions()
	defer cancel()
	if err := sendRevision(stream, current); err != nil {
		return err
	}
	return h.streamRevisions(ctx, stream, current, updates)
}

// subscribeRevisions は購読できない構成でも成立する形に揃える。CodeUnimplemented を返すと
// クライアントに恒久失敗として扱われ、フォーカス再取得への縮退ではなく警告表示になる。
func (h *Handler) subscribeRevisions() (uint64, <-chan uint64, func()) {
	if h.revisions == nil {
		return 1, nil, func() {}
	}
	return h.revisions.Subscribe()
}

// watchingRevisions は変更を検知できているかを返す。購読できない構成は監視の失敗と
// 区別する。前者は heartbeat だけの縮退で、後者は更新が止まっていることの通知である。
func (h *Handler) watchingRevisions() bool {
	return h.revisions == nil || h.revisions.Watching()
}

// errNotWatching は更新が止まっていることをクライアントへ伝える。heartbeat を送り
// 続けると、変更が届かないまま接続だけが健全に見える。
func errNotWatching() error {
	return connect.NewError(connect.CodeUnavailable, errors.New("the database is not being watched for changes"))
}

func (h *Handler) streamRevisions(
	ctx context.Context,
	stream *connect.ServerStream[nnxv1.WatchRevisionResponse],
	current uint64,
	updates <-chan uint64,
) error {
	ticker := time.NewTicker(h.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return connect.NewError(connect.CodeCanceled, ctx.Err())
		case value, ok := <-updates:
			// 購読チャネルの close はサーバーの停止。正常終了として畳む。
			if !ok {
				return nil
			}
			current = value
		case <-ticker.C:
			if !h.watchingRevisions() {
				return errNotWatching()
			}
		}
		if err := sendRevision(stream, current); err != nil {
			return err
		}
	}
}

// sendRevision は送信失敗をそのまま返す。無視して回し続けると、切断済みクライアント
// 向けの goroutine が heartbeat 間隔で永久に残る。
func sendRevision(stream *connect.ServerStream[nnxv1.WatchRevisionResponse], revision uint64) error {
	return stream.Send(&nnxv1.WatchRevisionResponse{Revision: revision})
}
