//go:build noupdate

package app

import "github.com/HappyOnigiri/nnx/internal/domain"

// updateBuildDisabledReason は noupdate タグのビルドで機能全体を無効にする。
// 配布元を読む実装も置き換えの実装もコンパイルされていない。
func updateBuildDisabledReason() domain.UpdateDisabledReason {
	return domain.UpdateDisabledExcludedFromBuild
}
