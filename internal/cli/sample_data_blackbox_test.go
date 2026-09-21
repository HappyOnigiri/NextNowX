package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runDetachedSetup は制御端末を持たないセッションで `prx setup` を起こす。stdin を
// リダイレクトするだけでは制御端末が残り、setup が /dev/tty を開いて言語の問いかけで
// 止まるためである。
func runDetachedSetup(t *testing.T, binary string, env []string, args ...string) commandOutput {
	t.Helper()
	command := exec.CommandContext(context.Background(), binary, args...)
	command.Env = append(os.Environ(), env...)
	command.SysProcAttr = detachedProcessAttributes()
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	// Stdin を渡さないと null デバイスが割り当たるので、問いかけは入力を得られない。
	err := command.Run()
	exit := 0
	if err != nil {
		var typed *exec.ExitError
		if errors.As(err, &typed) {
			exit = typed.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return commandOutput{stdout: stdout.String(), stderr: stderr.String(), exit: exit}
}

type sampleSnapshot struct {
	Projects []struct{} `json:"projects"`
	Features []struct{} `json:"features"`
	Tasks    []struct {
		Title        string `json:"title"`
		DisplayState string `json:"display_state"`
	} `json:"tasks"`
	Documents    []struct{} `json:"documents"`
	PullRequests []struct{} `json:"pull_requests"`
}

// TestSetupSeedsSampleDataIntoTheRealDatabase は、別プロセスの `prx setup` が作った
// データベースを `prx snapshot` から読めることを確かめる。darwin では端末が無いと
// 常駐の問いかけに入る前に失敗するので linux だけで走らせる。
func TestSetupSeedsSampleDataIntoTheRealDatabase(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("prx setup needs a terminal for its daemon questions outside linux")
	}
	binary := buildCLI(t)
	root := t.TempDir()
	dbPath := filepath.Join(root, "prx.db")
	setup := runDetachedSetup(t, binary, sampleDataEnv(root), "--db", dbPath, "setup")
	if setup.exit != 0 {
		t.Fatalf("setup exit=%d stderr=%q", setup.exit, setup.stderr)
	}
	if !strings.Contains(setup.stdout, "Added sample data") {
		t.Fatalf("setup stdout=%q, want the sample data line", setup.stdout)
	}
	// 端末が無いので言語は尋ねない。問いかけの見出しが出ていないことで確かめる。
	if strings.Contains(setup.stderr, "Language / 言語") {
		t.Fatalf("setup stderr=%q, want no language question", setup.stderr)
	}
	snapshot := readSampleSnapshot(t, binary, dbPath)
	if len(snapshot.Projects) != 1 || len(snapshot.Features) != 1 ||
		len(snapshot.Tasks) != 6 || len(snapshot.Documents) != 2 {
		t.Fatalf("snapshot=%+v, want 1 project, 1 feature, 6 tasks and 2 documents", snapshot)
	}
	if len(snapshot.PullRequests) != 0 {
		t.Errorf("pull requests=%d, want none", len(snapshot.PullRequests))
	}
	designed := ""
	for _, task := range snapshot.Tasks {
		if task.DisplayState == "designed" {
			designed = task.Title
		}
	}
	if designed != "Write the guide" {
		t.Errorf("designed task=%q, want the task that carries the implementation plan", designed)
	}
}

func TestSetupSampleDataOptOutLeavesTheDatabaseEmpty(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("prx setup needs a terminal for its daemon questions outside linux")
	}
	binary := buildCLI(t)
	for _, test := range []struct {
		name string
		args []string
		env  []string
	}{
		{name: "flag", args: []string{"--no-sample-data"}},
		{name: "environment", env: []string{"PRX_NO_SAMPLE_DATA=1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			dbPath := filepath.Join(root, "prx.db")
			args := append([]string{"--db", dbPath, "setup"}, test.args...)
			setup := runDetachedSetup(t, binary, append(sampleDataEnv(root), test.env...), args...)
			if setup.exit != 0 {
				t.Fatalf("setup exit=%d stderr=%q", setup.exit, setup.stderr)
			}
			if strings.Contains(setup.stdout, "sample data") {
				t.Fatalf("setup stdout=%q, want no sample data line", setup.stdout)
			}
			snapshot := readSampleSnapshot(t, binary, dbPath)
			if len(snapshot.Projects) != 0 || len(snapshot.Tasks) != 0 {
				t.Fatalf("snapshot=%+v, want an empty database", snapshot)
			}
		})
	}
}

// sampleDataEnv は実効言語を英語に固定する。ロケールだけでは足りない。設定の
// language が auto でないマシンでは、設定ファイルのほうが実効言語を決めるためである。
func sampleDataEnv(root string) []string {
	return append(append([]string{}, englishLocale...), "PRX_CONFIG="+filepath.Join(root, "config.yaml"))
}

func readSampleSnapshot(t *testing.T, binary, dbPath string) sampleSnapshot {
	t.Helper()
	result, stderr, exit := runCLI(t, binary, dbPath, "snapshot")
	if exit != 0 || !result.OK {
		t.Fatalf("snapshot exit=%d stderr=%q", exit, stderr)
	}
	var snapshot sampleSnapshot
	if err := json.Unmarshal(result.Data, &snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}
