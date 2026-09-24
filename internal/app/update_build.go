//go:build !noupdate

package app

import "github.com/HappyOnigiri/nnx/internal/domain"

// updateBuildDisabledReason は既定のビルドでは何も無効にしない。更新機能を外した
// ビルドとの差分はこのファイルの選択だけで表し、実行時の条件を増やさない。
func updateBuildDisabledReason() domain.UpdateDisabledReason { return domain.UpdateEnabled }
