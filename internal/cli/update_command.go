//go:build !noupdate

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/daemon"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/release"
	"github.com/HappyOnigiri/PRX/internal/tui"
	"github.com/HappyOnigiri/PRX/internal/update"
)

// 以下は差し替えられるよう変数にする。go test は配布元へ出られず、標準入力も端末ではない。
var (
	updateBuildVersion    = prx.Version
	updateReleaseProvider = func() release.Provider { return release.New() }
	updateApplier         = func() updateApplyFunc { return update.New().Apply }
	updateIsTerminal      = tui.IsTerminal
	updateOpenTTY         = func() (*os.File, error) { return os.Open("/dev/tty") }
	updateTerminal        = openUpdateTerminal
)

// updateApplyFunc は更新の実行そのもの。CLI は取得と bash の起動を internal/update に委ね、
// 自分では配置も検証も行わない。
type updateApplyFunc func(ctx context.Context, version string) (domain.UpdateApply, error)

type updateReleaseResponse struct {
	Version     string `json:"version"`
	PublishedAt string `json:"published_at,omitempty"`
	Body        string `json:"body,omitempty"`
	URL         string `json:"url,omitempty"`
}

// updateResponse は `prx update` の応答。スキップはサーバーが出す案内のための設定なので、
// 案内を出さない CLI では読みも書きもしない。
type updateResponse struct {
	Enabled         bool                    `json:"enabled"`
	DisabledReason  string                  `json:"disabled_reason,omitempty"`
	CurrentVersion  string                  `json:"current_version"`
	UpdateAvailable bool                    `json:"update_available"`
	LatestVersion   string                  `json:"latest_version,omitempty"`
	Releases        []updateReleaseResponse `json:"releases"`
	Applied         bool                    `json:"applied"`
	AppliedVersion  string                  `json:"applied_version,omitempty"`
	InstalledPath   string                  `json:"installed_path,omitempty"`
	RestartRequired bool                    `json:"restart_required"`
}

func (s *state) updateCommand() *cobra.Command {
	var apply bool
	command := &cobra.Command{
		Use:   "update",
		Short: "Check for a newer PRX release and install it",
		Long: "Check for a newer PRX release and install it.\n\n" +
			"The check reads the public GitHub releases of PRX without credentials, and the install runs the " +
			"install.sh attached to the release being installed.\n" +
			"Development builds and demo runs report that updates are disabled.",
		Example: "prx update",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return s.runUpdate(cmd.Context(), apply)
		},
	}
	command.Flags().BoolVar(&apply, "apply", false, "install the newest release without asking")
	return command
}

func (s *state) runUpdate(ctx context.Context, apply bool) error {
	version := updateBuildVersion()
	// --demo は serve だけのフラグなので、`prx update` が demo で走ることはない。
	// demo の無効化は RPC 経路が持つ。
	if domain.IsDevelopmentBuild(version) {
		return s.reportDisabledUpdate(version, apply)
	}
	releases, err := updateReleaseProvider().Releases(ctx)
	if err != nil {
		return domain.NewError(domain.DomainErrorCodeUpdateCheckFailed, "%s", err)
	}
	status := domain.NewUpdateStatus(domain.UpdateStatusInput{CurrentVersion: version, Releases: releases})
	if !status.UpdateAvailable {
		return s.write(updateResponseOf(status), renderUpdateStatus(status))
	}
	if !apply {
		confirmed, err := s.confirmUpdate(ctx, status)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}
	return s.applyUpdate(ctx, status)
}

// reportDisabledUpdate は確認だけなら状態として成功させ、--apply は失敗させる。
// できない操作を依頼されたときは、終了ステータスでもそれが分かる必要がある。
func (s *state) reportDisabledUpdate(version string, apply bool) error {
	if apply {
		return domain.NewError(domain.DomainErrorCodeUpdateUnavailable, "%s", updateDisabledMessage)
	}
	status := domain.NewUpdateStatus(domain.UpdateStatusInput{
		CurrentVersion: version, Disabled: domain.UpdateDisabledDevelopmentBuild,
	})
	return s.write(updateResponseOf(status), renderUpdateStatus(status))
}

const updateDisabledMessage = "updates are disabled for development builds"

// confirmUpdate は確認結果を見せてから 2 択で尋ねる。端末が無い、または JSON 出力では
// 尋ねず、確認結果だけを出して終わる。
func (s *state) confirmUpdate(ctx context.Context, status domain.UpdateStatus) (bool, error) {
	if s.json {
		return false, s.write(updateResponseOf(status), nil)
	}
	if err := renderUpdateStatus(status)(s.out); err != nil {
		return false, err
	}
	input, closeInput, ok := updateTerminal()
	if !ok {
		return false, nil
	}
	defer closeInput()
	answer, err := tui.Select(ctx, input, s.errOut, tui.Selection{
		Title:       "Install " + status.LatestVersion + "?",
		Description: "The installer replaces the prx binary in ~/.local/bin.",
		Initial:     0,
		Options: []tui.Option{
			{Value: "install", Label: "Install now", Description: "download and replace the prx binary"},
			{Value: "cancel", Label: "Not now", Description: "leave the installed version unchanged"},
		},
	})
	if err != nil {
		if errors.Is(err, tui.ErrCancelled) {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(answer) == "install", nil
}

// openUpdateTerminal は問いかけに使える端末を返す。見つからなければ尋ねずに終わる。
func openUpdateTerminal() (io.Reader, func(), bool) {
	input := os.Stdin
	if updateIsTerminal(int(input.Fd())) {
		return input, func() {}, true
	}
	tty, err := updateOpenTTY()
	if err != nil {
		return nil, nil, false
	}
	if !updateIsTerminal(int(tty.Fd())) {
		_ = tty.Close()
		return nil, nil, false
	}
	return tty, func() { _ = tty.Close() }, true
}

func (s *state) applyUpdate(ctx context.Context, status domain.UpdateStatus) error {
	applied, err := updateApplier()(ctx, status.LatestVersion)
	if err != nil {
		return domain.NewError(domain.DomainErrorCodeUpdateFailed, "%s", err)
	}
	result := domain.UpdateResult{
		Version:         applied.Version,
		InstalledPath:   applied.InstalledPath,
		RestartRequired: updateRestartRequired(daemon.Inspect(launchd.New(), status.CurrentVersion)),
	}
	response := updateResponseOf(status)
	response.Applied = true
	response.AppliedVersion = result.Version
	response.InstalledPath = result.InstalledPath
	response.RestartRequired = result.RestartRequired
	return s.write(response, renderUpdateApplied(result))
}

// updateRestartRequired は daemon の観測を domain の規則へ渡すだけの薄い口。
func updateRestartRequired(status daemon.Status) bool {
	return domain.UpdateRestartRequired(status.Supported, status.Installed, status.Running)
}

func updateResponseOf(status domain.UpdateStatus) updateResponse {
	releases := make([]updateReleaseResponse, 0, len(status.Releases))
	for _, note := range status.Releases {
		value := updateReleaseResponse{Version: note.Version, Body: note.Body, URL: note.URL}
		if note.PublishedAt != nil {
			value.PublishedAt = formatTime(*note.PublishedAt)
		}
		releases = append(releases, value)
	}
	return updateResponse{
		Enabled:         status.Enabled,
		DisabledReason:  string(status.DisabledReason),
		CurrentVersion:  status.CurrentVersion,
		UpdateAvailable: status.UpdateAvailable,
		LatestVersion:   status.LatestVersion,
		Releases:        releases,
	}
}

func renderUpdateStatus(status domain.UpdateStatus) humanRenderer {
	return func(out io.Writer) error {
		if !status.Enabled {
			return renderMessage("%s.", capitalize(updateDisabledMessage))(out)
		}
		if !status.UpdateAvailable {
			return renderMessage("PRX %s is up to date.", status.CurrentVersion)(out)
		}
		if _, err := fmt.Fprintf(
			out, "PRX %s is installed. %s is available.\n", status.CurrentVersion, status.LatestVersion,
		); err != nil {
			return err
		}
		for _, note := range status.Releases {
			if err := writeReleaseNote(out, note); err != nil {
				return err
			}
		}
		return nil
	}
}

func writeReleaseNote(out io.Writer, note domain.ReleaseNote) error {
	heading := note.Version
	if note.PublishedAt != nil {
		heading += " (" + formatTime(*note.PublishedAt) + ")"
	}
	if _, err := fmt.Fprintf(out, "\n%s\n", heading); err != nil {
		return err
	}
	if body := strings.TrimSpace(note.Body); body != "" {
		if _, err := fmt.Fprintf(out, "%s\n", body); err != nil {
			return err
		}
	}
	if note.URL == "" {
		return nil
	}
	_, err := fmt.Fprintf(out, "%s\n", note.URL)
	return err
}

func renderUpdateApplied(result domain.UpdateResult) humanRenderer {
	return func(out io.Writer) error {
		if _, err := fmt.Fprintf(out, "Installed prx %s to %s.\n", result.Version, result.InstalledPath); err != nil {
			return err
		}
		if !result.RestartRequired {
			return nil
		}
		return renderMessage("Restart the running prx serve to use the new version.")(out)
	}
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
