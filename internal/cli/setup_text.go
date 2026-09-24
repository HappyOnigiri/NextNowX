package cli

import "github.com/HappyOnigiri/nnx/internal/prompt"

// setupOptionText は選択肢 1 つ分の文言。
type setupOptionText struct {
	label       string
	description string
}

// setupQuestionText は問いかけ 1 つ分の文言。setup の問いかけはどれも 2 択で、
// accept が既定の選択肢、decline が何もしない選択肢である。
type setupQuestionText struct {
	title       string
	description string
	accept      setupOptionText
	decline     setupOptionText
}

// setupText は setup の問いかけと進行メッセージの文言一式。実効言語ごとに 1 つ用意する。
// 文言の書き分けの範囲は AGENTS.md「## 記載言語」を参照。
type setupText struct {
	sampleDataAdded   string
	daemonUnsupported string
	install           setupQuestionText
	update            setupQuestionText
	start             setupQuestionText
	open              setupQuestionText
	skippedInstall    string
	keptStalePlist    string
	leftStopped       string
	alreadyRunning    string
	alreadyListening  string
	installedDaemon   string
	listening         string
	openedBrowser     string
}

func setupTextFor(language prompt.Language) setupText {
	if language == prompt.LanguageJapanese {
		return japaneseSetupText()
	}
	return englishSetupText()
}

// setupLanguageText は言語そのものを尋ねる 1 画面。選択肢のラベルは自言語表記にする。
type setupLanguageText struct {
	title       string
	description string
	options     map[prompt.Language]setupOptionText
}

// setupLanguageQuestion の見出しと説明は、まだ実効言語を決めていない段で読まれるので、
// どちらの話者も読めるよう二言語併記にする。
func setupLanguageQuestion() setupLanguageText {
	return setupLanguageText{
		title:       "Language / 言語",
		description: "Setup and the sample data use this language. / セットアップとサンプルデータでこの言語を使う。",
		options: map[prompt.Language]setupOptionText{
			prompt.LanguageEnglish: {
				label:       "English",
				description: "use English for setup and the sample data",
			},
			prompt.LanguageJapanese: {
				label:       "日本語",
				description: "セットアップとサンプルデータで日本語を使う",
			},
		},
	}
}

func englishSetupText() setupText {
	return setupText{
		sampleDataAdded:   sampleDataAddedMessage,
		daemonUnsupported: "The Next Now X daemon requires macOS. Run nnx serve directly instead.",
		install: setupQuestionText{
			title:       "Install Next Now X as a background service?",
			description: "The LaunchAgent starts Next Now X automatically when you log in.",
			accept: setupOptionText{
				label:       "Install daemon",
				description: "start Next Now X at login and wait until it is ready",
			},
			decline: setupOptionText{
				label:       "Run manually",
				description: "do not install a LaunchAgent; use nnx serve when needed",
			},
		},
		update: setupQuestionText{
			title:       "Update the Next Now X background service?",
			description: "The LaunchAgent does not match this Next Now X binary.",
			accept: setupOptionText{
				label:       "Update daemon",
				description: "rewrite the LaunchAgent and start the current Next Now X",
			},
			decline: setupOptionText{
				label:       "Keep current",
				description: "leave the existing LaunchAgent unchanged",
			},
		},
		start: setupQuestionText{
			title:       "Start the Next Now X background service?",
			description: "The LaunchAgent is installed but the server is not running.",
			accept: setupOptionText{
				label:       "Start daemon",
				description: "start Next Now X now and wait until it is ready",
			},
			decline: setupOptionText{
				label:       "Leave stopped",
				description: "leave the server stopped for now",
			},
		},
		open: setupQuestionText{
			title:       "Open Next Now X in your browser?",
			description: "The WebUI is ready at %s.",
			accept: setupOptionText{
				label:       "Open browser",
				description: "open the Next Now X WebUI now",
			},
			decline: setupOptionText{
				label:       "Keep terminal",
				description: "leave the browser closed",
			},
		},
		skippedInstall:   "Skipped daemon installation. Run nnx serve to start Next Now X manually.",
		keptStalePlist:   "Kept the existing LaunchAgent. Run nnx daemon install to update it.",
		leftStopped:      "Left the Next Now X server stopped. Run nnx daemon start when you need it.",
		alreadyRunning:   "Next Now X is already set up and running.",
		alreadyListening: "Next Now X is already set up and listening on %s.",
		installedDaemon:  "Installed %s at %s. Next Now X is listening on %s.",
		listening:        "Next Now X is listening on %s.",
		openedBrowser:    "Opened %s.",
	}
}
