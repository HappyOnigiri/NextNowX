//go:build noupdate

package cli

import "github.com/spf13/cobra"

// addUpdateCommand は noupdate タグのビルドでは何も登録しない。更新の実装は
// コンパイル対象から外れており、呼び出せるコマンドがない。
func (s *state) addUpdateCommand(*cobra.Command) {}
