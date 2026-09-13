package domain

import (
	"slices"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// UpdateDisabledReason は更新の確認も適用も行わない理由。空文字列は有効を表す。
type UpdateDisabledReason string

const (
	// UpdateEnabled は機能が有効であることを表す。
	UpdateEnabled UpdateDisabledReason = ""
	// UpdateDisabledDevelopmentBuild は開発ビルドを表す。基にしたリリースしか分からず、
	// 置き換える対象も配布物ではない。
	UpdateDisabledDevelopmentBuild UpdateDisabledReason = "development_build"
	// UpdateDisabledDemo は demo 実行を表す。demo は一時環境で、実ネットワークにも出ない。
	UpdateDisabledDemo UpdateDisabledReason = "demo"
)

// MaxUpdateReleases は 1 回の確認で保持するリリース件数の上限。
// 長く放置された環境でも、保存とモーダルの大きさが際限なく育たないようにする。
const MaxUpdateReleases = 20

// ReleaseNote は 1 件のリリースと、その本文。Body は Release の本文そのもので、
// Markdown として描画する。
type ReleaseNote struct {
	Version     string     `json:"version"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Body        string     `json:"body,omitempty"`
	URL         string     `json:"url,omitempty"`
}

// UpdateCheckState は最後に行った確認の結果。SQLite の singleton 行に対応する。
type UpdateCheckState struct {
	LastCheckedAt *time.Time
	Error         string
	Releases      []ReleaseNote
}

// UpdateStatus は案内を出すかどうかの判断に必要な事実を 1 つにまとめる。
type UpdateStatus struct {
	Enabled         bool                 `json:"enabled"`
	DisabledReason  UpdateDisabledReason `json:"disabled_reason,omitempty"`
	CurrentVersion  string               `json:"current_version"`
	UpdateAvailable bool                 `json:"update_available"`
	ShouldNotify    bool                 `json:"should_notify"`
	LatestVersion   string               `json:"latest_version,omitempty"`
	Releases        []ReleaseNote        `json:"releases"`
	LastCheckedAt   *time.Time           `json:"last_checked_at,omitempty"`
	LastError       string               `json:"last_error,omitempty"`
	SkippedVersion  string               `json:"skipped_version,omitempty"`
}

// UpdateApply は更新の実行そのものが報告した事実。実行器と application 層の境界を渡る。
type UpdateApply struct {
	Version       string
	InstalledPath string
	Output        string
}

// UpdateResult は適用した更新を呼び出し側へ返す形。
type UpdateResult struct {
	Version         string `json:"version"`
	InstalledPath   string `json:"installed_path,omitempty"`
	RestartRequired bool   `json:"restart_required"`
}

// UpdateStatusInput は UpdateStatus を組み立てる材料。CLI と RPC が同じ規則で
// 案内の要否を決めるため、判定はこの 1 か所に置く。
type UpdateStatusInput struct {
	CurrentVersion string
	Disabled       UpdateDisabledReason
	Releases       []ReleaseNote
	SkippedVersion string
	LastCheckedAt  *time.Time
	LastError      string
}

// IsDevelopmentBuild は埋め込みのないビルドを判定する。リリースビルドだけが
// 版番号そのものを名乗る。
func IsDevelopmentBuild(version string) bool {
	return strings.HasSuffix(version, "-dev")
}

// CanonicalVersion は比較に使う v 付きの表記を返す。semver として読めない値、
// および -dev のようなビルド固有の接尾辞が付いた値では空を返す。
func CanonicalVersion(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "v") {
		trimmed = "v" + trimmed
	}
	// 配布のタグは常に vX.Y.Z なので、v1.2 のような短縮形もビルドメタデータも受けない。
	// 取得元の URL はこの値をそのまま使う。
	if !semver.IsValid(trimmed) || semver.Canonical(trimmed) != trimmed || semver.Prerelease(trimmed) != "" {
		return ""
	}
	return trimmed
}

// NewerReleases は現在のバージョンより新しいリリースだけを新しい順で返す。
// draft と prerelease は取得側が落としているので、ここでは比較と並べ替えだけを行う。
func NewerReleases(current string, candidates []ReleaseNote) []ReleaseNote {
	currentVersion := CanonicalVersion(current)
	result := make([]ReleaseNote, 0, len(candidates))
	for _, candidate := range candidates {
		version := CanonicalVersion(candidate.Version)
		if version == "" {
			continue
		}
		if currentVersion != "" && semver.Compare(version, currentVersion) <= 0 {
			continue
		}
		result = append(result, candidate)
	}
	slices.SortStableFunc(result, func(a, b ReleaseNote) int {
		return semver.Compare(CanonicalVersion(b.Version), CanonicalVersion(a.Version))
	})
	if len(result) > MaxUpdateReleases {
		result = result[:MaxUpdateReleases]
	}
	return result
}

// ShouldNotifyUpdate はスキップの意味を「そのバージョン以下を案内しない」と定める。
// より新しいタグが出れば案内は自動的に戻る。
func ShouldNotifyUpdate(latest, skipped string) bool {
	latestVersion := CanonicalVersion(latest)
	if latestVersion == "" {
		return false
	}
	skippedVersion := CanonicalVersion(skipped)
	if skippedVersion == "" {
		return true
	}
	return semver.Compare(latestVersion, skippedVersion) > 0
}

// NewUpdateStatus は案内の要否まで含めた状態を組み立てる。無効なときは現在の
// バージョンと理由だけを返し、確認結果もキャッシュも見せない。
func NewUpdateStatus(input UpdateStatusInput) UpdateStatus {
	if input.Disabled != UpdateEnabled {
		return UpdateStatus{
			Enabled:        false,
			DisabledReason: input.Disabled,
			CurrentVersion: input.CurrentVersion,
			Releases:       []ReleaseNote{},
		}
	}
	releases := NewerReleases(input.CurrentVersion, input.Releases)
	status := UpdateStatus{
		Enabled:        true,
		CurrentVersion: input.CurrentVersion,
		Releases:       releases,
		LastCheckedAt:  input.LastCheckedAt,
		LastError:      input.LastError,
		SkippedVersion: input.SkippedVersion,
	}
	if len(releases) == 0 {
		return status
	}
	status.UpdateAvailable = true
	status.LatestVersion = releases[0].Version
	status.ShouldNotify = ShouldNotifyUpdate(status.LatestVersion, input.SkippedVersion)
	return status
}
