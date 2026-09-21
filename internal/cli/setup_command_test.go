package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/daemon"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/runstate"
	"github.com/HappyOnigiri/PRX/internal/tui"
)

func TestSetupSettledMessage(t *testing.T) {
	tests := []struct {
		name   string
		status daemon.Status
		want   string
	}{
		{
			name: "stale plist kept",
			status: daemon.Status{
				Installed: true, PlistStatus: launchd.PlistStale,
				Running: true, State: runstate.State{URL: "http://127.0.0.1:7331"},
			},
			want: "Kept the existing LaunchAgent. Run prx daemon install to update it.",
		},
		{
			name:   "left stopped",
			status: daemon.Status{Installed: true, PlistStatus: launchd.PlistCurrent},
			want:   "Left the PRX server stopped. Run prx daemon start when you need it.",
		},
		{
			name: "already running",
			status: daemon.Status{
				Installed: true, PlistStatus: launchd.PlistCurrent,
				Running: true, State: runstate.State{URL: "http://127.0.0.1:7331"},
			},
			want: "PRX is already set up and listening on http://127.0.0.1:7331.",
		},
		{
			name: "running with unknown address",
			status: daemon.Status{
				Installed: true, PlistStatus: launchd.PlistCurrent,
				Running: true, AddressUnknown: true,
			},
			want: "PRX is already set up and running.",
		},
		{
			name: "running without recorded url",
			status: daemon.Status{
				Installed: true, PlistStatus: launchd.PlistCurrent, Running: true,
			},
			want: "PRX is already set up and running.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := setupSettledMessage(test.status); got != test.want {
				t.Fatalf("message=%q, want %q", got, test.want)
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

// runSetupWithoutTerminal は端末を持たない環境の setup を走らせる。darwin では
// TUI の問いかけに入る前に終端し、他の OS では macOS 判定で終わる。
func runSetupWithoutTerminal(
	t *testing.T,
	args []string,
	open OpenService,
) (stdout, stderr string, err error) {
	t.Helper()
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
