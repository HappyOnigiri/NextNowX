package app

// 日本語版のサンプルデータの文言。英語版と同じ並び・同じ件数にし、コマンド名・
// ステータス値・識別子は原語のまま残す。
// docs/design/persistence.md「初回のサンプルデータ」を参照。

func japaneseSampleText() sampleDataText {
	return sampleDataText{
		projectTitle:       "サンプル project",
		projectDescription: "PRX が feature とその作業をどうまとめるかを最初に見るための project。",
		featureTitle:       "サンプル feature",
		featureDescription: "6 件の task と、その間の依存関係。",
		documentTitle:      "はじめに",
		documentContent:    japaneseSampleDocument,
		planContent:        japaneseSamplePlan,
		tasks: []sampleTaskText{
			{title: "計画を描く", scope: "この feature で何を届けるかを決める"},
			{title: "作業環境を整える", scope: "道具とリポジトリを使える状態にする"},
			{title: "中心の流れを作る", scope: "他のすべてが必要とする経路を実装する"},
			{title: "ガイドを書く", scope: "使う人に向けてこの feature を説明する"},
			{title: "UI を仕上げる", scope: "中心の流れの上に画面を仕上げる"},
			{title: "リリースする", scope: "上の作業が入ったらリリースを切る"},
		},
	}
}

const japaneseSampleDocument = `# はじめに

PRX を入れた直後なので、このサンプル project が何を追跡するかを示している。

- task は status を持ち、表示状態は plan と pull request にも従う。
- 依存はどの task が先に入るかを表し、待っている task は blocked のまま残る。
- サンプル feature の task は 6 件で、completed が 2 件、in progress が 1 件、not started が 3 件。
- 「ガイドを書く」は implementation plan を持つので、ready のまま designed として見える。
- 「リリースする」は他のすべてを待つので blocked のまま残る。

不要になったらこの project を削除する。次回以降の投入を止めるには
prx setup --no-sample-data を使う。
`

const japaneseSamplePlan = `1. この feature に必要な手順を挙げる。
2. 読む人が最初に見る場所へ書く。
3. 結果を graph と照らして確かめる。
`
