package app

import (
	"testing"
	"time"
)

// SetNowForTest は service が読む時刻を差し替える。間引きの検証は実時間を待てない。
func (s *Service) SetNowForTest(now func() time.Time) { s.now = now }

// StubUpdateBuildVersionForTest はこのプロセスが名乗るビルドのバージョンを差し替える。
// go test は常に -dev のビルドで動くので、更新が有効な経路はこれがないと検証できない。
func StubUpdateBuildVersionForTest(t *testing.T, version string) {
	t.Helper()
	previous := updateBuildVersion
	updateBuildVersion = func() string { return version }
	t.Cleanup(func() { updateBuildVersion = previous })
}
