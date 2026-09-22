# エージェントプロンプト方針

PRX は PRX を知らないエージェントに task を渡すため、prompt は task の識別子と、それを操作するために必要なコマンドを含んでいなければならない。
task prompt や batch prompt のレンダリングは、task の状態・readiness・依存関係・implementation plan のいずれも変更しない。
他の読み取りコマンドと同じく、`prx prompt TASK_ID` は [github-sync.md](github-sync.md) の共有 GitHub refresh 間隔を確認する。
組み込みテンプレートは、PRX がローカルのツールであってリポジトリの読み手には見えないことを伝え、コードコメント・コミットメッセージ・pull request で PRX やその識別子・コマンドに言及しないよう指示する。
組み込みの design テンプレートは、未確定事項をまず自力で解消させる。リポジトリ・既存の規約・添付資料が答えを持つものは調査させ、調査では決められず設計の形を左右する判断だけをユーザーに質問させ、回答を待ってから計画に反映させる。設計の前提は後工程すべてを決めるため、そこだけは人の判断を仰ぐ価値がある。局所的な実装判断は実装のステップに委ねさせ、それ以外は仮定として明記させる。
組み込みの implementation テンプレートは計画のない task にも渡せるので、計画の取得が失敗したときは自分で方針を決めて進めさせ、計画は登録させない。実装のために設計をやり直させると、その計画がどこにも残らないまま実装だけが進むためである。
組み込みの implementation テンプレートと 2 つの batch テンプレートは無人実行を前提とし、質問を禁じて未確定事項を仮定として明記させる。これらは人の見ていない実行にも batch の SubAgent にも同じ文面のまま渡り、SubAgent は人と対話できないためである。
組み込みの batch_design テンプレートは、SubAgent が受け取る design テンプレートが質問を指示することを打ち消し、質問せず仮定として計画に書かせる。

## 言語

組み込みテンプレートは英語版と日本語版を持ち、実効言語で選ぶ。
実効言語は設定ファイルの `language` から決める。`auto`（キーがない場合を含む）のときはサーバが環境変数 `LC_ALL`・`LC_MESSAGES`・`LANG` をこの順で読み、`en` か `ja` を判定できなければ `en` にする。`en` または `ja` を明示した場合はロケールより優先する。
実効言語は `prx prompt` が描くプロンプトの言語であり、WebUI の表示言語でもある。WebUI は設定を取得した時点でサーバの実効言語へ切り替えるので、表示とプロンプトの言語がずれることはない。
project・feature の上書き本文は言語で切り替えない。ユーザーが書いた文字列であり、PRX が翻訳するものではないためである。
組み込みの既定値と一致するかの判定も実効言語の既定値に対して行う。そのため未カスタマイズの環境は、言語を切り替えてもファイルにテンプレートを書き出さず、その言語の文面の更新に追従し続ける。言語を切り替えた時点で既定値のままだったテンプレートは、新しい言語の既定値へ移る。
`prx config language` は設定した値と実効言語を表示し、`prx config language update auto|en|ja` が値を書き込む。
`prx setup` も `language` を書く。最初に `en` / `ja` を尋ね、選択を現在の実効言語と同じでも保存する（[daemon.md](daemon.md)）。

`prx daemon` を LaunchAgent から起動した場合、ロケール変数が渡らず `auto` の判定がターミナルからの `prx prompt` と食い違うことがある。WebUI の表示はサーバの実効言語に従うので画面とプロンプトはずれないが、ターミナルとデーモンの間ではずれる。`en` か `ja` を明示すれば解消する。
`prx setup` を端末から通した環境では `language` が `auto` でなくなるので、このずれは起きない。端末が無い環境で setup した場合、および setup の言語選択をキャンセルした場合は `auto` のまま残る。

## Task prompt

どのテンプレートで展開するかは要求側が指定する。指定がないときだけ implementation plan の有無から導出し、表示状態や readiness は関与しない。
`prx prompt TASK_ID --kind design|implementation` と WebUI のプロンプトダイアログのタブが、この指定を行う。task の状態に合わない種類も指定でき、WebUI はその間だけ注意を表示する。

導出の規則は次のとおりである。

| Implementation plan | テンプレート | ステータスの設定時点 | 最後の動作 |
|---|---|---|---|
| Absent | Design | 何よりも先に `designing` を設定する | plan を登録し、保存済みステータスは変更しない |
| Present | Implementation | 変更前に `in_progress` を設定する | 結果を記録する |

作業前にステータスを設定することで進行中の作業が可視化され、plan を登録すれば designing の task は designed として提示される。
組み込みの implementation テンプレートと batch テンプレートは、作業のベースから分岐するようエージェントに指示する。stacked pull request のために、未解決の blocker のブランチも対象に含む。

組み込みの design テンプレートは feature graph を、兄弟 task の scope を自分の設計から切り分けるために読ませる。graph にまだない依存を見つけた場合は `prx dependency add` で登録させ、計画の中だけに書き残させない。
計画の登録は `prx plan set TASK_ID --file PATH` だけを案内し、その本文は対象リポジトリの外の一時ディレクトリに書いて登録後に削除させる。設計の成果物が対象リポジトリに混入し、次の実装ステップでコミットされるのを避けるためである。
標準入力からの登録は案内しない。本文をシェルの heredoc に通すと、引用符なしのデリミタでは `$` や `` ` `` が展開され、計画に含めたコマンド例が書き換わったり実行されたりするためである。エージェントに正しい引用を任せるより、ファイル経由に一本化するほうが安全である。
計画の本文は、対象リポジトリ自身の文書とエージェント向け指示が使っている言語で書かせる。組み込みテンプレートは対象リポジトリを選ばないので、言語を固定せずリポジトリ側の慣習に従わせる。

組み込みの design テンプレートと implementation テンプレートは、設計や実装を始める前に task・feature・project の document を読むようエージェントに案内する。
プロンプトは document の本文を含めず、`prx document` と `prx document get DOCUMENT_ID` へ誘導する。plan と同じく本文が大きくなり得るためである。
project の識別子は置換語彙にないので、`prx feature` で所属 project を辿るよう案内する。

## Batch prompt

batch には実装用の `batch` と設計用の `batch_design` の 2 種類がある。WebUI は、feature と選択した task に対してどちらかを明示的に要求し、指定がない要求は `batch` を意味する。
どちらの組み込みテンプレートも各 task を個別の SubAgent に委譲し、その SubAgent は batch 本文に複製された task の文面ではなく `prx prompt TASK_ID` から指示を得る。
どちらの batch テンプレートも SubAgent に種類を明示させる。`batch` は `--kind implementation`、`batch_design` は `--kind design` を付けさせる。導出に任せると、計画のない task を一括実装へ渡したときに設計プロンプトが返り、実装のつもりで設計が回るためである。
`batch_design` の SubAgent は pull request を開かず、`prx plan set` で計画を登録して終える。
各 SubAgent はそれぞれ新規の worktree で作業する。テンプレートは独立した task の並行実装を許すので、checkout を共有した SubAgent が HEAD と index を取り合い、互いの書きかけの編集をコミットしてしまうためである。

選択したすべての task が指定した feature に属し、未解決の blocker がすべて blocked な task と一緒に含まれていなければ、レンダリングは失敗する。
受け取ったエージェントは blocker を先に処理し、実装では依存する pull request をその上に積む。
WebUI のモーダルは最上位のタブで設計と実装を選び、それが候補のベース集合と要求するテンプレートの両方を決める。タブを移ると選択とトグルはクリアする。
実装タブは designed な task を、設計タブは実装計画のない task、すなわち表示状態が not started か designing の task をベースにする。
各タブには、もう一方のベース集合を候補に加えるトグルがある。設計タブの「設計済みを表示」は designed な task を再設計の候補として加え、実装タブの「実装計画がないものを表示」は not started と designing の task を加える。
後者を有効にしている間、モーダルは実装プロンプトがまだ無い計画を読ませることを注意として示す。単一 task のダイアログが状態に合わない種類で出す注意と同じ扱いで、選べること自体は誤りではないので操作は止めない。
両方のタブに「依存未解決を表示」がある。
blocker を一緒に含められない task と、未解決の blocker が複数ある task は、どちらのタブでも除外する。pull request のベースは 1 つしか取れないためであり、設計でも後続の設計が blocker の計画を前提にするため順序は同じ規則に従う。
そうした task は、blocker が着地してから個別に引き渡す。
選択が確定するとモーダルはサーバーが描いた本文をプレビューし、コピーはそのプレビューと同じテキストを渡す。レンダリングが失敗している間はコピーできない。

## テンプレートの設定とレンダリング

テンプレートは共有設定であり、CLI と WebUI が同じ文面を出力するようにする。
一部だけ更新された状態を避けるため、テンプレートはすべてまとめて書き込む。
省略されたテンプレートや空のテンプレートは組み込みの既定値に戻すので、古い設定ファイルも読み込めるままになる。
組み込みの既定値と一致するテンプレートはファイルから省く。カスタマイズしていない環境が、アップグレード後の文面更新に追従できるようにするためである。
上書きしたテンプレートは更新に追従しないので、計画のない task の扱いのように後から加えた条項は上書きした種類には入らない。WebUI が種類の不一致に出す注意はこれを踏まえ、上書きしたテンプレートが計画のある前提のままかもしれないと伝える。

テンプレートは単純な置換を使い、task prompt と batch prompt それぞれについて `internal/prompt/prompt.go` で定義された、独立した閉じた語彙の上で動く。`batch` と `batch_design` は同じ batch の語彙を共有する。
未知のプレースホルダは拒否する。task テンプレートには `{{task_id}}`、batch テンプレートには `{{task_list}}` を必須とし、対象が必ず特定されるようにする。
plan の本文は含めない。1 MiB に達することもあれば locator の先にあることもあるため、prompt はエージェントを `prx plan TASK_ID` に誘導する。
scope が無い場合は `(not specified)` としてレンダリングし、読み込みに失敗した値と区別する。
サーバは保存済みテンプレートと合わせて語彙と組み込み既定値も提供するので、エディタはそのサーバが受け付ける内容で検証・復元できる。

WebUI は、その場でレンダリングした prompt をコピーする。snapshot 取得後に plan が登録・削除されても誤ったテンプレートが選ばれないようにするためである。
`prx prompt TASK_ID` は prompt 本文だけを、そのまま使える形で出力する。`--kind design|implementation` を付けると導出を上書きし、batch 系の種類は単一 task のプロンプトを持たないので受け付けない。
diagnostics は各テンプレートの長さと、組み込み既定値と一致するかどうかを報告する。ユーザーが書いた文面を露出せずにカスタマイズの有無を示す。

### project・feature の上書き

テンプレートの解決順は global、所属 project、feature の 3 層で、各層の design・implementation・batch・batch_design は独立している。下位層の値が空、または DB の NULL なら上位層を継承し、非空の値だけをその種類について置き換える。
feature を別の project へ移しても feature 自身の上書きは残り、継承している種類だけ新しい project の値に追従する。

project と feature の応答には `prompt_overrides` オブジェクトを常に含め、空の各キーはその層で上書きしていないことを表す。人間向けの詳細表示は本文を露出せず、設定されている種類だけを示す。

CLI では次のコマンドで本文をファイルまたは標準入力から設定・解除できる。`KIND` は `design`、`implementation`、`batch`、`batch_design` のいずれかで、`set` は空本文と不正な placeholder を拒否する。

```
prx project prompt set PROJECT_ID KIND --file PATH|--stdin
prx project prompt unset PROJECT_ID KIND
prx feature prompt set FEATURE_ID KIND --file PATH|--stdin
prx feature prompt unset FEATURE_ID KIND
```

アーカイブ済みの project・feature は本文を読めるが、ライフサイクル解除以外の書き込みを拒否する。上書き本文の検証に失敗した場合は `INVALID_PROMPT_TEMPLATE` を返し、エラーには scope と種類を含める。
