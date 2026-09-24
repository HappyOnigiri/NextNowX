package domain_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

func TestCanonicalVersionAcceptsStableReleasesOnly(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "0.4.0", want: "v0.4.0"},
		{value: "v0.4.0", want: "v0.4.0"},
		{value: "  v1.2.3  ", want: "v1.2.3"},
		{value: "0.4.0-dev", want: ""},
		{value: "v1.2.3-rc.1", want: ""},
		{value: "v1.2.3+build", want: ""},
		{value: "v1.2", want: ""},
		{value: "nightly", want: ""},
		{value: "", want: ""},
	}
	for _, test := range tests {
		if got := domain.CanonicalVersion(test.value); got != test.want {
			t.Errorf("CanonicalVersion(%q)=%q, want %q", test.value, got, test.want)
		}
	}
}

func TestIsDevelopmentBuildFollowsTheEmbeddedSuffix(t *testing.T) {
	if !domain.IsDevelopmentBuild("0.4.0-dev") || domain.IsDevelopmentBuild("0.4.0") {
		t.Fatal("development builds are identified by the -dev suffix")
	}
}

// 判定は文字列の不一致ではなく semver の比較で行う。10 は 9 より新しい。
func TestNewerReleasesOrdersBySemanticVersion(t *testing.T) {
	releases := domain.NewerReleases("0.9.0", []domain.ReleaseNote{
		{Version: "v0.10.0"},
		{Version: "v0.9.0"},
		{Version: "v0.11.0"},
		{Version: "v0.8.0"},
		{Version: "nightly"},
	})
	if len(releases) != 2 {
		t.Fatalf("releases=%+v", releases)
	}
	if releases[0].Version != "v0.11.0" || releases[1].Version != "v0.10.0" {
		t.Fatalf("releases=%+v", releases)
	}
}

func TestNewerReleasesCapsTheStoredCount(t *testing.T) {
	candidates := make([]domain.ReleaseNote, 0, domain.MaxUpdateReleases+5)
	for index := range domain.MaxUpdateReleases + 5 {
		candidates = append(candidates, domain.ReleaseNote{Version: versionAt(index + 1)})
	}
	if got := len(domain.NewerReleases("0.0.1", candidates)); got != domain.MaxUpdateReleases {
		t.Fatalf("count=%d, want %d", got, domain.MaxUpdateReleases)
	}
}

// 開発ビルドのように semver として読めない現在バージョンでは、比較の相手がない。
// 案内を消すより、読めたリリースをすべて新しい候補として扱う。
func TestNewerReleasesKeepsEveryReleaseForUnreadableVersions(t *testing.T) {
	releases := domain.NewerReleases("0.4.0-dev", []domain.ReleaseNote{{Version: "v0.1.0"}})
	if len(releases) != 1 {
		t.Fatalf("releases=%+v", releases)
	}
}

// スキップは「そのバージョン以下を案内しない」意味なので、新しいタグで案内が戻る。
func TestShouldNotifyUpdateComparesAgainstTheSkippedVersion(t *testing.T) {
	tests := []struct {
		latest  string
		skipped string
		want    bool
	}{
		{latest: "v0.4.0", skipped: "", want: true},
		{latest: "v0.4.0", skipped: "v0.4.0", want: false},
		{latest: "v0.4.0", skipped: "v0.5.0", want: false},
		{latest: "v0.5.0", skipped: "v0.4.0", want: true},
		{latest: "v0.4.0", skipped: "nightly", want: true},
		{latest: "", skipped: "v0.4.0", want: false},
	}
	for _, test := range tests {
		if got := domain.ShouldNotifyUpdate(test.latest, test.skipped); got != test.want {
			t.Errorf("ShouldNotifyUpdate(%q, %q)=%t", test.latest, test.skipped, got)
		}
	}
}

func TestNewUpdateStatusReportsTheNewestRelease(t *testing.T) {
	published := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checked := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	status := domain.NewUpdateStatus(domain.UpdateStatusInput{
		CurrentVersion: "0.3.0",
		Releases: []domain.ReleaseNote{
			{Version: "v0.4.0", PublishedAt: &published, Body: "Notes"},
			{Version: "v0.2.0"},
		},
		SkippedVersion: "v0.3.5",
		LastCheckedAt:  &checked,
		LastError:      "",
	})
	if !status.Enabled || !status.UpdateAvailable || !status.ShouldNotify {
		t.Fatalf("status=%+v", status)
	}
	if status.LatestVersion != "v0.4.0" || len(status.Releases) != 1 {
		t.Fatalf("status=%+v", status)
	}
	if status.LastCheckedAt == nil || !status.LastCheckedAt.Equal(checked) {
		t.Fatalf("checked=%v", status.LastCheckedAt)
	}
}

func TestNewUpdateStatusStaysQuietWhenTheLatestIsSkipped(t *testing.T) {
	status := domain.NewUpdateStatus(domain.UpdateStatusInput{
		CurrentVersion: "0.3.0",
		Releases:       []domain.ReleaseNote{{Version: "v0.4.0"}},
		SkippedVersion: "v0.4.0",
	})
	if !status.UpdateAvailable || status.ShouldNotify {
		t.Fatalf("status=%+v", status)
	}
}

func TestNewUpdateStatusReportsNothingWhenUpToDate(t *testing.T) {
	status := domain.NewUpdateStatus(domain.UpdateStatusInput{
		CurrentVersion: "0.4.0",
		Releases:       []domain.ReleaseNote{{Version: "v0.4.0"}},
	})
	if status.UpdateAvailable || status.ShouldNotify || status.LatestVersion != "" {
		t.Fatalf("status=%+v", status)
	}
	if status.Releases == nil {
		t.Fatal("releases must be an empty collection, not nil")
	}
}

// 無効なときは確認結果もキャッシュも見せない。案内する材料がそもそもない。
func TestNewUpdateStatusHidesEverythingWhenDisabled(t *testing.T) {
	status := domain.NewUpdateStatus(domain.UpdateStatusInput{
		CurrentVersion: "0.4.0-dev",
		Disabled:       domain.UpdateDisabledDevelopmentBuild,
		Releases:       []domain.ReleaseNote{{Version: "v9.9.9"}},
		SkippedVersion: "v0.1.0",
		LastError:      "boom",
	})
	if status.Enabled || status.UpdateAvailable || len(status.Releases) != 0 {
		t.Fatalf("status=%+v", status)
	}
	if status.DisabledReason != domain.UpdateDisabledDevelopmentBuild || status.LastError != "" {
		t.Fatalf("status=%+v", status)
	}
}

func versionAt(minor int) string {
	return "v0." + strconv.Itoa(minor) + ".0"
}

// 稼働している常駐だけが置き換えを自分で検知する。停止中に再起動を促す意味はない。
func TestUpdateRestartRequiredFollowsTheDaemon(t *testing.T) {
	tests := []struct {
		name                          string
		supported, installed, running bool
		want                          bool
	}{
		{name: "no server running", supported: true, installed: true},
		{name: "managed daemon", supported: true, installed: true, running: true},
		{name: "unmanaged serve", supported: true, running: true, want: true},
		{name: "unsupported platform", running: true, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := domain.UpdateRestartRequired(test.supported, test.installed, test.running)
			if got != test.want {
				t.Fatalf("restart=%t, want %t", got, test.want)
			}
		})
	}
}
