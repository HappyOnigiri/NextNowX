# エージェントプロンプト方針

PRX は PRX を知らないエージェントに task を渡すため、prompt は task の識別子と、それを操作するために必要なコマンドを含んでいなければならない。
task prompt や batch prompt のレンダリングは、task の状態・readiness・依存関係・implementation plan のいずれも変更しない。
他の読み取りコマンドと同じく、`prx prompt TASK_ID` は [github-sync.md](github-sync.md) の共有 GitHub refresh 間隔を確認する。
組み込みテンプレートは、PRX がローカルのツールであってリポジトリの読み手には見えないことを伝え、コードコメント・コミットメッセージ・pull request で PRX やその識別子・コマンドに言及しないよう指示する。
組み込みテンプレートは 3 種類とも無人実行を前提とし、質問を禁じて未確定事項を仮定として明記させる。プロンプトは人の見ていない実行にも batch の SubAgent にも同じ文面のまま渡るためである。

## Task prompt

task のテンプレートを選ぶのは implementation plan の有無だけで、表示状態や readiness は関与しない。

| Implementation plan | テンプレート | ステータスの設定時点 | 最後の動作 |
|---|---|---|---|
| Absent | Design | 何よりも先に `designing` を設定する | plan を登録し、保存済みステータスは変更しない |
| Present | Implementation | 変更前に `in_progress` を設定する | 結果を記録する |

作業前にステータスを設定することで進行中の作業が可視化され、plan を登録すれば designing の task は designed として提示される。
組み込みの implementation テンプレートと batch テンプレートは、作業のベースから分岐するようエージェントに指示する。stacked pull request のために、未解決の blocker のブランチも対象に含む。

組み込みの design テンプレートは feature graph を、兄弟 task の scope を自分の設計から切り分けるために読ませる。graph にまだない依存を見つけた場合は `prx dependency add` で登録させ、計画の中だけに書き残させない。
計画の登録は `prx plan set TASK_ID --stdin` を主な経路とし、ファイル経由で渡す場合は対象リポジトリの外に置いて登録後に削除するよう指示する。設計の成果物が対象リポジトリに混入し、次の実装ステップでコミットされるのを避けるためである。
計画の本文は、対象リポジトリ自身の文書とエージェント向け指示が使っている言語で書かせる。組み込みテンプレートは対象リポジトリを選ばないので、言語を固定せずリポジトリ側の慣習に従わせる。

組み込みの design テンプレートと implementation テンプレートは、設計や実装を始める前に task・feature・project の document を読むようエージェントに案内する。
プロンプトは document の本文を含めず、`prx document` と `prx document get DOCUMENT_ID` へ誘導する。plan と同じく本文が大きくなり得るためである。
project の識別子は置換語彙にないので、`prx feature` で所属 project を辿るよう案内する。

## Batch prompt

WebUI は、feature と選択した task に対して batch テンプレートを明示的に要求する。
組み込みテンプレートは各 task を個別の SubAgent に委譲し、その SubAgent は batch 本文に複製された task の文面ではなく `prx prompt TASK_ID` から指示を得る。

選択したすべての task が指定した feature に属し、未解決の blocker がすべて blocked な task と一緒に含まれていなければ、レンダリングは失敗する。
受け取ったエージェントは blocker を先に実装し、依存する pull request をその上に積む。
WebUI は既定で designed かつ ready な task を提示し、要求があれば blocked な task も提示する。選択はユーザーに委ねる。
blocker を一緒に含められない task と、未解決の blocker が複数ある task は除外する。pull request のベースは 1 つしか取れないためである。
そうした task は、blocker が着地してから個別に引き渡す。

## テンプレートの設定とレンダリング

テンプレートは共有設定であり、CLI と WebUI が同じ文面を出力するようにする。
一部だけ更新された状態を避けるため、テンプレートはすべてまとめて書き込む。
省略されたテンプレートや空のテンプレートは組み込みの既定値に戻すので、古い設定ファイルも読み込めるままになる。
組み込みの既定値と一致するテンプレートはファイルから省く。カスタマイズしていない環境が、アップグレード後の文面更新に追従できるようにするためである。

テンプレートは単純な置換を使い、task prompt と batch prompt それぞれについて `internal/prompt/prompt.go` で定義された、独立した閉じた語彙の上で動く。
未知のプレースホルダは拒否する。task テンプレートには `{{task_id}}`、batch テンプレートには `{{task_list}}` を必須とし、対象が必ず特定されるようにする。
plan の本文は含めない。1 MiB に達することもあれば locator の先にあることもあるため、prompt はエージェントを `prx plan TASK_ID` に誘導する。
scope が無い場合は `(not specified)` としてレンダリングし、読み込みに失敗した値と区別する。
サーバは保存済みテンプレートと合わせて語彙と組み込み既定値も提供するので、エディタはそのサーバが受け付ける内容で検証・復元できる。

WebUI は、その場でレンダリングした prompt をコピーする。snapshot 取得後に plan が登録・削除されても誤ったテンプレートが選ばれないようにするためである。
`prx prompt TASK_ID` は prompt 本文だけを、そのまま使える形で出力する。
diagnostics は各テンプレートの長さと、組み込み既定値と一致するかどうかを報告する。ユーザーが書いた文面を露出せずにカスタマイズの有無を示す。

### project・feature の上書き

テンプレートの解決順は global、所属 project、feature の 3 層で、各層の design・implementation・batch は独立している。下位層の値が空、または DB の NULL なら上位層を継承し、非空の値だけをその種類について置き換える。
feature を別の project へ移しても feature 自身の上書きは残り、継承している種類だけ新しい project の値に追従する。

project と feature の応答には `prompt_overrides` オブジェクトを常に含め、空の各キーはその層で上書きしていないことを表す。人間向けの詳細表示は本文を露出せず、設定されている種類だけを示す。

CLI では次のコマンドで本文をファイルまたは標準入力から設定・解除できる。`KIND` は `design`、`implementation`、`batch` のいずれかで、`set` は空本文と不正な placeholder を拒否する。

```
prx project prompt set PROJECT_ID KIND --file PATH|--stdin
prx project prompt unset PROJECT_ID KIND
prx feature prompt set FEATURE_ID KIND --file PATH|--stdin
prx feature prompt unset FEATURE_ID KIND
```

アーカイブ済みの project・feature は本文を読めるが、ライフサイクル解除以外の書き込みを拒否する。上書き本文の検証に失敗した場合は `INVALID_PROMPT_TEMPLATE` を返し、エラーには scope と種類を含める。
