// Package update は確認済みのリリースに添付された install.sh を取得して実行する。
// checksum 検証・版番号検証・atomic な置換・PATH の案内はすべてインストーラーが
// 持つので、Go 側には持たない。方針は docs/design/updates.md にある。
package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// DefaultBaseURL はリリース資産の取得元。latest ではなく確認したタグを指すことで、
// 確認した版と実際に入る版を一致させる。
const DefaultBaseURL = "https://github.com/HappyOnigiri/PRX/releases/download"

const (
	// maxScriptBytes は取得するインストーラーの上限。
	maxScriptBytes = 1 << 20
	// downloadTimeout はインストーラーの取得にかける上限。
	downloadTimeout = 30 * time.Second
	// DefaultTimeout はインストーラーの実行にかける上限。取得と検証とダウンロードを
	// 含むので、確認の上限より長く取る。
	DefaultTimeout = 10 * time.Minute
	// fallbackPath は PATH が空の環境で install.sh が必要とするコマンドを探す場所。
	// launchd が渡す環境は極小なので、明示しないと常駐からの更新だけが失敗する。
	fallbackPath = "/usr/bin:/bin:/usr/sbin:/sbin"
)

// installedPattern は install.sh が報告する配置先を読み取る。標準以外の場所に
// 入れている環境では報告が出ないため、置き換わったかどうかの判定に使う。
// 配置先は行末まで取る。HOME に空白を含む環境でも報告を取りこぼさないためである。
var installedPattern = regexp.MustCompile(`(?m)^Installed prx (\S+) to (.+)$`)

// Applier は 1 つのリリースへ更新する実行器。
type Applier struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
	// run はインストーラーの実行だけを差し替える点。テストは bash を起動しない。
	run func(ctx context.Context, scriptPath string) (string, error)
}

// New は既定の取得元と実行方法を使う実行器を返す。
func New() *Applier { return NewWithOptions(DefaultBaseURL, nil, DefaultTimeout) }

// NewWithOptions は取得元・HTTP クライアント・実行の上限を差し替えられる実行器を返す。
func NewWithOptions(baseURL string, client *http.Client, timeout time.Duration) *Applier {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: downloadTimeout}
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	applier := &Applier{baseURL: baseURL, client: client, timeout: timeout}
	applier.run = applier.runInstaller
	return applier
}

// Apply は指定したタグの install.sh を取得して実行する。version は semver のタグでなければ
// ならない。取得元の URL に任意の文字列を差し込ませないための検査でもある。
func (a *Applier) Apply(ctx context.Context, version string) (domain.UpdateApply, error) {
	tag := domain.CanonicalVersion(version)
	if tag == "" {
		return domain.UpdateApply{}, fmt.Errorf("%q is not a release version", version)
	}
	script, err := a.download(ctx, tag)
	if err != nil {
		return domain.UpdateApply{}, err
	}
	directory, err := os.MkdirTemp("", "prx-update-")
	if err != nil {
		return domain.UpdateApply{}, fmt.Errorf("create update workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(directory) }()
	scriptPath := filepath.Join(directory, "install.sh")
	if err := os.WriteFile(scriptPath, script, 0o700); err != nil {
		return domain.UpdateApply{}, fmt.Errorf("stage installer: %w", err)
	}
	runContext, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	output, err := a.run(runContext, scriptPath)
	if err != nil {
		return domain.UpdateApply{}, installerError(err, output)
	}
	return applyResult(tag, output)
}

func (a *Applier) download(ctx context.Context, tag string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/install.sh", a.baseURL, tag)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build installer request: %w", err)
	}
	response, err := a.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download installer for %s: %w", tag, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download installer for %s: GitHub returned status %d", tag, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxScriptBytes))
	if err != nil {
		return nil, fmt.Errorf("download installer for %s: %w", tag, err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("the installer for %s is empty", tag)
	}
	return body, nil
}

// runInstaller は bash を起動する。標準入力は閉じ、環境は明示した最小限だけを渡す。
// launchd 配下の常駐は PATH も HOME も持たないまま起動されるためである。
func (a *Applier) runInstaller(ctx context.Context, scriptPath string) (string, error) {
	command := exec.CommandContext(ctx, "bash", scriptPath)
	command.Env = installerEnvironment()
	command.Stdin = nil
	// 標準入力を閉じるだけでは install.sh が呼ぶ `prx setup` が /dev/tty を開けてしまい、
	// 誰も答えない問いかけを上限時間まで待つ。制御端末ごと渡さない。
	command.SysProcAttr = installerProcessAttributes()
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	// 上限を超えたら bash を kill し、出力を握ったままの子プロセスを待ち続けない。
	// さもないと Run が返らず、更新 RPC が応答を返せなくなる。
	command.WaitDelay = time.Second
	err := command.Run()
	return output.String(), err
}

func installerEnvironment() []string {
	path := os.Getenv("PATH")
	if strings.TrimSpace(path) == "" {
		path = fallbackPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	environment := []string{"PATH=" + path, "HOME=" + home}
	if value := os.Getenv("TMPDIR"); value != "" {
		environment = append(environment, "TMPDIR="+value)
	}
	return environment
}

func installerError(err error, output string) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("the installer did not finish in time: %s", summarize(output))
	}
	return fmt.Errorf("the installer failed: %s", summarize(output))
}

// applyResult は置き換えが本当に起きたかを、インストーラー自身の報告で確かめる。
// 配置先は install.sh が固定しているので、別の場所に入れた版は更新されない。
func applyResult(tag, output string) (domain.UpdateApply, error) {
	match := installedPattern.FindStringSubmatch(output)
	if match == nil {
		return domain.UpdateApply{}, fmt.Errorf(
			"the installer did not report a replaced binary: %s", summarize(output),
		)
	}
	if domain.CanonicalVersion(match[1]) != tag {
		return domain.UpdateApply{}, fmt.Errorf("the installer reported %s instead of %s", match[1], tag)
	}
	return domain.UpdateApply{Version: tag, InstalledPath: strings.TrimSpace(match[2]), Output: output}, nil
}

// summarize はインストーラーの出力の末尾だけを残す。失敗の理由は最後の行にあり、
// 全文をエラーメッセージに載せると読み手が理由を見失う。
func summarize(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "the installer produced no output"
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > 5 {
		lines = lines[len(lines)-5:]
	}
	return strings.Join(lines, "; ")
}
