package prompt

// 日本語版の組み込みテンプレート。英語版と同じ条項を同じ順序で並べ、置換語彙・
// `prx` のコマンド例・ステータス値は原語のまま残す。
// docs/design/agent-prompts.md を参照。

const japaneseDesignTemplate = `feature {{feature_id}} の PRX task {{task_id}} を設計する。

Title: {{task_title}}
Scope: {{task_scope}}

PRX は task と task 同士の依存関係を追跡するローカルの CLI である。
正確なコマンド体系は ` + "`prx --help`" + ` と ` + "`prx <command> --help`" + ` で確認する。
PRX はこのマシンの中だけで動くので、リポジトリを読む人からは見えない。
リポジトリが運ぶものに持ち込まないこと。コードコメント・コミットメッセージ・pull request の
いずれも PRX やその識別子・コマンドに言及してはならない。
この実行を見ている人はいないので、質問はしない。資料が決めていない点があれば、
根拠を示せる選択肢を採り、仮定として明記して計画に書く。

1. 何よりも先に、この task を設計中として記録する。
   - ` + "`prx task update {{task_id}} --status designing`" + `
2. task と、それが依存している作業を読む。
   - ` + "`prx task {{task_id}}`" + `
   - ` + "`prx graph {{feature_id}}`" + `
   その graph にある他の task にはそれぞれの scope がある。結果に依存してよいが、ここで設計しない。
   graph にまだない依存は ` + "`prx dependency add BLOCKER_TASK_ID {{task_id}}`" + ` で登録する。
3. 何かを決める前に、PRX に添付された資料を読む。
   document は task・その feature・その feature が属する project のそれぞれに付き、
   どれにもこの scope が満たすべき要件が入っている可能性がある。
   - ` + "`prx document --task {{task_id}}`" + ` と ` + "`prx document --feature {{feature_id}}`" + `
   - ` + "`prx feature {{feature_id}}`" + ` が project を示すので、続けて ` + "`prx document --project PROJECT_ID`" + `
   - ` + "`prx document get DOCUMENT_ID`" + ` は保存済みの document を出力する。URL やローカル
     ファイルを指す document は locator を出力するだけなので、自分で開く。
4. リポジトリを調査し、上記の scope をどう実装するべきか決める。
5. できあがった計画を task に登録する。
   計画はリポジトリの外の一時ディレクトリにファイルとして書き、そのファイルを登録し、
   登録後に削除する。リポジトリの中に残した計画は次のステップでコミットされてしまう。
   - ` + "`prx plan set {{task_id}} --file PATH`" + `
   計画は、対象リポジトリ自身の文書とエージェント向け指示が使っている言語で書く。
   計画の登録がこの task を設計済みとして提示するので、その後ステータスは変更しない。

設計だけを行う。実装と pull request は次のステップに任せる。
`

const japaneseImplementationTemplate = `feature {{feature_id}} の PRX task {{task_id}} を実装する。

Title: {{task_title}}
Scope: {{task_scope}}

PRX は task と task 同士の依存関係を追跡するローカルの CLI である。
正確なコマンド体系は ` + "`prx --help`" + ` と ` + "`prx <command> --help`" + ` で確認する。
PRX はこのマシンの中だけで動くので、リポジトリを読む人からは見えない。
リポジトリが運ぶものに持ち込まないこと。コードコメント・コミットメッセージ・pull request の
いずれも PRX やその識別子・コマンドに言及してはならない。
この実行を見ている人はいないので、質問はしない。計画が決めていない点があれば、
根拠を示せる選択肢を採り、仮定として明記して報告する。

1. task と、それが依存している作業と、登録済みの計画を読む。
   - ` + "`prx task {{task_id}}`" + `
   - ` + "`prx graph {{feature_id}}`" + `
   - ` + "`prx plan {{task_id}}`" + `
2. コードを書く前に、PRX に添付された資料を読む。
   document は task・その feature・その feature が属する project のそれぞれに付き、
   どれにもこの scope が満たすべき要件が入っている可能性がある。
   - ` + "`prx document --task {{task_id}}`" + ` と ` + "`prx document --feature {{feature_id}}`" + `
   - ` + "`prx feature {{feature_id}}`" + ` が project を示すので、続けて ` + "`prx document --project PROJECT_ID`" + `
   - ` + "`prx document get DOCUMENT_ID`" + ` は保存済みの document を出力する。URL やローカル
     ファイルを指す document は locator を出力するだけなので、自分で開く。
3. 何かを変更する前に、この task を作業中として記録する。
   - ` + "`prx task update {{task_id}} --status in_progress`" + `
4. 上記の scope の内側にとどまって計画を実装する。
   既定で main や master から分岐せず、その作業が本来乗るべきベースから分岐する。
   blocker が未解決の task はその blocker のブランチに乗せ、2 つの pull request を積む。
5. 結果を PRX に記録する。
   - pull request として着地する作業: 分岐したベースに対して pull request を開き、
     ` + "`prx pr attach {{task_id}} PULL_REQUEST_URL`" + ` を実行する。
     その後の状態は pull request に従うので、ステータスを手で設定しない。
   - pull request を伴わない作業: 終わったら ` + "`prx task update {{task_id}} --status completed`" + ` を実行する。

変更した内容と、計画が扱っていなかった点を報告する。
`

const japaneseBatchTemplate = `feature {{feature_id}} の、以下に挙げる PRX task を実装する。

PRX は task と task 同士の依存関係を追跡するローカルの CLI である。
正確なコマンド体系は ` + "`prx --help`" + ` と ` + "`prx <command> --help`" + ` で確認する。
PRX はこのマシンの中だけで動くので、リポジトリを読む人からは見えない。
自分の作業でも各 SubAgent の作業でも、リポジトリが運ぶものに持ち込まないこと。コードコメント・
コミットメッセージ・pull request のいずれも PRX やその識別子・コマンドに言及してはならない。
この実行を見ている人はいないので、自分も各 SubAgent も質問はしない。資料が決めていない点が
あれば、根拠を示せる選択肢を採り、仮定として明記して報告する。

Tasks:
{{task_list}}

1. feature の graph を読み、挙がっている task が残りの作業とどう関係するかを把握する。
   - ` + "`prx graph {{feature_id}}`" + `
2. 各 task をそれぞれ専用の SubAgent に渡す。1 つの SubAgent に 1 つの task だけを渡し、
   同じ SubAgent に 2 つの task を渡さない。
   各 SubAgent は指示を自分からではなく PRX から受け取る。
   - 渡された task について ` + "`prx prompt TASK_ID`" + ` を実行し、表示されたプロンプトに従う。
   - 変更した内容と、プロンプトが扱っていなかった点を報告する。
   graph 上で独立している task は並行して進めてよい。
   この一覧の別の task に依存する task は、その task が終わってから実装し、
   その pull request は依存先の task の pull request に積む。
   既定で main や master から分岐せず、その作業が本来乗るべきベースから分岐する。
3. すべての SubAgent を待ち、それぞれの報告を読む。
   SubAgent が失敗した task は未完のままにする。自分で実装せずに報告する。
   それに依存する task もベースがないので、着手しないままにする。

各 task の結果を、失敗したものも含めて個別に報告する。
`
