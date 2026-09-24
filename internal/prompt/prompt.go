// Package prompt は CLI・RPC サーバー・WebUI が共有するエージェント用プロンプト
// テンプレートを管理する。組み込みテンプレート、その検証、プレースホルダの置換、
// タスクごとにテンプレートを選ぶ規則を含む。
package prompt

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/HappyOnigiri/nnx/internal/domain"
)

// MaximumTemplateBytes は保存されるテンプレート 1 件の上限。テンプレートは YAML・
// JSON・Protocol Buffers を経由し、他のエージェントの入力に貼るためのものなので、
// 本文が無制限でも誰の得にもならない。
const MaximumTemplateBytes = 8192

// Kind はテンプレートの種類を示す。どの種類を使うかは呼び出し元が選び、
// 選ばなかった場合だけ task から導出する。いずれも保存されることはない。
type Kind string

const (
	// KindDesign はエージェントに実装計画の作成を依頼する。
	KindDesign Kind = "design"
	// KindImplementation はエージェントに登録済みの計画の実行を依頼する。
	KindImplementation Kind = "implementation"
	// KindBatch は複数の task をまとめて実装するようエージェントに依頼する。
	KindBatch Kind = "batch"
	// KindBatchDesign は複数の task をまとめて設計するようエージェントに依頼する。
	KindBatchDesign Kind = "batch_design"
)

// Templates は task 用テンプレートと、複数 task をまとめる batch テンプレートを保持する。
// batch 系は個々の task から導出されず、複数 task をまとめて要求する呼び出し元が選ぶ。
type Templates struct {
	Design         string `yaml:"design"         json:"design"`
	Implementation string `yaml:"implementation" json:"implementation"`
	Batch          string `yaml:"batch"          json:"batch"`
	BatchDesign    string `yaml:"batch_design"   json:"batch_design"`
}

// IsBatch は複数 task をまとめる種類かどうかを返す。batch 系は task 用とは別の
// 置換語彙を持つため、検証と描画がこの区別を使う。
func (k Kind) IsBatch() bool { return k == KindBatch || k == KindBatchDesign }

// placeholderPattern は未対応のものも含めて置換トークン全てに一致する。レンダラが
// 黙って残してしまう名前を検証で弾けるようにするため。
var placeholderPattern = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// requiredPlaceholder はテンプレートを 1 つのタスクに向け続ける。これがないと
// 生成されたプロンプトは対象を示さず、受け取ったエージェントが着手できない。
const requiredPlaceholder = "task_id"

// supportedPlaceholders は置換語彙の全体。計画の本文は意図的に含めない。計画は
// 最大 1 MiB になり得るしロケータの先にある場合もあるため、プロンプトでは
// 代わりに `nnx plan TASK_ID` で読むようエージェントに指示する。
var supportedPlaceholders = []string{
	requiredPlaceholder,
	"feature_id",
	"task_title",
	"task_scope",
}

// batchRequiredPlaceholder は batch テンプレートを要求されたタスク群に向け続ける。
// これがないと生成されたプロンプトは対象を一切示さず、タスク用プロンプトが
// 識別子を欠くよりさらに悪い。
const batchRequiredPlaceholder = "task_list"

// batchSupportedPlaceholders は batch の語彙で、個々のタスクのタイトルやスコープは
// 含まない。
// docs/design/agent-prompts.md を参照。
var batchSupportedPlaceholders = []string{
	batchRequiredPlaceholder,
	"feature_id",
}

const defaultDesignTemplate = `Design Next Now X task {{task_id}} of feature {{feature_id}}.

Title: {{task_title}}
Scope: {{task_scope}}

Next Now X is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`nnx --help`" + ` and ` + "`nnx <command> --help`" + ` for its exact surface.
Next Now X runs on this machine only, so nobody reading the repository can see it.
Keep it out of what the repository carries: no code comment, commit message, or pull request
may mention Next Now X, its identifiers, or its commands.
Resolve what you can on your own: investigate whatever the repository, its conventions, or the
attached material can answer, and ask the user only about the decisions that would change the shape
of the design and that no investigation settles. Wait for that answer before you build the plan
around it. Leave local implementation choices to the implementation step, and write everything else
into the plan as a stated assumption.

1. Mark the task as being designed before anything else.
   - ` + "`nnx task update {{task_id}} --status designing`" + `
2. Read the task and the work it depends on.
   - ` + "`nnx task {{task_id}}`" + `
   - ` + "`nnx graph {{feature_id}}`" + `
   The other tasks in that graph carry their own scope: rely on their result, do not design it here.
   A dependency the graph does not show yet goes in as ` + "`nnx dependency add BLOCKER_TASK_ID {{task_id}}`" + `.
3. Read the reference material attached in Next Now X before you decide anything.
   Documents hang off the task, off its feature, and off the project that feature belongs to,
   and any of them may carry the requirements this scope has to meet.
   - ` + "`nnx document --task {{task_id}}`" + ` and ` + "`nnx document --feature {{feature_id}}`" + `
   - ` + "`nnx feature {{feature_id}}`" + ` names the project, then ` + "`nnx document --project PROJECT_ID`" + `
   - ` + "`nnx document get DOCUMENT_ID`" + ` prints a stored document; one that points at a URL or a
     local file prints that locator instead, so open it yourself.
4. Investigate the repository and decide how the scope above should be built.
5. Register the resulting plan on the task.
   Write the plan to a file in a temporary directory outside the repository, register that file,
   and delete it afterwards. A plan left inside the repository ends up committed by the next step.
   - ` + "`nnx plan set {{task_id}} --file PATH`" + `
   Write the plan in the language the repository's own documents and agent instructions use.
   Registering the plan is what presents the task as designed, so leave the status alone afterwards.

Design only: leave the implementation and the pull request to the next step.
`

const defaultImplementationTemplate = `Implement Next Now X task {{task_id}} of feature {{feature_id}}.

Title: {{task_title}}
Scope: {{task_scope}}

Next Now X is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`nnx --help`" + ` and ` + "`nnx <command> --help`" + ` for its exact surface.
Next Now X runs on this machine only, so nobody reading the repository can see it.
Keep it out of what the repository carries: no code comment, commit message, or pull request
may mention Next Now X, its identifiers, or its commands.
Nobody is watching this run, so do not ask questions. Where the plan leaves something undecided,
take the option you can defend and report it as a stated assumption.

1. Read the task, the work it depends on, and its registered plan.
   - ` + "`nnx task {{task_id}}`" + `
   - ` + "`nnx graph {{feature_id}}`" + `
   - ` + "`nnx plan {{task_id}}`" + `
   That last command fails when no plan was registered. That is not an error to fix: settle the
   approach yourself from the title, the scope, and the material below, and report it as a stated
   assumption. Do not register a plan.
2. Read the reference material attached in Next Now X before you write any code.
   Documents hang off the task, off its feature, and off the project that feature belongs to,
   and any of them may carry the requirements this scope has to meet.
   - ` + "`nnx document --task {{task_id}}`" + ` and ` + "`nnx document --feature {{feature_id}}`" + `
   - ` + "`nnx feature {{feature_id}}`" + ` names the project, then ` + "`nnx document --project PROJECT_ID`" + `
   - ` + "`nnx document get DOCUMENT_ID`" + ` prints a stored document; one that points at a URL or a
     local file prints that locator instead, so open it yourself.
3. Mark the task as being worked on before you change anything.
   - ` + "`nnx task update {{task_id}} --status in_progress`" + `
4. Implement the plan, or the approach you settled on, staying inside the scope above.
   Branch from the base the work actually belongs on rather than from main or master by default.
   A task whose blocker is still open belongs on that blocker's branch, so the two pull requests stack.
5. Record the result in Next Now X.
   - Work that lands as a pull request: open it against the base you branched from,
     then run ` + "`nnx pr attach {{task_id}} PULL_REQUEST_URL`" + `.
     Its state then follows the pull request, so do not set the status by hand.
   - Work without a pull request: run ` + "`nnx task update {{task_id}} --status completed`" + ` once it is done.

Report what you changed and anything the plan did not cover.
`

const defaultBatchTemplate = `Implement the Next Now X tasks of feature {{feature_id}} listed below.

Next Now X is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`nnx --help`" + ` and ` + "`nnx <command> --help`" + ` for its exact surface.
Next Now X runs on this machine only, so nobody reading the repository can see it.
Keep it out of what the repository carries, in your work and in every SubAgent's: no code comment,
commit message, or pull request may mention Next Now X, its identifiers, or its commands.
Nobody is watching this run, so neither you nor any SubAgent asks questions. Where the material
leaves something undecided, take the option you can defend and report it as a stated assumption.

Tasks:
{{task_list}}

1. Read the feature graph so you know how the listed tasks relate to the rest of the work.
   - ` + "`nnx graph {{feature_id}}`" + `
2. Hand every task to its own SubAgent: one task per SubAgent, and never two tasks to the same one.
   Each SubAgent works in a fresh git worktree of its own: SubAgents sharing a checkout
   commit each other's half-finished edits.
   Each SubAgent takes its instructions from Next Now X rather than from you.
   - It runs ` + "`nnx prompt TASK_ID --kind implementation`" + ` for the task it was given and follows
     the prompt that prints. The flag matters: without it a task that has no registered plan is
     handed the design prompt and the task is designed instead of implemented.
   - It reports what it changed and anything the prompt did not cover.
   Tasks the graph shows as independent may run in parallel.
   A task that depends on another task of this list is implemented after that task is finished,
   and its pull request is stacked on the pull request of the task it depends on.
   Branch from the base the work belongs on rather than from main or master by default.
3. Wait for every SubAgent and read what each one reported.
   A task whose SubAgent failed stays unfinished: report it instead of implementing it yourself.
   Whatever depends on it stays unstarted as well, because its base is not there.

Report each task's outcome separately, including the ones that failed.
`

const defaultBatchDesignTemplate = `Design the Next Now X tasks of feature {{feature_id}} listed below.

Next Now X is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`nnx --help`" + ` and ` + "`nnx <command> --help`" + ` for its exact surface.
Next Now X runs on this machine only, so nobody reading the repository can see it.
Keep it out of what the repository carries, in your work and in every SubAgent's: no code comment,
commit message, or pull request may mention Next Now X, its identifiers, or its commands.
Nobody is watching this run, so neither you nor any SubAgent asks questions. Where the material
leaves something undecided, take the option you can defend and report it as a stated assumption.

Tasks:
{{task_list}}

1. Read the feature graph so you know how the listed tasks relate to the rest of the work.
   - ` + "`nnx graph {{feature_id}}`" + `
2. Hand every task to its own SubAgent: one task per SubAgent, and never two tasks to the same one.
   Each SubAgent works in a fresh git worktree of its own: SubAgents sharing a checkout
   commit each other's half-finished edits.
   Each SubAgent takes its instructions from Next Now X rather than from you.
   - It runs ` + "`nnx prompt TASK_ID --kind design`" + ` for the task it was given and follows the
     prompt that prints.
   - That prompt tells it to ask the user about decisions that shape the design. Nobody is here to
     answer, so it writes those into the plan as stated assumptions instead.
   - It ends by registering the plan with ` + "`nnx plan set TASK_ID --file PATH`" + `.
     It writes no production code and opens no pull request.
   - It reports what it decided and anything the prompt did not cover.
   Tasks the graph shows as independent may run in parallel.
   A task that depends on another task of this list is designed after that task's plan is
   registered, so its own design can rely on what that plan settles.
3. Wait for every SubAgent and read what each one reported.
   A task whose SubAgent failed stays undesigned: report it instead of designing it yourself.
   Whatever depends on it stays undesigned as well, because the plan it would build on is missing.

Design only: leave the implementation and the pull requests to a later step.
Report each task's outcome separately, including the ones that failed.
`

// SupportedPlaceholders は置換語彙を波括弧なしで返す。クライアントが独自の一覧を
// 抱えて黙って乖離するのではなく、サーバーが受け付ける内容を提示できるようにある。
func SupportedPlaceholders() []string {
	return slices.Clone(supportedPlaceholders)
}

// RequiredPlaceholder はすべてのタスクテンプレートが使うべきプレースホルダを返す。
func RequiredPlaceholder() string { return requiredPlaceholder }

// BatchSupportedPlaceholders は batch の置換語彙を返す。batch テンプレートがタスク用
// プレースホルダを使っても展開元がないため、別建てで提供している。
func BatchSupportedPlaceholders() []string {
	return slices.Clone(batchSupportedPlaceholders)
}

// BatchRequiredPlaceholder はすべての batch テンプレートが使うべきプレースホルダを返す。
func BatchRequiredPlaceholder() string { return batchRequiredPlaceholder }

// DefaultTemplates は設定が独自のテンプレートを定義していないときに使う
// 組み込みテンプレートを、実効言語の版で返す。未知の言語は英語版を返す。
func DefaultTemplates(language Language) Templates {
	if language == LanguageJapanese {
		return Templates{
			Design:         japaneseDesignTemplate,
			Implementation: japaneseImplementationTemplate,
			Batch:          japaneseBatchTemplate,
			BatchDesign:    japaneseBatchDesignTemplate,
		}
	}
	return Templates{
		Design:         defaultDesignTemplate,
		Implementation: defaultImplementationTemplate,
		Batch:          defaultBatchTemplate,
		BatchDesign:    defaultBatchDesignTemplate,
	}
}

// KindFor は種類を選ばなかった呼び出し元のためにタスクから既定の種類を導出する。
// 判断材料は実装計画の有無だけ。表示状態や着手可否は進捗を表すものであって、
// エージェントに投げる問いがどれかを決めるものではない。
func KindFor(task domain.Task) Kind {
	if task.HasImplementationPlan {
		return KindImplementation
	}
	return KindDesign
}

// Template は指定した kind に対応する保存済みテンプレートを返す。
func (t Templates) Template(kind Kind) string {
	switch kind {
	case KindImplementation:
		return t.Implementation
	case KindBatch:
		return t.Batch
	case KindBatchDesign:
		return t.BatchDesign
	case KindDesign:
		return t.Design
	}
	return t.Design
}

// Resolve は global、project、feature の順にテンプレートを種類ごとに合成する。
// 下位スコープの空文字列は上位スコープから継承する。global は必ず Normalize
// を通すが、DB から読んだ上書きはここで個別に検証する。
func Resolve(
	language Language,
	global Templates,
	project, feature domain.PromptTemplateOverrides,
) (Templates, error) {
	base, err := global.Normalize(language)
	if err != nil {
		return Templates{}, err
	}
	result := base
	for _, candidate := range []struct {
		name      string
		overrides domain.PromptTemplateOverrides
	}{
		{name: "project", overrides: project},
		{name: "feature", overrides: feature},
	} {
		if err := applyOverride(&result, candidate.name, candidate.overrides); err != nil {
			return Templates{}, err
		}
	}
	return result, nil
}

// ValidateOverride は project / feature に保存する上書き 1 件を検証する。
// 空文字列は継承を意味するため受理し、それ以外は global と同じ語彙・上限を
// 必須 placeholder とともに適用する。
func ValidateOverride(kind Kind, value string) error {
	if value == "" {
		return nil
	}
	supported, required := supportedPlaceholders, requiredPlaceholder
	if kind.IsBatch() {
		supported, required = batchSupportedPlaceholders, batchRequiredPlaceholder
	}
	if kind != KindDesign && kind != KindImplementation && !kind.IsBatch() {
		return newError("prompt_overrides", "unsupported template kind %q", kind)
	}
	return validateTemplate("prompt_overrides."+string(kind), value, supported, required)
}

func applyOverride(result *Templates, scope string, overrides domain.PromptTemplateOverrides) error {
	for _, item := range []struct {
		kind   Kind
		value  string
		target *string
	}{
		{kind: KindDesign, value: overrides.Design, target: &result.Design},
		{kind: KindImplementation, value: overrides.Implementation, target: &result.Implementation},
		{kind: KindBatch, value: overrides.Batch, target: &result.Batch},
		{kind: KindBatchDesign, value: overrides.BatchDesign, target: &result.BatchDesign},
	} {
		if item.value == "" {
			continue
		}
		if err := ValidateOverride(item.kind, item.value); err != nil {
			var typed *Error
			if errors.As(err, &typed) {
				return newError(scope+"."+typed.Field, "%s", typed.Message)
			}
			return err
		}
		*item.target = item.value
	}
	return nil
}

// Normalize は省略されたテンプレートを実効言語の既定値で埋め、最初に見つかった
// 不正なテンプレートを報告する。prompts がなかった頃の設定ファイルがそのまま
// 読み込めるのはこの処理のおかげ。
func (t Templates) Normalize(language Language) (Templates, error) {
	result := t
	defaults := DefaultTemplates(language)
	if strings.TrimSpace(result.Design) == "" {
		result.Design = defaults.Design
	}
	if strings.TrimSpace(result.Implementation) == "" {
		result.Implementation = defaults.Implementation
	}
	if strings.TrimSpace(result.Batch) == "" {
		result.Batch = defaults.Batch
	}
	if strings.TrimSpace(result.BatchDesign) == "" {
		result.BatchDesign = defaults.BatchDesign
	}
	if err := validateTemplate(
		"prompts.design", result.Design, supportedPlaceholders, requiredPlaceholder,
	); err != nil {
		return Templates{}, err
	}
	if err := validateTemplate(
		"prompts.implementation", result.Implementation, supportedPlaceholders, requiredPlaceholder,
	); err != nil {
		return Templates{}, err
	}
	if err := validateTemplate(
		"prompts.batch", result.Batch, batchSupportedPlaceholders, batchRequiredPlaceholder,
	); err != nil {
		return Templates{}, err
	}
	if err := validateTemplate(
		"prompts.batch_design", result.BatchDesign, batchSupportedPlaceholders, batchRequiredPlaceholder,
	); err != nil {
		return Templates{}, err
	}
	return result, nil
}

// Render は 1 つのタスクを指定された種類のプロンプトに展開する。種類が空なら
// KindFor で導出する。読み込みから描画までの間に手編集されたファイルが未展開の
// プレースホルダを出さないよう、保存済みテンプレートをここで再検証する。
func Render(task domain.Task, kind Kind, templates Templates, language Language) (Kind, string, error) {
	normalized, err := templates.Normalize(language)
	if err != nil {
		return "", "", err
	}
	if kind != KindDesign && kind != KindImplementation {
		kind = KindFor(task)
	}
	values := map[string]string{
		"task_id":    task.ID,
		"feature_id": task.FeatureID,
		"task_title": task.Title,
		"task_scope": describedScope(task.Scope),
	}
	body := placeholderPattern.ReplaceAllStringFunc(normalized.Template(kind), func(match string) string {
		return values[placeholderName(match)]
	})
	return kind, body, nil
}

// RenderBatch は指定された batch 種別のテンプレートを複数タスクに展開する。
// 呼び出し元が並べた順を保ち、種類が batch 系でなければ実装用の batch に落とす。
// docs/design/agent-prompts.md を参照。
func RenderBatch(
	featureID string,
	tasks []domain.Task,
	kind Kind,
	templates Templates,
	language Language,
) (Kind, string, error) {
	normalized, err := templates.Normalize(language)
	if err != nil {
		return "", "", err
	}
	if !kind.IsBatch() {
		kind = KindBatch
	}
	values := map[string]string{
		"feature_id": featureID,
		"task_list":  batchTaskList(tasks),
	}
	body := placeholderPattern.ReplaceAllStringFunc(
		normalized.Template(kind),
		func(match string) string { return values[placeholderName(match)] },
	)
	return kind, body, nil
}

// batchTaskList は各タスクを、エージェントが `nnx prompt` に渡し返す識別子で列挙する。
// プロンプトを読む人がタスクを見分けられるよう、後ろにタイトルを添える。
func batchTaskList(tasks []domain.Task) string {
	lines := make([]string, 0, len(tasks))
	for _, task := range tasks {
		lines = append(lines, fmt.Sprintf("- %s: %s", task.ID, task.Title))
	}
	return strings.Join(lines, "\n")
}

// unspecifiedScope はスコープなしで作られたタスクの代わりに置く値。テンプレートは
// 「上記のスコープ」を参照させるが、Next Now X を知らないエージェントは空行と読み込みに
// 失敗した値を区別できない。
const unspecifiedScope = "(not specified)"

func describedScope(scope string) string {
	if strings.TrimSpace(scope) == "" {
		return unspecifiedScope
	}
	return scope
}

// Error は拒否されたテンプレートを報告する。呼び出し元が修正すべきテンプレートを
// 指し示せるよう、フィールド名を保持する。
type Error struct {
	Field   string
	Message string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Message) }

func newError(field, format string, args ...any) *Error {
	return &Error{Field: field, Message: fmt.Sprintf(format, args...)}
}

func validateTemplate(field, value string, supported []string, required string) error {
	if len(value) > MaximumTemplateBytes {
		return newError(field, "template must be at most %d bytes", MaximumTemplateBytes)
	}
	found := make(map[string]struct{})
	for _, match := range placeholderPattern.FindAllString(value, -1) {
		name := placeholderName(match)
		if !slices.Contains(supported, name) {
			return newError(field, "template uses unsupported placeholder %s", match)
		}
		found[name] = struct{}{}
	}
	if _, ok := found[required]; !ok {
		return newError(field, "template must use {{%s}}", required)
	}
	return nil
}

func placeholderName(match string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
}
