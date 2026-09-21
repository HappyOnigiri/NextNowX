package app

import (
	"context"
	"errors"
	"time"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

const (
	// updateCheckIntervalSeconds は自動確認の間引き。案内のためだけに未認証の API を
	// 頻繁に叩かない。
	updateCheckIntervalSeconds = int64(24 * 60 * 60)
	// updateCheckTimeout は日和見の確認を打ち切る上限。画面の読み込みを待たせない。
	updateCheckTimeout = 20 * time.Second
)

// updateBuildVersion は差し替えられるよう変数にする。go test は常に -dev のビルドで
// 動くので、有効な経路を検証するにはリリースビルドを名乗る必要がある。
var updateBuildVersion = prx.Version

// GetUpdateStatus は保存済みの確認結果を返し、間引きが切れていればその場で確認する。
// 確認の失敗は記録するだけで、応答自体は成功させる。
func (s *Service) GetUpdateStatus(ctx context.Context) (domain.UpdateStatus, error) {
	if disabled := s.updateDisabledReason(); disabled != domain.UpdateEnabled {
		return domain.NewUpdateStatus(domain.UpdateStatusInput{
			CurrentVersion: updateBuildVersion(), Disabled: disabled,
		}), nil
	}
	repository, err := s.updateCheckRepository()
	if err != nil {
		return domain.UpdateStatus{}, err
	}
	now := s.now().UTC()
	acquired, err := repository.AcquireUpdateCheck(ctx, now, saturatedSubtract(now.Unix(), updateCheckIntervalSeconds))
	if err != nil {
		return domain.UpdateStatus{}, err
	}
	if !acquired {
		return s.storedUpdateStatus(ctx, repository)
	}
	releases, checkErr := s.checkReleases(ctx)
	// 保存する一覧は表示する一覧とそろえる。上限も新旧の絞り込みもここで掛からないと、
	// 取得したページ全体の本文がそのまま 1 行に溜まる。
	releases = domain.NewerReleases(updateBuildVersion(), releases)
	// 確認できなかったときは直前の結果を残す。オフラインでもモーダルは開ける。
	previous, previousErr := repository.UpdateCheckState(ctx)
	runError := ""
	if checkErr != nil {
		// 直前の結果を読めないまま保存すると、残すはずのキャッシュを空で上書きする。
		if previousErr != nil {
			return domain.UpdateStatus{}, errors.Join(checkErr, previousErr)
		}
		runError = checkErr.Error()
		releases = previous.Releases
	}
	recordContext, cancel := recordingContext(ctx)
	defer cancel()
	if err := repository.CompleteUpdateCheck(recordContext, s.now().UTC(), releases, runError); err != nil {
		return domain.UpdateStatus{}, errors.Join(previousErr, err)
	}
	return s.storedUpdateStatus(recordContext, repository)
}

// SkipUpdateVersion は案内しないバージョンを設定に記録する。より新しいタグが出れば
// 案内は自動的に戻るので、指定は消さない。
func (s *Service) SkipUpdateVersion(ctx context.Context, version string) (domain.UpdateStatus, error) {
	if disabled := s.updateDisabledReason(); disabled != domain.UpdateEnabled {
		return domain.UpdateStatus{}, updateDisabledError(disabled)
	}
	if s.configStore == nil {
		return domain.UpdateStatus{}, domain.NewError(
			domain.DomainErrorCodeInvalidConfig, "configuration is unavailable",
		)
	}
	if _, err := s.configStore.Update(func(settings *config.Config) error {
		return settings.SetSkippedUpdateVersion(version)
	}); err != nil {
		return domain.UpdateStatus{}, configDomainError(err)
	}
	repository, err := s.updateCheckRepository()
	if err != nil {
		return domain.UpdateStatus{}, err
	}
	return s.storedUpdateStatus(ctx, repository)
}

// ApplyUpdate は対象タグのインストーラーを実行する。現在より新しいタグだけを受け付け、
// 取得元の URL に利用者の入力をそのまま差し込ませない。
func (s *Service) ApplyUpdate(ctx context.Context, version string) (domain.UpdateResult, error) {
	if disabled := s.updateDisabledReason(); disabled != domain.UpdateEnabled {
		return domain.UpdateResult{}, updateDisabledError(disabled)
	}
	if s.updater == nil {
		return domain.UpdateResult{}, domain.NewError(
			domain.DomainErrorCodeUpdateUnavailable, "the updater is unavailable in this build",
		)
	}
	target := domain.CanonicalVersion(version)
	if target == "" {
		return domain.UpdateResult{}, domain.NewError(
			domain.DomainErrorCodeUpdateFailed, "%q is not a release version", version,
		)
	}
	current := updateBuildVersion()
	if len(domain.NewerReleases(current, []domain.ReleaseNote{{Version: target}})) == 0 {
		return domain.UpdateResult{}, domain.NewError(
			domain.DomainErrorCodeUpdateFailed, "%s is not newer than the running version %s", target, current,
		)
	}
	applied, err := s.updater.Apply(ctx, target)
	if err != nil {
		return domain.UpdateResult{}, domain.NewError(domain.DomainErrorCodeUpdateFailed, "%s", err)
	}
	return domain.UpdateResult{
		Version:         applied.Version,
		InstalledPath:   applied.InstalledPath,
		RestartRequired: s.updateRestartRequired(ctx),
	}, nil
}

// updateDisabledReason は機能全体の有効性を決める。開発ビルドは配布物ではなく、
// demo は一時環境で実ネットワークにも出ない。ビルド時の除外を先に見るのは、
// provider 未注入が「利用不能」という失敗として記録されるより先に返すためである。
func (s *Service) updateDisabledReason() domain.UpdateDisabledReason {
	if reason := updateBuildDisabledReason(); reason != domain.UpdateEnabled {
		return reason
	}
	switch {
	case domain.IsDevelopmentBuild(updateBuildVersion()):
		return domain.UpdateDisabledDevelopmentBuild
	case s.processInfo.Demo:
		return domain.UpdateDisabledDemo
	default:
		return domain.UpdateEnabled
	}
}

func updateDisabledError(reason domain.UpdateDisabledReason) error {
	message := "updates are disabled for development builds"
	switch reason {
	case domain.UpdateDisabledDemo:
		message = "updates are disabled in demo mode"
	case domain.UpdateDisabledExcludedFromBuild:
		message = "updates are not included in this build"
	case domain.UpdateEnabled, domain.UpdateDisabledDevelopmentBuild:
	}
	return domain.NewError(domain.DomainErrorCodeUpdateUnavailable, "%s", message)
}

func (s *Service) updateCheckRepository() (UpdateCheckStateRepository, error) {
	repository, ok := s.repository.(UpdateCheckStateRepository)
	if !ok {
		return nil, domain.NewError(domain.DomainErrorCodeInternal, "update state is unavailable")
	}
	return repository, nil
}

func (s *Service) checkReleases(ctx context.Context) ([]domain.ReleaseNote, error) {
	if s.releases == nil {
		return nil, errors.New("the release feed is unavailable in this build")
	}
	checkContext, cancel := context.WithTimeout(ctx, updateCheckTimeout)
	defer cancel()
	return s.releases.Releases(checkContext)
}

func (s *Service) storedUpdateStatus(
	ctx context.Context,
	repository UpdateCheckStateRepository,
) (domain.UpdateStatus, error) {
	state, err := repository.UpdateCheckState(ctx)
	if err != nil {
		return domain.UpdateStatus{}, err
	}
	return domain.NewUpdateStatus(domain.UpdateStatusInput{
		CurrentVersion: updateBuildVersion(),
		Releases:       state.Releases,
		SkippedVersion: s.skippedUpdateVersion(),
		LastCheckedAt:  state.LastCheckedAt,
		LastError:      state.Error,
	}), nil
}

// skippedUpdateVersion は設定を読めないときを「スキップなし」と同じに扱う。
// 設定の破損で案内が消えるより、案内が出るほうが安全である。
func (s *Service) skippedUpdateVersion() string {
	if s.configStore == nil {
		return ""
	}
	settings, err := s.configStore.Load()
	if err != nil {
		return ""
	}
	return settings.Update.SkippedVersion
}

// updateRestartRequired は CLI と同じ規則で再起動の要否を決める。常駐を観測できない
// 配線では、この RPC に届いている以上サーバーは動いているので、再起動を伝える。
func (s *Service) updateRestartRequired(ctx context.Context) bool {
	if s.daemonInspector == nil {
		return true
	}
	status := s.daemonInspector(ctx)
	return domain.UpdateRestartRequired(status.Supported, status.Installed, status.Running)
}
