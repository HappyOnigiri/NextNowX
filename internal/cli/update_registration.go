//go:build !noupdate

package cli

import "github.com/spf13/cobra"

// addUpdateCommand は `nnx update` を登録する。更新機能を外したビルドでは
// コマンドごと存在しないので、登録の有無をファイルの選択で表す。
func (s *state) addUpdateCommand(root *cobra.Command) { root.AddCommand(s.updateCommand()) }
