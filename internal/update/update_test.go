//go:build !noupdate

package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func newTestApplier(t *testing.T, body string, run func(string) (string, error)) *Applier {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/install.sh") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	applier := NewWithOptions(server.URL, server.Client(), time.Minute)
	if run != nil {
		applier.run = func(_ context.Context, scriptPath string) (string, error) {
			return run(scriptPath)
		}
	}
	return applier
}

// インストーラーは確認したタグの資産から取り、その報告で置き換えを確かめる。
func TestApplyRunsTheInstallerForTheRequestedTag(t *testing.T) {
	var staged string
	applier := newTestApplier(t, "#!/bin/bash\n", func(scriptPath string) (string, error) {
		staged = scriptPath
		return "Downloading prx v0.4.0...\nInstalled prx v0.4.0 to /home/example/.local/bin/prx\n", nil
	})
	result, err := applier.Apply(context.Background(), "0.4.0")
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "v0.4.0" || result.InstalledPath != "/home/example/.local/bin/prx" {
		t.Fatalf("result=%+v", result)
	}
	if !strings.HasSuffix(staged, "install.sh") {
		t.Fatalf("script path=%q", staged)
	}
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatalf("the staged installer was left behind: %v", err)
	}
}

// 標準以外の場所へ入れている環境では置き換えが起きないので、成功と報告してはならない。
func TestApplyFailsWhenTheInstallerReplacedNothing(t *testing.T) {
	applier := newTestApplier(t, "#!/bin/bash\n", func(string) (string, error) {
		return "Downloading prx v0.4.0...\n", nil
	})
	_, err := applier.Apply(context.Background(), "v0.4.0")
	if err == nil || !strings.Contains(err.Error(), "did not report a replaced binary") {
		t.Fatalf("error=%v", err)
	}
}

func TestApplyFailsWhenTheInstallerReportsAnotherVersion(t *testing.T) {
	applier := newTestApplier(t, "#!/bin/bash\n", func(string) (string, error) {
		return "Installed prx v0.3.0 to /home/example/.local/bin/prx\n", nil
	})
	_, err := applier.Apply(context.Background(), "v0.4.0")
	if err == nil || !strings.Contains(err.Error(), "instead of v0.4.0") {
		t.Fatalf("error=%v", err)
	}
}

func TestApplyReportsTheInstallerFailureTail(t *testing.T) {
	applier := newTestApplier(t, "#!/bin/bash\n", func(string) (string, error) {
		return strings.Repeat("noise\n", 8) + "prx install: checksum verification failed\n", errStub
	})
	_, err := applier.Apply(context.Background(), "v0.4.0")
	if err == nil || !strings.Contains(err.Error(), "checksum verification failed") {
		t.Fatalf("error=%v", err)
	}
	if strings.Count(err.Error(), "noise") > 4 {
		t.Fatalf("the whole installer log was reported: %v", err)
	}
}

func TestApplyRejectsValuesThatAreNotReleaseTags(t *testing.T) {
	applier := newTestApplier(t, "#!/bin/bash\n", func(string) (string, error) {
		t.Fatal("the installer ran for an invalid version")
		return "", nil
	})
	for _, version := range []string{"", "latest", "../../etc", "v1.2.3-rc.1"} {
		if _, err := applier.Apply(context.Background(), version); err == nil {
			t.Fatalf("version %q was accepted", version)
		}
	}
}

// HOME に空白を含む環境でも置き換えの報告は読み取れなければならない。
func TestApplyReadsInstallPathsThatContainSpaces(t *testing.T) {
	applier := newTestApplier(t, "#!/bin/bash\n", func(string) (string, error) {
		return "Installed prx v0.4.0 to /Users/Ada Lovelace/.local/bin/prx\n", nil
	})
	result, err := applier.Apply(context.Background(), "v0.4.0")
	if err != nil {
		t.Fatal(err)
	}
	if result.InstalledPath != "/Users/Ada Lovelace/.local/bin/prx" {
		t.Fatalf("installed path=%q", result.InstalledPath)
	}
}

func TestApplyFailsWhenTheInstallerCannotBeDownloaded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	applier := NewWithOptions(server.URL, server.Client(), 0)
	if _, err := applier.Apply(context.Background(), "v0.4.0"); err == nil {
		t.Fatal("expected a download failure")
	}
}

// 切り詰めたスクリプトを実行すると、取得の破損が無関係な失敗として現れる。
func TestApplyFailsOnAnOversizedInstaller(t *testing.T) {
	applier := newTestApplier(t, strings.Repeat("#", maxScriptBytes+1), nil)
	_, err := applier.Apply(context.Background(), "v0.4.0")
	if err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("error=%v", err)
	}
}

func TestApplyFailsOnAnEmptyInstaller(t *testing.T) {
	applier := newTestApplier(t, "", nil)
	if _, err := applier.Apply(context.Background(), "v0.4.0"); err == nil {
		t.Fatal("expected an empty installer to fail")
	}
}

// 常駐サーバーの環境は launchd が渡す極小のものなので、PATH と HOME は明示して渡す。
func TestInstallerEnvironmentAlwaysCarriesPathAndHome(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("TMPDIR", "/tmp/prx-test")
	environment := installerEnvironment()
	joined := strings.Join(environment, "\n")
	if !strings.Contains(joined, "PATH="+fallbackPath) {
		t.Fatalf("environment=%v", environment)
	}
	if !strings.Contains(joined, "HOME=") || !strings.Contains(joined, "TMPDIR=/tmp/prx-test") {
		t.Fatalf("environment=%v", environment)
	}
}

// プロキシ必須の環境では、スクリプトの中の curl も同じ設定を要る。
func TestInstallerEnvironmentForwardsProxySettings(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://proxy.example:3128")
	t.Setenv("NO_PROXY", "localhost")
	t.Setenv("SSL_CERT_FILE", "/etc/ssl/company.pem")
	t.Setenv("GITHUB_TOKEN", "secret")
	joined := strings.Join(installerEnvironment(), "\n")
	for _, want := range []string{
		"HTTPS_PROXY=http://proxy.example:3128", "NO_PROXY=localhost", "SSL_CERT_FILE=/etc/ssl/company.pem",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("environment=%q want %q", joined, want)
		}
	}
	if strings.Contains(joined, "GITHUB_TOKEN") {
		t.Fatalf("an unrelated secret reached the installer: %q", joined)
	}
}

// 既定の実行器は bash を起動する。端末を持たせないことが、インストーラーの
// セットアップ TUI が応答を待ち続ける事態を防ぐ。
func TestRunInstallerExecutesTheStagedScript(t *testing.T) {
	applier := New()
	directory := t.TempDir()
	script := directory + "/install.sh"
	body := "#!/bin/bash\nif [ -t 0 ]; then echo tty; fi\necho Installed prx v0.4.0 to $HOME/.local/bin/prx\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := applier.run(context.Background(), script)
	if err != nil {
		t.Fatalf("error=%v output=%q", err, output)
	}
	if strings.Contains(output, "tty") {
		t.Fatalf("the installer was given a terminal: %q", output)
	}
	if !strings.Contains(output, "Installed prx v0.4.0 to ") {
		t.Fatalf("output=%q", output)
	}
}

func TestSummarizeDescribesSilentInstallers(t *testing.T) {
	if got := summarize("  \n "); got != "the installer produced no output" {
		t.Fatalf("summary=%q", got)
	}
}

var errStub = &stubError{}

type stubError struct{}

func (*stubError) Error() string { return "exit status 1" }
