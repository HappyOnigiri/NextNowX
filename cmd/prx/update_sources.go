//go:build !noupdate

package main

import (
	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/release"
	"github.com/HappyOnigiri/PRX/internal/update"
)

// setUpdateSources は更新の確認と実行を行う実装を注入する。配布元だけを相手にする
// 経路なので、demo と開発ビルドでの無効化は app 側が持つ。
func setUpdateSources(service *app.Service) {
	service.SetUpdateSources(release.New(), update.New())
}
