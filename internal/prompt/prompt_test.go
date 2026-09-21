package prompt_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

func designTask() domain.Task {
	return domain.Task{
		ID: "T-7", FeatureID: "F-3", Title: "Add the checkout API",
		Scope: "Server only",
	}
}

// 種類を指定しなかった呼び出しの既定は実装計画の有無だけで決まる。
func TestDefaultKindFollowsOnlyTheImplementationPlan(t *testing.T) {
	task := designTask()
	if got := prompt.KindFor(task); got != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", got, prompt.KindDesign)
	}
	// 計画のない完了タスクも依然として設計依頼になる。進捗はプロンプトが
	// 投げる問いの答えにはならない。
	task.Status = domain.TaskStatusCompleted
	if got := prompt.KindFor(task); got != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", got, prompt.KindDesign)
	}
	task.HasImplementationPlan = true
	if got := prompt.KindFor(task); got != prompt.KindImplementation {
		t.Fatalf("kind=%q, want %q", got, prompt.KindImplementation)
	}
}

func TestRenderExpandsEveryPlaceholderOfTheSelectedTemplate(t *testing.T) {
	templates := prompt.Templates{
		Design:         "design {{task_id}} {{feature_id}} {{task_title}} {{task_scope}}",
		Implementation: "implement {{task_id}}",
	}
	kind, body, err := prompt.Render(designTask(), "", templates, prompt.LanguageEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", kind, prompt.KindDesign)
	}
	if want := "design T-7 F-3 Add the checkout API Server only"; body != want {
		t.Fatalf("body=%q, want %q", body, want)
	}

	// タスクはスコープなしでも作れるが、テンプレートは「上記のスコープ」内に
	// 留まるよう指示する。そこが空行だと、制約がないのではなく値の読み込みに
	// 失敗したと読めてしまう。
	scopeless := designTask()
	scopeless.Scope = "  "
	_, body, err = prompt.Render(scopeless, "", templates, prompt.LanguageEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if want := "design T-7 F-3 Add the checkout API (not specified)"; body != want {
		t.Fatalf("body=%q, want %q", body, want)
	}

	planned := designTask()
	planned.HasImplementationPlan = true
	kind, body, err = prompt.Render(planned, "", templates, prompt.LanguageEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindImplementation || body != "implement T-7" {
		t.Fatalf("kind=%q body=%q", kind, body)
	}
}

func TestRenderFillsAnOmittedTemplateWithItsDefault(t *testing.T) {
	kind, body, err := prompt.Render(designTask(), "", prompt.Templates{}, prompt.LanguageEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", kind, prompt.KindDesign)
	}
	if !strings.Contains(body, "T-7") || !strings.Contains(body, "F-3") {
		t.Fatalf("default design prompt does not name the task: %q", body)
	}
	if strings.Contains(body, "{{") {
		t.Fatalf("default design prompt kept a placeholder: %q", body)
	}
}

// 既定テンプレートは保存されるテンプレートと同じ検証を通る。日本語は UTF-8 で
// 1 文字 3 バイトになるため、上限に収まることをここで確かめる。
func TestDefaultTemplatesSatisfyTheStoredTemplateRules(t *testing.T) {
	for _, language := range prompt.SupportedLanguages() {
		t.Run(string(language), func(t *testing.T) {
			defaults := prompt.DefaultTemplates(language)
			if _, err := defaults.Normalize(language); err != nil {
				t.Fatal(err)
			}
			for name, template := range map[string]string{
				"design":         defaults.Design,
				"implementation": defaults.Implementation,
				"batch":          defaults.Batch,
				"batch_design":   defaults.BatchDesign,
			} {
				if len(template) > prompt.MaximumTemplateBytes {
					t.Fatalf("default %s template is %d bytes", name, len(template))
				}
			}
		})
	}
}

func TestNormalizeRejectsTemplatesTheRendererCouldNotExpand(t *testing.T) {
	for name, test := range map[string]struct {
		templates prompt.Templates
		message   string
	}{
		"design without the task": {
			templates: prompt.Templates{Design: "no target", Implementation: "{{task_id}}"},
			message:   "prompts.design: template must use {{task_id}}",
		},
		"implementation with an unknown placeholder": {
			templates: prompt.Templates{Design: "{{task_id}}", Implementation: "{{task_id}} {{plan_body}}"},
			message:   "prompts.implementation: template uses unsupported placeholder {{plan_body}}",
		},
		"design above the size limit": {
			templates: prompt.Templates{
				Design:         "{{task_id}}" + strings.Repeat("x", prompt.MaximumTemplateBytes),
				Implementation: "{{task_id}}",
			},
			message: "prompts.design: template must be at most 8192 bytes",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := test.templates.Normalize(prompt.LanguageEnglish); err == nil || err.Error() != test.message {
				t.Fatalf("error=%v, want %q", err, test.message)
			}
			if _, _, err := prompt.Render(designTask(), "", test.templates, prompt.LanguageEnglish); err == nil {
				t.Fatal("Render accepted a template Normalize rejects")
			}
		})
	}
}

func TestNormalizeKeepsWhitespaceOtherThanAnEmptyTemplate(t *testing.T) {
	templates := prompt.Templates{Design: "  {{task_id}}\n\n", Implementation: "   "}
	normalized, err := templates.Normalize(prompt.LanguageEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Design != "  {{task_id}}\n\n" {
		t.Fatalf("design=%q, want the stored text unchanged", normalized.Design)
	}
	if normalized.Implementation != prompt.DefaultTemplates(prompt.LanguageEnglish).Implementation {
		t.Fatal("a blank implementation template was not replaced by its default")
	}
	if got := normalized.Template(prompt.KindDesign); got != normalized.Design {
		t.Fatalf("Template(design)=%q", got)
	}
}

func batchTasks() []domain.Task {
	return []domain.Task{
		{ID: "T-7", FeatureID: "F-3", Title: "Add the checkout API"},
		{ID: "T-9", FeatureID: "F-3", Title: "Bill the order"},
	}
}

func TestRenderBatchNamesEveryTaskInTheOrderItWasGiven(t *testing.T) {
	_, body, err := prompt.RenderBatch("F-3", batchTasks(), prompt.KindBatch, prompt.Templates{
		Batch: "batch {{feature_id}}\n{{task_list}}",
	}, prompt.LanguageEnglish)
	if err != nil {
		t.Fatal(err)
	}
	want := "batch F-3\n- T-7: Add the checkout API\n- T-9: Bill the order"
	if body != want {
		t.Fatalf("body=%q, want %q", body, want)
	}
}

// batch テンプレートはタスク用テンプレートと並べて保存されるため、一度も
// カスタマイズしていないインストールには組み込みの文言が届き続ける必要がある。
func TestRenderBatchFillsAnOmittedTemplateWithItsDefault(t *testing.T) {
	_, body, err := prompt.RenderBatch(
		"F-3", batchTasks(), prompt.KindBatch, prompt.Templates{}, prompt.LanguageEnglish,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"F-3", "T-7", "T-9", "SubAgent", "prx prompt TASK_ID"} {
		if !strings.Contains(body, want) {
			t.Fatalf("default batch prompt does not mention %q: %q", want, body)
		}
	}
	if strings.Contains(body, "{{") {
		t.Fatalf("default batch prompt kept a placeholder: %q", body)
	}
}

// batch テンプレートは独立したタスクの並行実装を許すので、SubAgent ごとに
// 作業ツリーを分ける指示が消えると、同じ checkout を取り合って互いの
// 書きかけの編集をコミットする。どちらの言語でも残っていなければならない。
func TestDefaultBatchTemplatesGiveEverySubAgentItsOwnWorktree(t *testing.T) {
	for _, language := range []prompt.Language{prompt.LanguageEnglish, prompt.LanguageJapanese} {
		_, body, err := prompt.RenderBatch("F-3", batchTasks(), prompt.KindBatch, prompt.Templates{}, language)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(body, "git worktree") {
			t.Fatalf("the %s batch prompt does not isolate each SubAgent: %q", language, body)
		}
	}
}

// batch の語彙はタスクの語彙とは別物。batch は複数タスクを対象にするため、
// タスク用プレースホルダを置いても展開元がない。
func TestNormalizeRejectsABatchTemplateOutsideItsOwnVocabulary(t *testing.T) {
	for name, test := range map[string]struct {
		batch   string
		message string
	}{
		"without the task list": {
			batch:   "batch of {{feature_id}}",
			message: "prompts.batch: template must use {{task_list}}",
		},
		"with a task placeholder": {
			batch:   "batch {{task_list}} {{task_title}}",
			message: "prompts.batch: template uses unsupported placeholder {{task_title}}",
		},
	} {
		t.Run(name, func(t *testing.T) {
			templates := prompt.Templates{Batch: test.batch}
			if _, err := templates.Normalize(prompt.LanguageEnglish); err == nil || err.Error() != test.message {
				t.Fatalf("error=%v, want %q", err, test.message)
			}
			if _, _, err := prompt.RenderBatch(
				"F-3", batchTasks(), prompt.KindBatch, templates, prompt.LanguageEnglish,
			); err == nil {
				t.Fatal("RenderBatch accepted a template Normalize rejects")
			}
		})
	}
	if slices.Contains(prompt.BatchSupportedPlaceholders(), "task_id") {
		t.Fatal("the batch vocabulary offers a placeholder no batch can expand")
	}
	if prompt.BatchRequiredPlaceholder() != "task_list" {
		t.Fatalf("batch required placeholder=%q", prompt.BatchRequiredPlaceholder())
	}
}

func TestResolveAppliesIndependentProjectAndFeatureOverrides(t *testing.T) {
	global := prompt.Templates{
		Design:         "global design {{task_id}}",
		Implementation: "global implementation {{task_id}}",
		Batch:          "global batch {{task_list}}",
	}
	resolved, err := prompt.Resolve(
		prompt.LanguageEnglish,
		global,
		domain.PromptTemplateOverrides{
			Design: "project design {{task_id}}",
		},
		domain.PromptTemplateOverrides{
			Implementation: "feature implementation {{task_id}}",
			Batch:          "feature batch {{task_list}}",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Design != "project design {{task_id}}" ||
		resolved.Implementation != "feature implementation {{task_id}}" ||
		resolved.Batch != "feature batch {{task_list}}" {
		t.Fatalf("resolved=%+v", resolved)
	}
}

func TestResolveRejectsAnInvalidOverrideWithItsScope(t *testing.T) {
	_, err := prompt.Resolve(
		prompt.LanguageEnglish,
		prompt.Templates{},
		domain.PromptTemplateOverrides{Design: "missing target"},
		domain.PromptTemplateOverrides{},
	)
	if err == nil || !strings.Contains(err.Error(), "project.prompt_overrides.design") {
		t.Fatalf("error=%v, want project scope", err)
	}
}

// 呼び出し元が種類を選べるので、計画のないタスクにも implementation を、計画の
// あるタスクにも design を描ける。これがないと WebUI のタブは何も選べない。
func TestRenderUsesTheRequestedKindInsteadOfTheDerivedOne(t *testing.T) {
	templates := prompt.Templates{
		Design:         "design {{task_id}}",
		Implementation: "implement {{task_id}}",
	}
	planned := designTask()
	planned.HasImplementationPlan = true
	for name, test := range map[string]struct {
		task domain.Task
		kind prompt.Kind
		want string
	}{
		"implementation for a task without a plan": {
			task: designTask(), kind: prompt.KindImplementation, want: "implement T-7",
		},
		"design for a task with a plan": {
			task: planned, kind: prompt.KindDesign, want: "design T-7",
		},
	} {
		t.Run(name, func(t *testing.T) {
			kind, body, err := prompt.Render(test.task, test.kind, templates, prompt.LanguageEnglish)
			if err != nil {
				t.Fatal(err)
			}
			if kind != test.kind || body != test.want {
				t.Fatalf("kind=%q body=%q, want %q %q", kind, body, test.kind, test.want)
			}
		})
	}
	// batch 系はタスク 1 件のプロンプトを持たないので、導出へ落として
	// batch テンプレートが単一タスクに漏れ出さないようにする。
	kind, body, err := prompt.Render(designTask(), prompt.KindBatch, templates, prompt.LanguageEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindDesign || body != "design T-7" {
		t.Fatalf("kind=%q body=%q", kind, body)
	}
}

// 一括設計は一括実装とは別のテンプレートを使う。実装用の文面は「実装する」前提
// で書かれており、設計をまとめて回す用途では破綻するためである。
func TestRenderBatchSelectsTheRequestedBatchTemplate(t *testing.T) {
	templates := prompt.Templates{
		Batch:       "implement {{feature_id}}\n{{task_list}}",
		BatchDesign: "design {{feature_id}}\n{{task_list}}",
	}
	kind, body, err := prompt.RenderBatch(
		"F-3", batchTasks(), prompt.KindBatchDesign, templates, prompt.LanguageEnglish,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "design F-3\n- T-7: Add the checkout API\n- T-9: Bill the order"
	if kind != prompt.KindBatchDesign || body != want {
		t.Fatalf("kind=%q body=%q, want %q", kind, body, want)
	}
	// 未指定の呼び出しは従来どおり一括実装のままにする。
	kind, body, err = prompt.RenderBatch("F-3", batchTasks(), "", templates, prompt.LanguageEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindBatch || !strings.HasPrefix(body, "implement F-3") {
		t.Fatalf("kind=%q body=%q", kind, body)
	}
}

// 組み込みの一括設計テンプレートは SubAgent を設計プロンプトへ向け、pull request
// ではなく計画の登録で終わらせる。どちらの言語でも同じ条項が要る。
func TestDefaultBatchDesignTemplatesSendEverySubAgentToTheDesignPrompt(t *testing.T) {
	for _, language := range prompt.SupportedLanguages() {
		_, body, err := prompt.RenderBatch(
			"F-3", batchTasks(), prompt.KindBatchDesign, prompt.Templates{}, language,
		)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{
			"prx prompt TASK_ID --kind design", "prx plan set TASK_ID --file PATH", "git worktree",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("the %s batch design prompt does not mention %q: %q", language, want, body)
			}
		}
		if strings.Contains(body, "{{") {
			t.Fatalf("the %s batch design prompt kept a placeholder: %q", language, body)
		}
	}
}

// 実装プロンプトは計画のない task にも渡せるので、組み込みテンプレートは計画の
// 取得が失敗したときの進め方を示していなければならない。
func TestDefaultImplementationTemplatesHandleAMissingPlan(t *testing.T) {
	for _, language := range prompt.SupportedLanguages() {
		_, body, err := prompt.Render(
			designTask(), prompt.KindImplementation, prompt.Templates{}, language,
		)
		if err != nil {
			t.Fatal(err)
		}
		want := "That is not an error to fix"
		if language == prompt.LanguageJapanese {
			want = "これは直すべきエラーではない"
		}
		if !strings.Contains(body, want) {
			t.Fatalf("the %s implementation prompt does not mention %q: %q", language, want, body)
		}
	}
}

// batch_design は batch と同じ語彙で検証する。上書きだけが別の語彙を持つと、
// 設定 UI が提示する placeholder 一覧と保存の可否がずれる。
func TestBatchDesignSharesTheBatchVocabulary(t *testing.T) {
	templates := prompt.Templates{BatchDesign: "design {{task_list}} {{task_title}}"}
	message := "prompts.batch_design: template uses unsupported placeholder {{task_title}}"
	if _, err := templates.Normalize(prompt.LanguageEnglish); err == nil || err.Error() != message {
		t.Fatalf("error=%v, want %q", err, message)
	}
	if err := prompt.ValidateOverride(prompt.KindBatchDesign, "design {{feature_id}}"); err == nil {
		t.Fatal("ValidateOverride accepted a batch design override without {{task_list}}")
	}
	if err := prompt.ValidateOverride(prompt.KindBatchDesign, "design {{task_list}}"); err != nil {
		t.Fatal(err)
	}
}
