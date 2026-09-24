//go:build !noupdate

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/HappyOnigiri/nnx/internal/domain"
	"github.com/HappyOnigiri/nnx/internal/release"
)

type updateHarness struct {
	out    bytes.Buffer
	errOut bytes.Buffer
	// applied は実行されたバージョン。実行されていなければ空。
	applied string
}

// stubUpdate は配布元と端末をテスト用に差し替える。既定は端末なしで、確認だけを行う。
func stubUpdate(t *testing.T, version string, provider release.Provider) *updateHarness {
	t.Helper()
	harness := &updateHarness{}
	previousVersion, previousProvider := updateBuildVersion, updateReleaseProvider
	previousApplier, previousTerminal := updateApplier, updateTerminal
	updateBuildVersion = func() string { return version }
	updateReleaseProvider = func() release.Provider { return provider }
	updateTerminal = func() (io.Reader, func(), bool) { return nil, nil, false }
	updateApplier = func() updateApplyFunc {
		return func(_ context.Context, target string) (domain.UpdateApply, error) {
			harness.applied = target
			return domain.UpdateApply{Version: target, InstalledPath: "/home/example/.local/bin/nnx"}, nil
		}
	}
	t.Cleanup(func() {
		updateBuildVersion, updateReleaseProvider = previousVersion, previousProvider
		updateApplier, updateTerminal = previousApplier, previousTerminal
	})
	return harness
}

func (h *updateHarness) run(t *testing.T, args ...string) error {
	t.Helper()
	return Execute(context.Background(), args, &h.out, &h.errOut, testOpenService)
}

func (h *updateHarness) decode(t *testing.T) updateResponse {
	t.Helper()
	var response updateResponse
	if err := json.Unmarshal(h.out.Bytes(), &response); err != nil {
		t.Fatalf("stdout=%q: %v", h.out.String(), err)
	}
	return response
}

func newReleaseProvider(versions ...string) release.Provider {
	published := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	notes := make([]domain.ReleaseNote, 0, len(versions))
	for _, version := range versions {
		notes = append(notes, domain.ReleaseNote{
			Version: version, PublishedAt: &published, Body: "Notes for " + version,
			URL: "https://example.test/" + version,
		})
	}
	return release.NewStaticProvider(notes)
}

func TestUpdateReportsAnUpToDateBuild(t *testing.T) {
	harness := stubUpdate(t, "0.4.0", newReleaseProvider("v0.4.0", "v0.3.0"))
	if err := harness.run(t, "update"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(harness.out.String(), "Next Now X 0.4.0 is up to date.") {
		t.Fatalf("stdout=%q", harness.out.String())
	}
	if harness.applied != "" {
		t.Fatalf("the check installed %q", harness.applied)
	}
}

// 端末が無ければ尋ねず、確認結果とリリース本文だけを出して終わる。
func TestUpdateWithoutATerminalOnlyReportsTheCheck(t *testing.T) {
	harness := stubUpdate(t, "0.3.0", newReleaseProvider("v0.4.0"))
	if err := harness.run(t, "update"); err != nil {
		t.Fatal(err)
	}
	output := harness.out.String()
	if !strings.Contains(output, "Next Now X 0.3.0 is installed. v0.4.0 is available.") {
		t.Fatalf("stdout=%q", output)
	}
	if !strings.Contains(output, "Notes for v0.4.0") || !strings.Contains(output, "https://example.test/v0.4.0") {
		t.Fatalf("stdout=%q", output)
	}
	if harness.applied != "" {
		t.Fatalf("the check installed %q", harness.applied)
	}
}

func TestUpdateJSONCarriesTheReleasesWithoutInstalling(t *testing.T) {
	harness := stubUpdate(t, "0.3.0", newReleaseProvider("v0.4.0"))
	if err := harness.run(t, "--json", "update"); err != nil {
		t.Fatal(err)
	}
	response := harness.decode(t)
	if !response.Enabled || !response.UpdateAvailable || response.LatestVersion != "v0.4.0" {
		t.Fatalf("response=%+v", response)
	}
	if len(response.Releases) != 1 || response.Releases[0].Body != "Notes for v0.4.0" {
		t.Fatalf("releases=%+v", response.Releases)
	}
	if response.Applied || harness.applied != "" {
		t.Fatalf("the JSON check installed %q", harness.applied)
	}
}

func TestUpdateApplyInstallsTheNewestRelease(t *testing.T) {
	harness := stubUpdate(t, "0.3.0", newReleaseProvider("v0.4.0", "v0.5.0"))
	if err := harness.run(t, "--json", "update", "--apply"); err != nil {
		t.Fatal(err)
	}
	if harness.applied != "v0.5.0" {
		t.Fatalf("applied=%q", harness.applied)
	}
	response := harness.decode(t)
	if !response.Applied || response.AppliedVersion != "v0.5.0" {
		t.Fatalf("response=%+v", response)
	}
	if response.InstalledPath != "/home/example/.local/bin/nnx" {
		t.Fatalf("response=%+v", response)
	}
}

func TestUpdateApplyRendersTheInstalledBinaryAsText(t *testing.T) {
	harness := stubUpdate(t, "0.3.0", newReleaseProvider("v0.4.0"))
	if err := harness.run(t, "update", "--apply"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(harness.out.String(), "Installed nnx v0.4.0 to /home/example/.local/bin/nnx.") {
		t.Fatalf("stdout=%q", harness.out.String())
	}
}

// 明示的に依頼した確認の失敗は、コマンドとして失敗させる。
func TestUpdateFailsWhenTheCheckFails(t *testing.T) {
	harness := stubUpdate(t, "0.3.0", release.NewFailingProvider(errors.New("the feed is unreachable")))
	err := harness.run(t, "--json", "update")
	if domain.ErrorCode(err) != domain.DomainErrorCodeUpdateCheckFailed {
		t.Fatalf("error=%v code=%q", err, domain.ErrorCode(err))
	}
	if !strings.Contains(harness.errOut.String(), `"code":"update_check_failed"`) {
		t.Fatalf("stderr=%q", harness.errOut.String())
	}
}

func TestUpdateReportsTheInstallerFailure(t *testing.T) {
	harness := stubUpdate(t, "0.3.0", newReleaseProvider("v0.4.0"))
	updateApplier = func() updateApplyFunc {
		return func(context.Context, string) (domain.UpdateApply, error) {
			return domain.UpdateApply{}, errors.New("checksum verification failed")
		}
	}
	err := harness.run(t, "update", "--apply")
	if domain.ErrorCode(err) != domain.DomainErrorCodeUpdateFailed {
		t.Fatalf("error=%v code=%q", err, domain.ErrorCode(err))
	}
}

// 開発ビルドでは確認も更新も行わない。確認だけなら状態として成功する。
func TestUpdateIsDisabledForDevelopmentBuilds(t *testing.T) {
	harness := stubUpdate(t, "0.4.0-dev", newReleaseProvider("v9.9.9"))
	if err := harness.run(t, "update"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(harness.out.String(), "Updates are disabled for development builds.") {
		t.Fatalf("stdout=%q", harness.out.String())
	}

	json := stubUpdate(t, "0.4.0-dev", newReleaseProvider("v9.9.9"))
	if err := json.run(t, "--json", "update"); err != nil {
		t.Fatal(err)
	}
	response := json.decode(t)
	if response.Enabled || response.DisabledReason != string(domain.UpdateDisabledDevelopmentBuild) {
		t.Fatalf("response=%+v", response)
	}
	if len(response.Releases) != 0 {
		t.Fatalf("the disabled build reported releases: %+v", response.Releases)
	}
}

// --apply は実行の依頼なので、無効なビルドでは終了ステータスでも失敗させる。
func TestUpdateApplyFailsOnDisabledBuilds(t *testing.T) {
	for _, args := range [][]string{{"update", "--apply"}, {"--json", "update", "--apply"}} {
		harness := stubUpdate(t, "0.4.0-dev", newReleaseProvider("v9.9.9"))
		err := harness.run(t, args...)
		if domain.ErrorCode(err) != domain.DomainErrorCodeUpdateUnavailable {
			t.Fatalf("args=%v error=%v code=%q", args, err, domain.ErrorCode(err))
		}
		if harness.applied != "" {
			t.Fatalf("args=%v installed %q", args, harness.applied)
		}
	}
}

// 端末があるときは確認結果を見せてから尋ねる。Enter で先頭の選択肢が確定する。
func TestUpdateAsksBeforeInstallingOnATerminal(t *testing.T) {
	tests := []struct {
		name  string
		keys  string
		want  string
		shown string
	}{
		{name: "accepted", keys: "\r", want: "v0.4.0", shown: "Installed nnx v0.4.0"},
		{name: "cancelled", keys: "\x1b[B\r", want: "", shown: "v0.4.0 is available."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := stubUpdate(t, "0.3.0", newReleaseProvider("v0.4.0"))
			updateTerminal = func() (io.Reader, func(), bool) {
				return strings.NewReader(test.keys), func() {}, true
			}
			if err := harness.run(t, "update"); err != nil {
				t.Fatal(err)
			}
			if harness.applied != test.want {
				t.Fatalf("applied=%q, want %q", harness.applied, test.want)
			}
			if !strings.Contains(harness.out.String(), test.shown) {
				t.Fatalf("stdout=%q", harness.out.String())
			}
		})
	}
}

// JSON 出力では尋ねない。対話と機械可読な出力は混ぜない。
func TestUpdateNeverAsksInJSONMode(t *testing.T) {
	harness := stubUpdate(t, "0.3.0", newReleaseProvider("v0.4.0"))
	updateTerminal = func() (io.Reader, func(), bool) {
		t.Fatal("the JSON output asked a question")
		return nil, nil, false
	}
	if err := harness.run(t, "--json", "update"); err != nil {
		t.Fatal(err)
	}
	if harness.decode(t).Applied {
		t.Fatal("the JSON check installed a release")
	}
}

func TestOpenUpdateTerminalFallsBackToTheControllingTerminal(t *testing.T) {
	previousTerminal, previousOpenTTY := updateIsTerminal, updateOpenTTY
	t.Cleanup(func() { updateIsTerminal, updateOpenTTY = previousTerminal, previousOpenTTY })

	updateIsTerminal = func(int) bool { return false }
	updateOpenTTY = func() (*os.File, error) { return nil, errors.New("no tty") }
	if _, _, ok := openUpdateTerminal(); ok {
		t.Fatal("a terminal was reported without one")
	}

	file, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	updateOpenTTY = func() (*os.File, error) { return file, nil }
	if _, _, ok := openUpdateTerminal(); ok {
		t.Fatal("a non-terminal file was accepted as a terminal")
	}

	// 標準入力が端末なら、そのまま問いかけに使う。
	updateIsTerminal = func(int) bool { return true }
	input, closeInput, ok := openUpdateTerminal()
	if !ok || input != os.Stdin {
		t.Fatalf("input=%v ok=%t", input, ok)
	}
	closeInput()
}
