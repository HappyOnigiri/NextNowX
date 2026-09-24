package cli

// 日本語版の setup の対話と進行メッセージ。英語版と同じ項目をすべて埋め、
// コマンド名・ラベル・URL は原語のまま残す。
// 書き分けの範囲は AGENTS.md「## 記載言語」を参照。

func japaneseSetupText() setupText {
	return setupText{
		sampleDataAdded: "サンプルデータを追加した: 小さな feature graph を持つ project 1 件。" +
			"入れずに始めるには nnx setup --no-sample-data を使う。",
		daemonUnsupported: "Next Now X の daemon は macOS が必要。代わりに nnx serve を直接実行する。",
		install: setupQuestionText{
			title:       "Next Now X を常駐サービスとして登録する？",
			description: "LaunchAgent はログイン時に Next Now X を自動で起動する。",
			accept: setupOptionText{
				label:       "daemon を登録",
				description: "ログイン時に Next Now X を起動し、応答するまで待つ",
			},
			decline: setupOptionText{
				label:       "手動で起動",
				description: "LaunchAgent を登録せず、必要なときに nnx serve を使う",
			},
		},
		update: setupQuestionText{
			title:       "Next Now X の常駐サービスを更新する？",
			description: "LaunchAgent がこの Next Now X バイナリと一致していない。",
			accept: setupOptionText{
				label:       "daemon を更新",
				description: "LaunchAgent を書き直し、現在の Next Now X を起動する",
			},
			decline: setupOptionText{
				label:       "現状を維持",
				description: "既存の LaunchAgent をそのままにする",
			},
		},
		start: setupQuestionText{
			title:       "Next Now X の常駐サービスを起動する？",
			description: "LaunchAgent は登録済みだが、サーバーが動いていない。",
			accept: setupOptionText{
				label:       "daemon を起動",
				description: "今すぐ Next Now X を起動し、応答するまで待つ",
			},
			decline: setupOptionText{
				label:       "停止のまま",
				description: "サーバーは停止したままにする",
			},
		},
		open: setupQuestionText{
			title:       "Next Now X をブラウザで開く？",
			description: "WebUI は %s で待ち受けている。",
			accept: setupOptionText{
				label:       "ブラウザで開く",
				description: "Next Now X の WebUI を今すぐ開く",
			},
			decline: setupOptionText{
				label:       "端末のまま",
				description: "ブラウザは開かない",
			},
		},
		skippedInstall:   "daemon の登録を省略した。Next Now X を手動で起動するには nnx serve を使う。",
		keptStalePlist:   "既存の LaunchAgent を残した。更新するには nnx daemon install を使う。",
		leftStopped:      "Next Now X のサーバーは停止したままにした。必要になったら nnx daemon start を使う。",
		alreadyRunning:   "Next Now X はセットアップ済みで、すでに動作している。",
		alreadyListening: "Next Now X はセットアップ済みで、%s で待ち受けている。",
		installedDaemon:  "%s を %s に登録した。Next Now X は %s で待ち受けている。",
		listening:        "Next Now X は %s で待ち受けている。",
		openedBrowser:    "%s を開いた。",
	}
}
