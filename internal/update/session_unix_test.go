//go:build unix

package update

import "testing"

// 制御端末を継承すると、インストーラーが呼ぶセットアップの TUI が応答を待ち続ける。
func TestInstallerRunsInItsOwnSession(t *testing.T) {
	attributes := installerProcessAttributes()
	if attributes == nil || !attributes.Setsid {
		t.Fatalf("attributes=%+v", attributes)
	}
}
