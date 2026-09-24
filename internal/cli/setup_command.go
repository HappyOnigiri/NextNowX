package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	nnx "github.com/HappyOnigiri/NextNowX"
	"github.com/HappyOnigiri/NextNowX/internal/browser"
	"github.com/HappyOnigiri/NextNowX/internal/config"
	"github.com/HappyOnigiri/NextNowX/internal/daemon"
	"github.com/HappyOnigiri/NextNowX/internal/launchd"
	"github.com/HappyOnigiri/NextNowX/internal/prompt"
	"github.com/HappyOnigiri/NextNowX/internal/runstate"
	"github.com/HappyOnigiri/NextNowX/internal/tui"
)

// 以下は差し替えられるよう変数にする。go test の stdin は端末ではないため、テストは
// 端末の判定と確保だけを置き換える。
var (
	setupIsTerminal = tui.IsTerminal
	setupOpenTTY    = func() (*os.File, error) { return os.Open("/dev/tty") }
	setupTerminal   = openSetupTerminal
)

type setupAction string

const (
	setupInstall setupAction = "install"
	setupUpdate  setupAction = "update"
	setupStart   setupAction = "start"
	setupSkip    setupAction = "skip"
	setupOpen    setupAction = "open"
	setupClose   setupAction = "close"
)

// setupSession は 1 回の setup が使う端末と文言を持つ。文言を持たせることで、
// 問いかけを出す関数が実効言語を引数で受け回さずに済む。
type setupSession struct {
	input  io.Reader
	out    io.Writer
	errOut io.Writer
	text   setupText
}

// sampleDataOptOutVariable は空でない値をすべて opt-out として扱う。値の語彙を
// 決めないのは、インストーラが真偽どちらの綴りを渡しても止まるほうが安全なため。
const sampleDataOptOutVariable = "NNX_NO_SAMPLE_DATA"

const sampleDataAddedMessage = "Added sample data: one project with a small feature graph. " +
	"Run nnx setup --no-sample-data to skip it."

func (s *state) setupCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "setup",
		Short: "Choose how Next Now X should start",
		Long: "Choose how Next Now X should start.\n\n" +
			"Setup first asks for a language and saves it as the language setting; its questions, " +
			"its progress messages, and the sample data follow that choice.\n" +
			"On macOS, the setup walk can register the LaunchAgent, start the background server, " +
			"and open the WebUI.\n" +
			"When it creates the database, setup adds a small sample project.",
		Example: "nnx setup",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if s.json {
				return errors.New("nnx setup does not support --json")
			}
			return s.runSetup(cmd.Context())
		},
	}
	// 永続フラグにすると、ストレージを見ないコマンドまで含めた全リファレンスの
	// Global Flags に出てしまうので setup のローカルフラグにする。
	command.Flags().BoolVar(
		&s.noSampleData,
		"no-sample-data",
		false,
		"skip the first-run sample data (env: "+sampleDataOptOutVariable+", any non-empty value)",
	)
	return command
}

func (s *state) runSetup(ctx context.Context) error {
	// 端末の確保は最初に 1 回だけ行う。言語の問いかけと常駐の問いかけで
	// /dev/tty を二度開かないためである。
	session, closeSession, hasTerminal := setupTerminal(s.out, s.errOut)
	defer closeSession()
	language, err := s.chooseSetupLanguage(ctx, session, hasTerminal)
	if err != nil {
		return err
	}
	session.text = setupTextFor(language)
	// 投入は macOS 判定より前に行う。常駐できない OS でも、端末が無い環境でも、
	// `nnx setup` を初回セットアップとして成立させるためである。
	s.seedSampleData(ctx, session.text)
	status := daemon.Inspect(launchd.New(), nnx.Version())
	if !status.Supported {
		return renderMessage("%s", session.text.daemonUnsupported)(s.out)
	}
	// 端末が要るのはここから先だけ。投入まで進んでから、従来と同じ形で断る。
	if !hasTerminal {
		return errors.New("nnx setup needs a terminal for its questions")
	}
	state, installedNow, err := s.applySetupDaemon(ctx, session, status)
	if err != nil {
		return err
	}
	if !shouldOfferSetupOpen(installedNow, state) {
		return nil
	}
	return s.offerSetupOpen(ctx, session, state.URL)
}

// seedSampleData は初回のサンプル投入を行う。setup の終了コードは install.sh が
// 「TUI が使えたか」の判定に使うので、投入の失敗は警告 1 行に留めて成功させる。
// docs/design/persistence.md「初回のサンプルデータ」を参照。
func (s *state) seedSampleData(ctx context.Context, text setupText) {
	if s.noSampleData || os.Getenv(sampleDataOptOutVariable) != "" {
		return
	}
	// root の PersistentPreRunE を通さず自分で開く。root 経由だと SyncIfDue が
	// `nnx setup` に乗り、オープン失敗が LaunchAgent の案内ごと消してしまう。
	service, closer, err := s.openService(config.WithPath(ctx, s.configPath), ServiceOptions{
		DatabasePath:       s.dbPath,
		DatabasePathSource: s.dbPathSource,
		ConfigPathSource:   s.configPathSource,
	})
	if err != nil {
		_, _ = fmt.Fprintf(s.errOut, "Warning: could not open the database for sample data: %s\n", err)
		return
	}
	// s.closer には渡さない。closeService は sync.Once なので、setup の後続で
	// 閉じ漏れと二重解放のどちらかを招く。
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}
	if service == nil {
		return
	}
	added, err := service.EnsureSampleData(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(s.errOut, "Warning: could not add the sample data: %s\n", err)
		return
	}
	if added {
		_, _ = fmt.Fprintln(s.out, text.sampleDataAdded)
	}
}

// chooseSetupLanguage は最初に言語を尋ね、選択を設定へ保存して実効言語を返す。端末が
// 無いとき、およびキャンセルされたときは尋ねず、設定とロケールから決めた言語を返す。
func (s *state) chooseSetupLanguage(
	ctx context.Context,
	session setupSession,
	hasTerminal bool,
) (prompt.Language, error) {
	current := s.effectiveSetupLanguage()
	if !hasTerminal {
		return current, nil
	}
	answer, err := tui.Select(ctx, session.input, session.errOut, setupLanguageSelection(current))
	if err != nil {
		// キャンセルで setup 全体を失敗させない。install.sh は終了ステータスだけで
		// 手動の案内へ落とすかを決めるので、言語を選ばなかっただけで誤報になる。
		if errors.Is(err, tui.ErrCancelled) {
			return current, nil
		}
		return "", err
	}
	chosen, ok := prompt.ParseLanguage(answer)
	if !ok {
		return current, nil
	}
	s.saveSetupLanguage(chosen)
	return chosen, nil
}

// setupLanguageSelection は選択肢を対応言語から組み立て、現在の実効言語を初期位置にする。
// auto は選択肢に出さない。選んだ後でロケール次第で変わると、サンプルの本文と噛み合わない。
func setupLanguageSelection(current prompt.Language) tui.Selection {
	question := setupLanguageQuestion()
	selection := tui.Selection{Title: question.title, Description: question.description}
	for index, language := range prompt.SupportedLanguages() {
		option := question.options[language]
		selection.Options = append(selection.Options, tui.Option{
			Value: string(language), Label: option.label, Description: option.description,
		})
		if language == current {
			selection.Initial = index
		}
	}
	return selection
}

// effectiveSetupLanguage は問いかけの前に使う実効言語を決める。設定を読めない初回でも
// setup は進むので、失敗はロケール解決へのフォールバックとして扱う。
func (s *state) effectiveSetupLanguage() prompt.Language {
	store, err := s.configStore()
	if err != nil {
		return prompt.ResolveLanguage(config.LanguageAutoValue)
	}
	settings, err := store.Load()
	if err != nil {
		return prompt.ResolveLanguage(config.LanguageAutoValue)
	}
	return settings.EffectiveLanguage()
}

// saveSetupLanguage は選択を現在の実効言語と同じでも必ず保存する。書かないと auto が
// 残り、ロケール変数の渡らない LaunchAgent で実効言語が端末とずれる。
func (s *state) saveSetupLanguage(language prompt.Language) {
	store, err := s.configStore()
	if err == nil {
		_, err = store.Update(func(settings *config.Config) error {
			return settings.SetLanguage(string(language))
		})
	}
	if err != nil {
		// 保存できなくても、そのセッションの表示には選んだ言語を使って進む。
		_, _ = fmt.Fprintf(s.errOut, "Warning: could not save the language setting: %s\n", err)
	}
}

// shouldOfferSetupOpen は daemon を新規導入した直後に URL が得られた場合だけブラウザを案内する。
func shouldOfferSetupOpen(installedNow bool, state runstate.State) bool {
	return installedNow && state.URL != ""
}

// openSetupTerminal は問いかけに使える端末を返す。見つからないことはここでは失敗に
// せず、端末を要る段まで setup を進めさせる。
func openSetupTerminal(out, errOut io.Writer) (setupSession, func(), bool) {
	session := setupSession{out: out, errOut: errOut}
	if input := os.Stdin; setupIsTerminal(int(input.Fd())) {
		session.input = input
		return session, func() {}, true
	}
	tty, err := setupOpenTTY()
	if err != nil {
		return session, func() {}, false
	}
	if !setupIsTerminal(int(tty.Fd())) {
		_ = tty.Close()
		return session, func() {}, false
	}
	session.input = tty
	return session, func() { _ = tty.Close() }, true
}

func (s *state) applySetupDaemon(
	ctx context.Context,
	session setupSession,
	status daemon.Status,
) (runstate.State, bool, error) {
	if !status.Installed || status.PlistStatus == launchd.PlistUnknown {
		action, err := selectSetupAction(ctx, session, setupChoice(session.text.install, setupInstall, setupSkip))
		if err != nil {
			return runstate.State{}, false, err
		}
		if action == setupSkip {
			_, _ = fmt.Fprintln(session.out, session.text.skippedInstall)
			return runstate.State{}, false, nil
		}
		state, err := s.installSetupDaemon(ctx, session)
		return state, true, err
	}
	if status.PlistStatus == launchd.PlistStale {
		action, err := selectSetupAction(ctx, session, setupChoice(session.text.update, setupUpdate, setupSkip))
		if err != nil {
			return runstate.State{}, false, err
		}
		if action == setupSkip {
			_, _ = fmt.Fprintln(session.out, setupSettledMessage(status, session.text))
			return status.State, false, nil
		}
		state, err := s.installSetupDaemon(ctx, session)
		return state, false, err
	}
	if status.Running {
		_, _ = fmt.Fprintln(session.out, setupSettledMessage(status, session.text))
		return status.State, false, nil
	}
	action, err := selectSetupAction(ctx, session, setupChoice(session.text.start, setupStart, setupSkip))
	if err != nil {
		return runstate.State{}, false, err
	}
	if action == setupSkip {
		_, _ = fmt.Fprintln(session.out, setupSettledMessage(status, session.text))
		return runstate.State{}, false, nil
	}
	state, err := s.startSetupDaemon(ctx, session)
	return state, false, err
}

// setupChoice は 2 択の問いかけを、承諾側を初期位置にして組み立てる。
func setupChoice(question setupQuestionText, accept, decline setupAction) tui.Selection {
	return tui.Selection{
		Title:       question.title,
		Description: question.description,
		Initial:     0,
		Options: []tui.Option{
			{Value: string(accept), Label: question.accept.label, Description: question.accept.description},
			{Value: string(decline), Label: question.decline.label, Description: question.decline.description},
		},
	}
}

// setupSettledMessage は何も変えずに終わる経路で残す 1 行を作る。稼働中でも待ち受け先を
// 読めないことがあるので、URL が無いときは待ち受け先に触れない。
func setupSettledMessage(status daemon.Status, text setupText) string {
	switch {
	case status.PlistStatus == launchd.PlistStale:
		return text.keptStalePlist
	case !status.Running:
		return text.leftStopped
	case status.AddressUnknown || status.State.URL == "":
		return text.alreadyRunning
	default:
		return fmt.Sprintf(text.alreadyListening, status.State.URL)
	}
}

func (s *state) installSetupDaemon(ctx context.Context, session setupSession) (runstate.State, error) {
	manager := launchd.New()
	if err := manager.Install(ctx); err != nil {
		return runstate.State{}, daemonError(err)
	}
	state, err := waitForRunState(ctx, func(runstate.State) bool { return true })
	if err != nil {
		return runstate.State{}, err
	}
	path, _ := manager.PlistPath()
	_, _ = fmt.Fprintf(session.out, session.text.installedDaemon+"\n", launchd.Label, path, state.URL)
	return state, nil
}

func (s *state) startSetupDaemon(ctx context.Context, session setupSession) (runstate.State, error) {
	manager := launchd.New()
	if err := manager.Start(ctx); err != nil {
		return runstate.State{}, daemonError(err)
	}
	state, err := waitForRunState(ctx, func(runstate.State) bool { return true })
	if err != nil {
		return runstate.State{}, err
	}
	_, _ = fmt.Fprintf(session.out, session.text.listening+"\n", state.URL)
	return state, nil
}

func (s *state) offerSetupOpen(ctx context.Context, session setupSession, url string) error {
	question := session.text.open
	question.description = fmt.Sprintf(question.description, url)
	action, err := selectSetupAction(ctx, session, setupChoice(question, setupOpen, setupClose))
	if err != nil {
		return err
	}
	if action == setupClose {
		return nil
	}
	opener := browser.New()
	if err := opener.Open(ctx, url); err != nil {
		return fmt.Errorf("open Next Now X in browser: %w", err)
	}
	_, _ = fmt.Fprintf(session.out, session.text.openedBrowser+"\n", url)
	return nil
}

func selectSetupAction(ctx context.Context, session setupSession, selection tui.Selection) (setupAction, error) {
	answer, err := tui.Select(ctx, session.input, session.errOut, selection)
	if err != nil {
		return "", err
	}
	return setupAction(strings.TrimSpace(answer)), nil
}
