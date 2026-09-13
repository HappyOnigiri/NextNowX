package rpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/prompt"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func newPromptClient(t *testing.T) prxv1connect.PRXServiceClient {
	t.Helper()
	// 組み込みテンプレートは実効言語で決まる。設定は auto のままなので、
	// 期待値が実行環境のロケールで変わらないよう言語を固定する。
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(name, "en_US.UTF-8")
	}
	root := t.TempDir()
	database, err := store.Open(context.Background(), filepath.Join(root, "prompt.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	configStore, err := config.NewStore(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return newConfigClient(t, app.NewWithConfig(database, nil, configStore))
}

func createPromptTask(t *testing.T, client prxv1connect.PRXServiceClient) string {
	t.Helper()
	ctx := context.Background()
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{
			Title:     "Prompts",
			ProjectId: newRPCProject(t, ctx, client, "Prompts"),
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	task, err := client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{
		FeatureId: feature.Msg.GetFeature().GetId(),
		Title:     "Add the checkout API",
		Scope:     "Server only",
	}))
	if err != nil {
		t.Fatal(err)
	}
	return task.Msg.GetTask().GetId()
}

func TestRPCPromptTemplatesRoundTripAndDriveTheTaskPrompt(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)
	taskID := createPromptTask(t, client)

	stored, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Msg.GetTemplates().GetDesign() != prompt.DefaultTemplates(prompt.LanguageEnglish).Design {
		t.Fatalf("design template=%q, want the built-in template", stored.Msg.GetTemplates().GetDesign())
	}
	// 語彙はテンプレートと一緒に届く。エディタがこのサーバーの受理内容を
	// 自前で持たなくて済むようにするため。
	if !slices.Equal(stored.Msg.GetSupportedPlaceholders(), prompt.SupportedPlaceholders()) {
		t.Fatalf("supported placeholders=%v, want %v",
			stored.Msg.GetSupportedPlaceholders(), prompt.SupportedPlaceholders())
	}
	if stored.Msg.GetRequiredPlaceholder() != prompt.RequiredPlaceholder() {
		t.Fatalf("required placeholder=%q, want %q",
			stored.Msg.GetRequiredPlaceholder(), prompt.RequiredPlaceholder())
	}

	updated, err := client.UpdatePromptTemplates(ctx, connect.NewRequest(&prxv1.UpdatePromptTemplatesRequest{
		Design:         "Design {{task_id}}: {{task_title}}\n",
		Implementation: "Implement {{task_id}} in {{feature_id}}\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg.GetTemplates().GetImplementation() != "Implement {{task_id}} in {{feature_id}}\n" {
		t.Fatalf("implementation template=%q", updated.Msg.GetTemplates().GetImplementation())
	}

	// 組み込みの組は保存済みの組と併せて提供される。エディタが実際に書き込む
	// 前に、復元したら何が書かれるかを見せられるようにするため。
	customized, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	english := prompt.DefaultTemplates(prompt.LanguageEnglish)
	if customized.Msg.GetBuiltIn().GetDesign() != english.Design ||
		customized.Msg.GetBuiltIn().GetImplementation() != english.Implementation {
		t.Fatalf("built-in templates=%+v", customized.Msg.GetBuiltIn())
	}

	design, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: taskID}))
	if err != nil {
		t.Fatal(err)
	}
	if design.Msg.GetKind() != prxv1.TaskPromptKind_TASK_PROMPT_KIND_DESIGN ||
		design.Msg.GetPrompt() != "Design "+taskID+": Add the checkout API\n" ||
		design.Msg.GetTaskId() != taskID {
		t.Fatalf("design prompt=%+v", design.Msg)
	}

	// 計画を登録すると、呼び出し元からの他の入力なしに、同じリクエストが
	// 描画するテンプレートが変わる。
	if _, err := client.AddDocument(ctx, connect.NewRequest(&prxv1.AddDocumentRequest{
		TaskId:               taskID,
		Title:                "Plan",
		Source:               &prxv1.AddDocumentRequest_Markdown{Markdown: "# Plan\n"},
		IsImplementationPlan: true,
	})); err != nil {
		t.Fatal(err)
	}
	implementation, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: taskID}))
	if err != nil {
		t.Fatal(err)
	}
	if implementation.Msg.GetKind() != prxv1.TaskPromptKind_TASK_PROMPT_KIND_IMPLEMENTATION ||
		!strings.HasPrefix(implementation.Msg.GetPrompt(), "Implement "+taskID+" in F-") {
		t.Fatalf("implementation prompt=%+v", implementation.Msg)
	}
}

func TestRPCPromptFailuresUseTheConfigurationVocabulary(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)

	_, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: "T-404"}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_NOT_FOUND {
		t.Fatalf("missing task error=%v", err)
	}

	_, err = client.UpdatePromptTemplates(ctx, connect.NewRequest(&prxv1.UpdatePromptTemplatesRequest{
		Design:         "no target",
		Implementation: "Implement {{task_id}}",
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_CONFIG {
		t.Fatalf("invalid template error=%v", err)
	}
	// 拒否された書き込みは保存済みの組をそのままにしている。
	stored, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
	if err != nil || stored.Msg.GetTemplates().GetDesign() != prompt.DefaultTemplates(prompt.LanguageEnglish).Design {
		t.Fatalf("stored templates=%+v err=%v", stored.Msg.GetTemplates(), err)
	}
}

func TestRPCPromptMethodsAreUnavailableWithoutConfigStore(t *testing.T) {
	path, handler := rpc.New(app.New(noopRepository{}, nil))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"GetPromptTemplates": func() error {
			_, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
			return err
		},
		"UpdatePromptTemplates": func() error {
			_, err := client.UpdatePromptTemplates(ctx, connect.NewRequest(&prxv1.UpdatePromptTemplatesRequest{}))
			return err
		},
		"GetTaskPrompt": func() error {
			_, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: "T-1"}))
			return err
		},
		"GetBatchPrompt": func() error {
			_, err := client.GetBatchPrompt(ctx, connect.NewRequest(&prxv1.GetBatchPromptRequest{
				FeatureId: "F-1", TaskIds: []string{"T-1"},
			}))
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); connect.CodeOf(err) != connect.CodeUnimplemented {
				t.Fatalf("code=%s err=%v", connect.CodeOf(err), err)
			}
		})
	}
}

// batch プロンプトは呼び出し元が選んだタスクから生成される。そこでこのテストは
// 1 つの選択を保存済みテンプレート経由で追い、選択が元の feature を表さなくなる
// 2 通りのケースを確認する。
func TestRPCBatchPromptCoversTheSelectedTasksOfOneFeature(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)
	featureID, taskIDs := createBatchFeature(t, client, "Batch", "First task", "Second task")

	stored, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Msg.GetTemplates().GetBatch() != prompt.DefaultTemplates(prompt.LanguageEnglish).Batch {
		t.Fatalf("batch template=%q, want the built-in template", stored.Msg.GetTemplates().GetBatch())
	}
	if !slices.Equal(stored.Msg.GetBatchSupportedPlaceholders(), prompt.BatchSupportedPlaceholders()) ||
		stored.Msg.GetBatchRequiredPlaceholder() != prompt.BatchRequiredPlaceholder() {
		t.Fatalf("batch vocabulary=%+v", stored.Msg)
	}

	if _, err := client.UpdatePromptTemplates(ctx, connect.NewRequest(&prxv1.UpdatePromptTemplatesRequest{
		Design:         prompt.DefaultTemplates(prompt.LanguageEnglish).Design,
		Implementation: prompt.DefaultTemplates(prompt.LanguageEnglish).Implementation,
		Batch:          "Batch {{feature_id}}\n{{task_list}}\n",
	})); err != nil {
		t.Fatal(err)
	}

	batch, err := client.GetBatchPrompt(ctx, connect.NewRequest(&prxv1.GetBatchPromptRequest{
		FeatureId: featureID, TaskIds: taskIDs,
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := "Batch " + featureID + "\n" +
		"- " + taskIDs[0] + ": First task\n" +
		"- " + taskIDs[1] + ": Second task\n"
	if batch.Msg.GetPrompt() != want {
		t.Fatalf("batch prompt=%q, want %q", batch.Msg.GetPrompt(), want)
	}
	if batch.Msg.GetFeatureId() != featureID || !slices.Equal(batch.Msg.GetTaskIds(), taskIDs) {
		t.Fatalf("batch response=%+v", batch.Msg)
	}
}

func TestRPCBatchPromptRejectsASelectionTheFeatureDoesNotOwn(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)
	featureID, taskIDs := createBatchFeature(t, client, "Batch", "First task")
	otherFeatureID, otherTaskIDs := createBatchFeature(t, client, "Other", "Other task")

	_, err := client.GetBatchPrompt(ctx, connect.NewRequest(&prxv1.GetBatchPromptRequest{
		FeatureId: featureID, TaskIds: []string{taskIDs[0], "T-404"},
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_NOT_FOUND {
		t.Fatalf("missing task error=%v", err)
	}

	// 別の feature のタスクは存在しないタスクとは違う。読み手は実在する作業を
	// 選んだのだから、メッセージはそれがどの feature のものかを伝える必要がある。
	_, err = client.GetBatchPrompt(ctx, connect.NewRequest(&prxv1.GetBatchPromptRequest{
		FeatureId: featureID, TaskIds: []string{taskIDs[0], otherTaskIDs[0]},
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PARENT ||
		!strings.Contains(err.Error(), otherTaskIDs[0]) {
		t.Fatalf("foreign task error=%v", err)
	}
	if otherFeatureID == featureID {
		t.Fatal("the two features share an identifier")
	}

	_, err = client.GetBatchPrompt(ctx, connect.NewRequest(&prxv1.GetBatchPromptRequest{
		FeatureId: featureID,
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PARENT {
		t.Fatalf("empty selection error=%v", err)
	}
}

// ブロックされたタスクも渡してよい。エージェントは待ち相手の作業のあとに実装し、
// PR を積み重ねるため。ただしそれはブロッカーが同じ batch にある間だけ成り立つので、
// ブロッカーを含まない選択は拒否する。
func TestRPCBatchPromptRejectsASelectionMissingABlocker(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)
	featureID, taskIDs := createBatchFeature(t, client, "Batch", "First task", "Second task")
	if _, err := client.AddDependency(ctx, connect.NewRequest(&prxv1.AddDependencyRequest{
		BlockerTaskId: taskIDs[0], BlockedTaskId: taskIDs[1],
	})); err != nil {
		t.Fatal(err)
	}

	_, err := client.GetBatchPrompt(ctx, connect.NewRequest(&prxv1.GetBatchPromptRequest{
		FeatureId: featureID, TaskIds: []string{taskIDs[1]},
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PARENT ||
		!strings.Contains(err.Error(), taskIDs[0]) {
		t.Fatalf("missing blocker error=%v", err)
	}

	// batch が待ち相手を含んだ途端、同じタスクが描画されるようになる。
	batch, err := client.GetBatchPrompt(ctx, connect.NewRequest(&prxv1.GetBatchPromptRequest{
		FeatureId: featureID, TaskIds: taskIDs,
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range taskIDs {
		if !strings.Contains(batch.Msg.GetPrompt(), id) {
			t.Fatalf("batch prompt=%q, want it to name %q", batch.Msg.GetPrompt(), id)
		}
	}
}

func TestRPCPromptOverridesResolveAtProjectAndFeatureScope(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)
	projectID := newRPCProject(t, ctx, client, "Prompt overrides")
	feature, err := client.CreateFeature(ctx, connect.NewRequest(&prxv1.CreateFeatureRequest{
		Title: "Override feature", ProjectId: projectID,
	}))
	if err != nil {
		t.Fatal(err)
	}
	featureID := feature.Msg.GetFeature().GetId()
	task, err := client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{
		FeatureId: featureID, Title: "Override task",
	}))
	if err != nil {
		t.Fatal(err)
	}
	taskID := task.Msg.GetTask().GetId()
	projectDesign := "Project design {{task_id}}"
	featureBatch := "Feature batch {{task_list}}"
	updatedProject, err := client.UpdateProject(ctx, connect.NewRequest(&prxv1.UpdateProjectRequest{
		Id:              projectID,
		PromptOverrides: &prxv1.PromptTemplateOverridesUpdate{Design: &projectDesign},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if updatedProject.Msg.GetProject().GetPromptOverrides().GetDesign() != projectDesign {
		t.Fatalf("project overrides=%+v", updatedProject.Msg.GetProject().GetPromptOverrides())
	}
	updatedFeature, err := client.UpdateFeature(ctx, connect.NewRequest(&prxv1.UpdateFeatureRequest{
		Id:              featureID,
		PromptOverrides: &prxv1.PromptTemplateOverridesUpdate{Batch: &featureBatch},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if updatedFeature.Msg.GetFeature().GetPromptOverrides().GetBatch() != featureBatch {
		t.Fatalf("feature overrides=%+v", updatedFeature.Msg.GetFeature().GetPromptOverrides())
	}
	result, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: taskID}))
	if err != nil || result.Msg.GetPrompt() != "Project design "+taskID {
		t.Fatalf("task prompt=%+v err=%v", result.Msg, err)
	}
	batch, err := client.GetBatchPrompt(ctx, connect.NewRequest(&prxv1.GetBatchPromptRequest{
		FeatureId: featureID, TaskIds: []string{taskID},
	}))
	if err != nil || !strings.HasPrefix(batch.Msg.GetPrompt(), "Feature batch ") {
		t.Fatalf("batch prompt=%+v err=%v", batch.Msg, err)
	}
	bad := "missing {{task_id}}?"
	_, err = client.UpdateFeature(ctx, connect.NewRequest(&prxv1.UpdateFeatureRequest{
		Id:              featureID,
		PromptOverrides: &prxv1.PromptTemplateOverridesUpdate{Batch: &bad},
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PROMPT_TEMPLATE ||
		!strings.Contains(err.Error(), "feature prompt override batch") {
		t.Fatalf("invalid override error=%v", err)
	}
}

func createBatchFeature(
	t *testing.T,
	client prxv1connect.PRXServiceClient,
	featureTitle string,
	taskTitles ...string,
) (string, []string) {
	t.Helper()
	ctx := context.Background()
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{
			Title:     featureTitle,
			ProjectId: newRPCProject(t, ctx, client, featureTitle),
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	featureID := feature.Msg.GetFeature().GetId()
	taskIDs := make([]string, 0, len(taskTitles))
	for _, title := range taskTitles {
		task, err := client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{
			FeatureId: featureID, Title: title,
		}))
		if err != nil {
			t.Fatal(err)
		}
		taskIDs = append(taskIDs, task.Msg.GetTask().GetId())
	}
	return featureID, taskIDs
}

// 言語は設定ファイルに置く共有の値なので、切り替えると組み込みテンプレートも
// レンダリング結果も追従する。WebUI の表示言語はこの実効言語に従う。
func TestRPCLanguageConfigSwitchesTheBuiltInTemplates(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)
	taskID := createPromptTask(t, client)

	settings, err := client.GetConfig(ctx, connect.NewRequest(&prxv1.GetConfigRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if settings.Msg.GetConfig().GetLanguage() != config.LanguageAutoValue ||
		settings.Msg.GetConfig().GetEffectiveLanguage() != string(prompt.LanguageEnglish) {
		t.Fatalf("config=%+v", settings.Msg.GetConfig())
	}

	updated, err := client.UpdateLanguageConfig(ctx, connect.NewRequest(&prxv1.UpdateLanguageConfigRequest{
		Language: string(prompt.LanguageJapanese),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg.GetConfig().GetLanguage() != string(prompt.LanguageJapanese) ||
		updated.Msg.GetConfig().GetEffectiveLanguage() != string(prompt.LanguageJapanese) {
		t.Fatalf("config=%+v", updated.Msg.GetConfig())
	}

	stored, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	japanese := prompt.DefaultTemplates(prompt.LanguageJapanese)
	if stored.Msg.GetBuiltIn().GetDesign() != japanese.Design ||
		stored.Msg.GetTemplates().GetDesign() != japanese.Design {
		t.Fatalf("design template=%q, want the japanese built-in template", stored.Msg.GetTemplates().GetDesign())
	}

	design, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: taskID}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(design.Msg.GetPrompt(), "設計する") {
		t.Fatalf("design prompt=%q, want japanese", design.Msg.GetPrompt())
	}

	_, err = client.UpdateLanguageConfig(ctx, connect.NewRequest(&prxv1.UpdateLanguageConfigRequest{
		Language: "fr",
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_CONFIG {
		t.Fatalf("unsupported language error=%v", err)
	}
}
