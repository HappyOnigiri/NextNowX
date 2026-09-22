package cli

// 日本語版の setup の対話と進行メッセージ。英語版と同じ項目をすべて埋め、
// コマンド名・ラベル・URL は原語のまま残す。
// 書き分けの範囲は AGENTS.md「## 記載言語」を参照。

func japaneseSetupText() setupText {
	return setupText{
		sampleDataAdded: "サンプルデータを追加した: 小さな feature graph を持つ project 1 件。" +
			"入れずに始めるには prx setup --no-sample-data を使う。",
		daemonUnsupported: "PRX の daemon は macOS が必要。代わりに prx serve を直接実行する。",
		install: setupQuestionText{
			title:       "PRX を常駐サービスとして登録する？",
			description: "LaunchAgent はログイン時に PRX を自動で起動する。",
			accept: setupOptionText{
				label:       "daemon を登録",
				description: "ログイン時に PRX を起動し、応答するまで待つ",
			},
			decline: setupOptionText{
				label:       "手動で起動",
				description: "LaunchAgent を登録せず、必要なときに prx serve を使う",
			},
		},
		update: setupQuestionText{
			title:       "PRX の常駐サービスを更新する？",
			description: "LaunchAgent がこの PRX バイナリと一致していない。",
			accept: setupOptionText{
				label:       "daemon を更新",
				description: "LaunchAgent を書き直し、現在の PRX を起動する",
			},
			decline: setupOptionText{
				label:       "現状を維持",
				description: "既存の LaunchAgent をそのままにする",
			},
		},
		start: setupQuestionText{
			title:       "PRX の常駐サービスを起動する？",
			description: "LaunchAgent は登録済みだが、サーバーが動いていない。",
			accept: setupOptionText{
				label:       "daemon を起動",
				description: "今すぐ PRX を起動し、応答するまで待つ",
			},
			decline: setupOptionText{
				label:       "停止のまま",
				description: "サーバーは停止したままにする",
			},
		},
		open: setupQuestionText{
			title:       "PRX をブラウザで開く？",
			description: "WebUI は %s で待ち受けている。",
			accept: setupOptionText{
				label:       "ブラウザで開く",
				description: "PRX の WebUI を今すぐ開く",
			},
			decline: setupOptionText{
				label:       "端末のまま",
				description: "ブラウザは開かない",
			},
		},
		skippedInstall:   "daemon の登録を省略した。PRX を手動で起動するには prx serve を使う。",
		keptStalePlist:   "既存の LaunchAgent を残した。更新するには prx daemon install を使う。",
		leftStopped:      "PRX のサーバーは停止したままにした。必要になったら prx daemon start を使う。",
		alreadyRunning:   "PRX はセットアップ済みで、すでに動作している。",
		alreadyListening: "PRX はセットアップ済みで、%s で待ち受けている。",
		installedDaemon:  "%s を %s に登録した。PRX は %s で待ち受けている。",
		listening:        "PRX は %s で待ち受けている。",
		openedBrowser:    "%s を開いた。",
	}
}
