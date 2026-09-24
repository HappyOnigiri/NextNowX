//go:build unix

package update

import "syscall"

// installerProcessAttributes は新しいセッションでインストーラーを起動させる。制御端末を
// 継承させないと、install.sh が呼ぶ `nnx setup` が /dev/tty を開いて応答を待ち続ける。
func installerProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
