package app_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/release"
	"github.com/HappyOnigiri/PRX/internal/store"
)

type updateEnvironment struct {
	service     *app.Service
	configStore *config.Store
}

func newUpdateService(t *testing.T, provider app.ReleaseProvider, updater app.Updater) updateEnvironment {
	t.Helper()
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "update.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	configStore, err := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := configStore.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	service := app.NewWithConfig(database, nil, configStore)
	service.SetUpdateSources(provider, updater)
	return updateEnvironment{service: service, configStore: configStore}
}

type countingProvider struct {
	releases []domain.ReleaseNote
	err      error
	calls    int
}

func (p *countingProvider) Releases(context.Context) ([]domain.ReleaseNote, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return append([]domain.ReleaseNote{}, p.releases...), nil
}

type recordingUpdater struct {
	version string
	result  domain.UpdateApply
	err     error
}

func (u *recordingUpdater) Apply(_ context.Context, version string) (domain.UpdateApply, error) {
	u.version = version
	if u.err != nil {
		return domain.UpdateApply{}, u.err
	}
	return u.result, nil
}

// 確認は保存され、間引きが切れるまで配布元へは出ない。
func TestUpdateStatusChecksOnceWithinTheInterval(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	provider := &countingProvider{releases: []domain.ReleaseNote{{Version: "v0.4.0", Body: "Notes"}}}
	environment := newUpdateService(t, provider, nil)
	ctx := context.Background()

	status, err := environment.service.GetUpdateStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Enabled || !status.UpdateAvailable || !status.ShouldNotify {
		t.Fatalf("status=%+v", status)
	}
	if status.LatestVersion != "v0.4.0" || status.LastCheckedAt == nil {
		t.Fatalf("status=%+v", status)
	}

	cached, err := environment.service.GetUpdateStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("checks=%d, want 1", provider.calls)
	}
	if len(cached.Releases) != 1 || cached.Releases[0].Body != "Notes" {
		t.Fatalf("cached=%+v", cached)
	}
}

// 確認の失敗は記録するだけで応答は成功させ、直前の結果を残す。
func TestUpdateStatusKeepsTheCacheWhenTheCheckFails(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	provider := &countingProvider{releases: []domain.ReleaseNote{{Version: "v0.4.0"}}}
	environment := newUpdateService(t, provider, nil)
	ctx := context.Background()
	if _, err := environment.service.GetUpdateStatus(ctx); err != nil {
		t.Fatal(err)
	}
	environment.service.SetNowForTest(func() time.Time { return time.Now().UTC().Add(48 * time.Hour) })
	provider.err = errors.New("the feed is unreachable")

	status, err := environment.service.GetUpdateStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.LastError == "" || len(status.Releases) != 1 {
		t.Fatalf("status=%+v", status)
	}
}

// スキップは設定に残り、より新しいタグが出れば案内が戻る。
func TestSkipUpdateVersionSilencesOnlyThatVersion(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	provider := &countingProvider{releases: []domain.ReleaseNote{{Version: "v0.4.0"}}}
	environment := newUpdateService(t, provider, nil)
	ctx := context.Background()
	if _, err := environment.service.GetUpdateStatus(ctx); err != nil {
		t.Fatal(err)
	}

	status, err := environment.service.SkipUpdateVersion(ctx, "v0.4.0")
	if err != nil {
		t.Fatal(err)
	}
	if status.ShouldNotify || !status.UpdateAvailable {
		t.Fatalf("status=%+v", status)
	}
	settings, err := environment.configStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Update.SkippedVersion != "v0.4.0" {
		t.Fatalf("config=%+v", settings.Update)
	}

	environment.service.SetNowForTest(func() time.Time { return time.Now().UTC().Add(48 * time.Hour) })
	provider.releases = append(provider.releases, domain.ReleaseNote{Version: "v0.5.0"})
	revived, err := environment.service.GetUpdateStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !revived.ShouldNotify || revived.LatestVersion != "v0.5.0" {
		t.Fatalf("status=%+v", revived)
	}
}

func TestSkipUpdateVersionRejectsValuesThatAreNotReleaseTags(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	environment := newUpdateService(t, &countingProvider{}, nil)
	if _, err := environment.service.SkipUpdateVersion(context.Background(), "nightly"); err == nil {
		t.Fatal("expected an invalid version to be rejected")
	}
}

func TestApplyUpdateRunsTheUpdaterForNewerReleasesOnly(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	updater := &recordingUpdater{
		result: domain.UpdateApply{Version: "v0.4.0", InstalledPath: "/home/example/.local/bin/prx"},
	}
	environment := newUpdateService(t, &countingProvider{}, updater)
	ctx := context.Background()

	result, err := environment.service.ApplyUpdate(ctx, "v0.4.0")
	if err != nil {
		t.Fatal(err)
	}
	if updater.version != "v0.4.0" || result.InstalledPath != "/home/example/.local/bin/prx" {
		t.Fatalf("result=%+v version=%q", result, updater.version)
	}
	// 常駐が観測できない配線では、置き換えが自動で反映される保証がない。
	if !result.RestartRequired {
		t.Fatalf("result=%+v", result)
	}
	for _, version := range []string{"v0.2.0", "0.3.0", "latest", ""} {
		if _, err := environment.service.ApplyUpdate(ctx, version); err == nil {
			t.Fatalf("version %q was accepted", version)
		}
	}
}

// launchd 配下の常駐は置き換えを自分で検知するので、利用者の再起動は要らない。
func TestApplyUpdateLeavesTheRestartToTheManagedDaemon(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	updater := &recordingUpdater{result: domain.UpdateApply{Version: "v0.4.0", InstalledPath: "/bin/prx"}}
	environment := newUpdateService(t, &countingProvider{}, updater)
	environment.service.SetDaemonInspector(func(context.Context) domain.DebugDaemonInput {
		return domain.DebugDaemonInput{Supported: true, Installed: true, Running: true}
	})
	result, err := environment.service.ApplyUpdate(context.Background(), "v0.4.0")
	if err != nil || result.RestartRequired {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestApplyUpdateReportsInstallerFailures(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	updater := &recordingUpdater{err: errors.New("checksum verification failed")}
	environment := newUpdateService(t, &countingProvider{}, updater)
	_, err := environment.service.ApplyUpdate(context.Background(), "v0.4.0")
	if domain.ErrorCode(err) != domain.DomainErrorCodeUpdateFailed {
		t.Fatalf("error=%v code=%q", err, domain.ErrorCode(err))
	}
}

// 開発ビルドは配布物ではないので、確認も適用も行わない。
func TestUpdateIsDisabledForDevelopmentBuilds(t *testing.T) {
	provider := &countingProvider{releases: []domain.ReleaseNote{{Version: "v9.9.9"}}}
	environment := newUpdateService(t, provider, &recordingUpdater{})
	ctx := context.Background()

	status, err := environment.service.GetUpdateStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Enabled || status.DisabledReason != domain.UpdateDisabledDevelopmentBuild {
		t.Fatalf("status=%+v", status)
	}
	if provider.calls != 0 {
		t.Fatalf("the disabled build reached the release feed %d times", provider.calls)
	}
	if _, err := environment.service.ApplyUpdate(ctx, "v9.9.9"); domain.ErrorCode(err) !=
		domain.DomainErrorCodeUpdateUnavailable {
		t.Fatalf("error=%v", err)
	}
	if _, err := environment.service.SkipUpdateVersion(ctx, "v9.9.9"); domain.ErrorCode(err) !=
		domain.DomainErrorCodeUpdateUnavailable {
		t.Fatalf("error=%v", err)
	}
}

// demo は一時環境で、E2E が実ネットワークへ出てはならない。
func TestUpdateIsDisabledInDemoMode(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	provider := &countingProvider{releases: []domain.ReleaseNote{{Version: "v9.9.9"}}}
	environment := newUpdateService(t, provider, nil)
	environment.service.SetProcessInfo(app.ProcessInfo{Mode: "serve", Demo: true})

	status, err := environment.service.GetUpdateStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Enabled || status.DisabledReason != domain.UpdateDisabledDemo || provider.calls != 0 {
		t.Fatalf("status=%+v checks=%d", status, provider.calls)
	}
}

// 配線が無い構成でも、確認の失敗として記録するだけで応答は返す。
func TestUpdateStatusRecordsAMissingReleaseProvider(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	environment := newUpdateService(t, nil, nil)
	status, err := environment.service.GetUpdateStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.LastError == "" {
		t.Fatalf("status=%+v", status)
	}
	if _, err := environment.service.ApplyUpdate(context.Background(), "v9.9.9"); domain.ErrorCode(err) !=
		domain.DomainErrorCodeUpdateUnavailable {
		t.Fatalf("error=%v", err)
	}
}

// 静的な provider は、実ネットワークへ出てはならない経路のために置く。
func TestUpdateStatusAcceptsAStaticProvider(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	provider := release.NewStaticProvider([]domain.ReleaseNote{{Version: "v0.4.0"}})
	environment := newUpdateService(t, provider, nil)
	status, err := environment.service.GetUpdateStatus(context.Background())
	if err != nil || status.LatestVersion != "v0.4.0" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
}
