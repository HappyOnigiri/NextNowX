package cli_test

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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
// データベースを `prx snapshot` から読めることを確かめる。darwin では setup 自身が
// /dev/tty を開き、stdin のリダイレクトで問いかけを防げないので linux だけで走らせる。
func TestSetupSeedsSampleDataIntoTheRealDatabase(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("prx setup opens /dev/tty for its questions outside linux")
	}
	binary := buildCLI(t)
	root := t.TempDir()
	dbPath := filepath.Join(root, "prx.db")
	setup := executeCLIWithEnv(t, binary, "", sampleDataEnv(root), "--db", dbPath, "setup")
	if setup.exit != 0 {
		t.Fatalf("setup exit=%d stderr=%q", setup.exit, setup.stderr)
	}
	if !strings.Contains(setup.stdout, "Added sample data") {
		t.Fatalf("setup stdout=%q, want the sample data line", setup.stdout)
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
		t.Skip("prx setup opens /dev/tty for its questions outside linux")
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
			setup := executeCLIWithEnv(t, binary, "", append(sampleDataEnv(root), test.env...), args...)
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
