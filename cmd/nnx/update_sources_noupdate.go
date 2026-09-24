//go:build noupdate

package main

import "github.com/HappyOnigiri/NextNowX/internal/app"

// setUpdateSources は noupdate タグのビルドでは何も注入しない。配布元を読む実装は
// コンパイル対象から外れており、app 側も機能ごと無効を返す。
func setUpdateSources(*app.Service) {}
