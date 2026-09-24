// Package release は Next Now X の配布元である GitHub Release を読む。取得は未認証で行い、
// 同期対象のリポジトリ向けに設定された資格情報を配布元へ送らない。
// 方針は docs/design/updates.md にある。
package release

import (
	"context"

	"github.com/HappyOnigiri/nnx/internal/domain"
)

// Provider は配布元のリリース一覧を読む境界。
type Provider interface {
	// Releases は draft と prerelease を除いたリリースを返す。並び順は呼び出し側が決める。
	Releases(ctx context.Context) ([]domain.ReleaseNote, error)
}

// StaticProvider は固定のリリース一覧を返す。テストと、実ネットワークへ出てはならない
// 経路のために置く。
type StaticProvider struct {
	releases []domain.ReleaseNote
	err      error
}

// NewStaticProvider は与えた一覧をそのまま返す provider を作る。
func NewStaticProvider(releases []domain.ReleaseNote) *StaticProvider {
	return &StaticProvider{releases: releases}
}

// NewFailingProvider は必ず失敗する provider を作る。確認の失敗経路の検証に使う。
func NewFailingProvider(err error) *StaticProvider { return &StaticProvider{err: err} }

func (p *StaticProvider) Releases(context.Context) ([]domain.ReleaseNote, error) {
	if p.err != nil {
		return nil, p.err
	}
	return append([]domain.ReleaseNote{}, p.releases...), nil
}
