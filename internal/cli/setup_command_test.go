package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/HappyOnigiri/PRX/internal/daemon"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/prompt"
	"github.com/HappyOnigiri/PRX/internal/runstate"
	"github.com/HappyOnigiri/PRX/internal/tui"
)

// TestSetupSettledMessage は分岐ごとの 1 行を両言語で確かめる。日本語の期待値を
// 並べることで、英語のまま取り残された分岐を機械的に見つけられる。
func TestSetupSettledMessage(t *testing.T) {
	tests := []struct {
		name   string
		status daemon.Status
		wantEn string
		wantJa string
	}{
		{
			name: "stale plist kept",
			status: daemon.Status{
				Installed: true, PlistStatus: launchd.PlistStale,
				Running: true, State: runstate.State{URL: "http://127.0.0.1:7331"},
			},
			wantEn: "Kept the existing LaunchAgent. Run prx daemon install to update it.",
			wantJa: "既存の LaunchAgent を残した。更新するには prx daemon install を使う。",
		},
		{
			name:   "left stopped",
			status: daemon.Status{Installed: true, PlistStatus: launchd.PlistCurrent},
			wantEn: "Left the PRX server stopped. Run prx daemon start when you need it.",
			wantJa: "PRX のサーバーは停止したままにした。必要になったら prx daemon start を使う。",
		},
		{
			name: "already running",
			status: daemon.Status{
				Installed: true, PlistStatus: launchd.PlistCurrent,
				Running: true, State: runstate.State{URL: "http://127.0.0.1:7331"},
			},
			wantEn: "PRX is already set up and listening on http://127.0.0.1:7331.",
			wantJa: "PRX はセットアップ済みで、http://127.0.0.1:7331 で待ち受けている。",
		},
		{
			name: "running with unknown address",
			status: daemon.Status{
				Installed: true, PlistStatus: launchd.PlistCurrent,
				Running: true, AddressUnknown: true,
			},
			wantEn: "PRX is already set up and running.",
			wantJa: "PRX はセットアップ済みで、すでに動作している。",
		},
		{
			name: "running without recorded url",
			status: daemon.Status{
				Installed: true, PlistStatus: launchd.PlistCurrent, Running: true,
			},
			wantEn: "PRX is already set up and running.",
			wantJa: "PRX はセットアップ済みで、すでに動作している。",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := setupSettledMessage(test.status, englishSetupText()); got != test.wantEn {
				t.Fatalf("english message=%q, want %q", got, test.wantEn)
			}
			if got := setupSettledMessage(test.status, japaneseSetupText()); got != test.wantJa {
				t.Fatalf("japanese message=%q, want %q", got, test.wantJa)
			}
		})
	}
}

// sampleDataService は cli.Service を埋め込み、サンプル投入だけを実装する。
// 埋め込みの nil メソッドは呼ばれた時点で panic するので、setup がここ以外の
// 境界に触れていないことも同時に確かめられる。
type sampleDataService struct {
	Service
	added bool
	err   error
	calls int
}

func (s *sampleDataService) EnsureSampleData(context.Context) (bool, error) {
	s.calls++
	return s.added, s.err
}

// isolateSetupConfig は設定ファイルと実効言語を固定し、開発者のホームにある設定や
// ロケールが setup の文言を変えないようにする。書き込み先のパスを返す。
func isolateSetupConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("PRX_CONFIG", path)
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(name, "en_US.UTF-8")
	}
	return path
}

// runSetupWithoutTerminal は端末を持たない環境の setup を走らせる。darwin では
// TUI の問いかけに入る前に終端し、他の OS では macOS 判定で終わる。
func runSetupWithoutTerminal(
	t *testing.T,
	args []string,
	open OpenService,
) (stdout, stderr string, err error) {
	t.Helper()
	isolateSetupConfig(t)
	previousTerminal := setupIsTerminal
	previousOpenTTY := setupOpenTTY
	setupIsTerminal = func(int) bool { return false }
	setupOpenTTY = func() (*os.File, error) { return nil, errors.New("no tty") }
	t.Cleanup(func() {
		setupIsTerminal = previousTerminal
		setupOpenTTY = previousOpenTTY
	})
	var out, errOut bytes.Buffer
	err = Execute(context.Background(), append([]string{"setup"}, args...), &out, &errOut, open)
	return out.String(), errOut.String(), err
}

// runSetupWithInput は問いかけに使える端末があるものとして setup を走らせる。端末の
// 確保ごと差し替えるのは、go test の stdin を端末に見せかけられないためである。
func runSetupWithInput(
	t *testing.T,
	input string,
	open OpenService,
) (stdout, stderr string, err error) {
	t.Helper()
	previous := setupTerminal
	setupTerminal = func(out, errOut io.Writer) (setupSession, func(), bool) {
		return setupSession{input: strings.NewReader(input), out: out, errOut: errOut}, func() {}, true
	}
	t.Cleanup(func() { setupTerminal = previous })
	var out, errOut bytes.Buffer
	err = Execute(context.Background(), []string{"setup"}, &out, &errOut, open)
	return out.String(), errOut.String(), err
}

// savedSetupLanguage は setup が書いた設定の language を読む。ファイルが無ければ空を返す。
func savedSetupLanguage(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Language string `yaml:"language"`
	}
	if err := yaml.Unmarshal(body, &saved); err != nil {
		t.Fatal(err)
	}
	return saved.Language
}

// TestSetupAsksForTheLanguageFirst は、下矢印 + Enter が日本語を選び、その言語が
// 設定へ残り、同じ実行のサンプル投入まで進むことを確かめる。darwin では言語の後に
// 常駐の問いかけへ入り、実機の LaunchAgent に触れるので走らせない。
func TestSetupAsksForTheLanguageFirst(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("the setup walk continues into the real LaunchAgent on darwin")
	}
	path := isolateSetupConfig(t)
	service := &sampleDataService{added: true}
	stdout, stderr, err := runSetupWithInput(t, "\x1b[B\r",
		func(context.Context, ServiceOptions) (Service, io.Closer, error) {
			return service, nil, nil
		},
	)
	if err != nil {
		t.Fatalf("error=%v stderr=%q", err, stderr)
	}
	if got := savedSetupLanguage(t, path); got != "ja" {
		t.Fatalf("saved language=%q, want ja", got)
	}
	if service.calls != 1 {
		t.Fatalf("EnsureSampleData calls=%d, want 1", service.calls)
	}
	if !strings.Contains(stdout, japaneseSetupText().sampleDataAdded) {
		t.Fatalf("stdout=%q, want the japanese sample data line", stdout)
	}
	if !strings.Contains(stdout, japaneseSetupText().daemonUnsupported) {
		t.Fatalf("stdout=%q, want the japanese macOS line", stdout)
	}
}

// TestChooseSetupLanguage は問いかけの結果と設定への保存を、setup の残りから切り
// 離して確かめる。実機の LaunchAgent に触れないので、どの OS でも走る。
func TestChooseSetupLanguage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  prompt.Language
		saved string
	}{
		// 初期位置は現在の実効言語。ロケールを英語に固定しているので en が選ばれる。
		{name: "enter keeps the current language", input: "\r", want: prompt.LanguageEnglish, saved: "en"},
		{name: "down selects japanese", input: "\x1b[B\r", want: prompt.LanguageJapanese, saved: "ja"},
		// キャンセルは設定を書かない。auto のままなので次回の setup でまた尋ねる。
		{name: "escape cancels", input: "\x1b", want: prompt.LanguageEnglish, saved: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := isolateSetupConfig(t)
			var errOut bytes.Buffer
			s := &state{out: io.Discard, errOut: &errOut}
			session := setupSession{input: strings.NewReader(test.input), out: io.Discard, errOut: io.Discard}
			got, err := s.chooseSetupLanguage(context.Background(), session, true)
			if err != nil {
				t.Fatalf("error=%v", err)
			}
			if got != test.want {
				t.Fatalf("language=%q, want %q", got, test.want)
			}
			if saved := savedSetupLanguage(t, path); saved != test.saved {
				t.Fatalf("saved language=%q, want %q", saved, test.saved)
			}
			if errOut.Len() != 0 {
				t.Fatalf("stderr=%q, want no warning", errOut.String())
			}
		})
	}
}

// TestChooseSetupLanguageWithoutATerminal は、端末が無いときは尋ねず、設定も書かず、
// 設定とロケールから決めた言語をそのまま使うことを確かめる。
func TestChooseSetupLanguageWithoutATerminal(t *testing.T) {
	path := isolateSetupConfig(t)
	s := &state{out: io.Discard, errOut: io.Discard}
	got, err := s.chooseSetupLanguage(context.Background(), setupSession{}, false)
	if err != nil || got != prompt.LanguageEnglish {
		t.Fatalf("language=%q err=%v", got, err)
	}
	if saved := savedSetupLanguage(t, path); saved != "" {
		t.Fatalf("saved language=%q, want no config file", saved)
	}
}

// TestSetupTextsDifferBetweenLanguages は、文言一式のどの項目も空でなく、言語ごとに
// 違うことを確かめる。英語のまま取り残された項目の検出が目的である。
func TestSetupTextsDifferBetweenLanguages(t *testing.T) {
	english := reflect.ValueOf(englishSetupText())
	japanese := reflect.ValueOf(japaneseSetupText())
	var walk func(t *testing.T, name string, left, right reflect.Value)
	walk = func(t *testing.T, name string, left, right reflect.Value) {
		t.Helper()
		if left.Kind() == reflect.Struct {
			for index := range left.NumField() {
				walk(t, name+"."+left.Type().Field(index).Name, left.Field(index), right.Field(index))
			}
			return
		}
		if left.String() == "" || right.String() == "" {
			t.Errorf("%s is empty in one of the languages", name)
			return
		}
		if left.String() == right.String() {
			t.Errorf("%s is the same in both languages: %q", name, left.String())
		}
	}
	walk(t, "setupText", english, japanese)
}

// assertSetupWalkOutcome は、投入とは無関係な setup 本体の結果を確かめる。端末が
// 無いので darwin だけが失敗し、投入の成否はこの結果に混ざってはならない。
func assertSetupWalkOutcome(t *testing.T, err error) {
	t.Helper()
	if runtime.GOOS == "darwin" {
		if err == nil || !strings.Contains(err.Error(), "needs a terminal") {
			t.Fatalf("error=%v", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("error=%v", err)
	}
}

func TestSetupAddsSampleDataOnANewDatabase(t *testing.T) {
	service := &sampleDataService{added: true}
	stdout, stderr, err := runSetupWithoutTerminal(t, nil,
		func(context.Context, ServiceOptions) (Service, io.Closer, error) {
			return service, nil, nil
		},
	)
	assertSetupWalkOutcome(t, err)
	if service.calls != 1 {
		t.Fatalf("EnsureSampleData calls=%d, want 1", service.calls)
	}
	if !strings.Contains(stdout, sampleDataAddedMessage) {
		t.Fatalf("stdout=%q, want the sample data line", stdout)
	}
	if strings.Contains(stderr, "Warning") {
		t.Fatalf("stderr=%q, want no warning", stderr)
	}
}

// TestSetupStaysQuietWhenTheDatabaseAlreadyExists は、既存データベースでは
// 1 行も出さないことを確かめる。投入したときだけ知らせる契約である。
func TestSetupStaysQuietWhenTheDatabaseAlreadyExists(t *testing.T) {
	service := &sampleDataService{}
	stdout, _, err := runSetupWithoutTerminal(t, nil,
		func(context.Context, ServiceOptions) (Service, io.Closer, error) {
			return service, nil, nil
		},
	)
	assertSetupWalkOutcome(t, err)
	if service.calls != 1 {
		t.Fatalf("EnsureSampleData calls=%d, want 1", service.calls)
	}
	if strings.Contains(stdout, "sample data") {
		t.Fatalf("stdout=%q, want no sample data line", stdout)
	}
}

func TestSetupOptOutDoesNotOpenService(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  string
	}{
		{name: "flag", args: []string{"--no-sample-data"}},
		{name: "environment", env: "1"},
		// 真偽の綴りを決めないので、opt-out を否定する語でも止まる。
		{name: "environment false spelling", env: "false"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.env != "" {
				t.Setenv(sampleDataOptOutVariable, test.env)
			}
			opened := false
			_, _, err := runSetupWithoutTerminal(t, test.args,
				func(context.Context, ServiceOptions) (Service, io.Closer, error) {
					opened = true
					return nil, nil, nil
				},
			)
			assertSetupWalkOutcome(t, err)
			if opened {
				t.Fatal("setup opened the application service")
			}
		})
	}
}

// TestSetupSurvivesSampleDataFailures は、データベースを開けない場合も投入が
// 失敗した場合も setup 自体は成功することを確かめる。install.sh は setup の
// 終了コードだけで TUI が使えたかを判断する。
func TestSetupSurvivesSampleDataFailures(t *testing.T) {
	tests := []struct {
		name string
		open OpenService
		want string
	}{
		{
			name: "open fails",
			open: func(context.Context, ServiceOptions) (Service, io.Closer, error) {
				return nil, nil, errors.New("database is locked")
			},
			want: "could not open the database for sample data: database is locked",
		},
		{
			name: "write fails",
			open: func(context.Context, ServiceOptions) (Service, io.Closer, error) {
				return &sampleDataService{err: errors.New("disk is full")}, nil, nil
			},
			want: "could not add the sample data: disk is full",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, stderr, err := runSetupWithoutTerminal(t, nil, test.open)
			assertSetupWalkOutcome(t, err)
			if !strings.Contains(stderr, test.want) {
				t.Fatalf("stderr=%q, want it to contain %q", stderr, test.want)
			}
		})
	}
}

func TestSetupRejectsJSONOutput(t *testing.T) {
	var errOut bytes.Buffer
	err := Execute(context.Background(), []string{"--json", "setup"}, io.Discard, &errOut, testOpenService)
	if err == nil || !strings.Contains(err.Error(), "does not support --json") {
		t.Fatalf("error=%v", err)
	}
	if !strings.Contains(errOut.String(), `"code":"`) {
		t.Fatalf("stderr=%q", errOut.String())
	}
}

func TestSelectSetupActionUsesTUISelection(t *testing.T) {
	selection := tui.Selection{
		Title: "Choose", Initial: 0,
		Options: []tui.Option{{Value: "install", Label: "Install"}, {Value: "skip", Label: "Skip"}},
	}
	got, err := selectSetupAction(context.Background(), setupSession{
		input: strings.NewReader("\x1b[B\r"), errOut: io.Discard,
	}, selection)
	if err != nil || got != setupSkip {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestSetupOffersBrowserOnlyAfterInstallingDaemon(t *testing.T) {
	tests := []struct {
		name         string
		installedNow bool
		url          string
		want         bool
	}{
		{name: "new daemon", installedNow: true, url: "http://127.0.0.1:7331", want: true},
		{name: "started daemon", installedNow: false, url: "http://127.0.0.1:7331", want: false},
		{name: "no address", installedNow: true, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldOfferSetupOpen(test.installedNow, runstate.State{URL: test.url}); got != test.want {
				t.Fatalf("offer=%t, want %t", got, test.want)
			}
		})
	}
}
