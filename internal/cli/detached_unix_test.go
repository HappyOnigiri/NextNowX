//go:build unix

package cli_test

import "syscall"

// detachedProcessAttributes は子プロセスを新しいセッションで起こす。制御端末を継承
// させると、`prx setup` が /dev/tty を開いて言語の問いかけで止まる。
func detachedProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
