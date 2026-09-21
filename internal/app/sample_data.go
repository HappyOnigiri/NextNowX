package app

import (
	"context"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

// DatabaseFileCreator は、その Open がデータベースファイルを新規作成したかを
// 報告する repository が実装する。任意のままにすることで、application 層の
// テストが使う小さな repository の fake をそのまま保てる。
type DatabaseFileCreator interface {
	CreatedDatabaseFile() bool
}

// sampleTaskText はサンプルの task 1 件分の文言。status と依存はグラフ側が持つ。
type sampleTaskText struct {
	title string
	scope string
}

// sampleDataText はサンプルデータの文言一式。実効言語ごとに 1 つ用意する。
type sampleDataText struct {
	projectTitle       string
	projectDescription string
	featureTitle       string
	featureDescription string
	documentTitle      string
	documentContent    string
	planContent        string
	tasks              []sampleTaskText
}

// EnsureSampleData は、この Open がデータベースファイルを新規作成していたときだけ
// サンプルを投入し、投入したかを返す。判定と投入を 1 つにまとめるのは、答えてから
// 書くまでの間に隙間を作らないためである。docs/design/persistence.md を参照。
func (s *Service) EnsureSampleData(ctx context.Context) (bool, error) {
	creator, ok := s.repository.(DatabaseFileCreator)
	if !ok || !creator.CreatedDatabaseFile() {
		return false, nil
	}
	text := sampleTextFor(s.sampleDataLanguage())
	project, err := s.CreateProject(ctx, text.projectTitle, text.projectDescription)
	if err != nil {
		return false, err
	}
	if err := s.seedSampleGraph(ctx, project.ID, text); err != nil {
		// 途中で失敗したサンプルは、中途半端なグラフを初回の画面に残すより
		// 消したほうがよい。巻き戻しの失敗はこれ以上追わない。
		_ = s.DeleteProject(ctx, project.ID, true)
		return false, err
	}
	return true, nil
}

// sampleDataLanguage はサンプルの文言に使う実効言語を決める。設定を読めない初回でも
// 投入は続けるので、失敗はロケール解決へのフォールバックとして扱う。
func (s *Service) sampleDataLanguage() prompt.Language {
	if s.configStore == nil {
		return prompt.ResolveLanguage(config.LanguageAutoValue)
	}
	settings, err := s.configStore.Load()
	if err != nil {
		return prompt.ResolveLanguage(config.LanguageAutoValue)
	}
	return settings.EffectiveLanguage()
}

// seedSampleGraph は表示状態を一通り見せる最小のグラフを作る。
// 実在しないリポジトリの同期エラーを初回から残さないよう pull request は付けない。
func (s *Service) seedSampleGraph(ctx context.Context, projectID string, text sampleDataText) error {
	if _, err := s.AddDocument(
		ctx,
		domain.DocumentParent{ProjectID: projectID},
		domain.DocumentKindMarkdown,
		text.documentTitle,
		"",
		text.documentContent,
		false,
	); err != nil {
		return err
	}
	feature, err := s.CreateFeature(ctx, text.featureTitle, text.featureDescription, projectID)
	if err != nil {
		return err
	}
	tasks := make([]domain.Task, len(text.tasks))
	for index, value := range text.tasks {
		task, err := s.CreateTask(ctx, feature.ID, value.title, value.scope, "")
		if err != nil {
			return err
		}
		if status := sampleTaskStatuses[index]; status != domain.TaskStatusNotStarted {
			if task, err = s.UpdateTask(ctx, task.ID, nil, nil, &status, nil); err != nil {
				return err
			}
		}
		tasks[index] = task
	}
	// 4 番目だけが plan を持つ。未着手のまま designed かつ ready になる組み合わせを
	// 初回の画面に出すためである。
	plan := domain.Document{Kind: domain.DocumentKindMarkdown, Content: text.planContent}
	if _, err := s.UpsertImplementationPlan(ctx, tasks[3].ID, plan); err != nil {
		return err
	}
	for _, edge := range sampleDependencyEdges {
		if _, err := s.AddDependency(ctx, tasks[edge[0]].ID, tasks[edge[1]].ID); err != nil {
			return err
		}
	}
	return nil
}

// sampleTaskStatuses は文言と同じ並びの status。完了 2 件・進行中 1 件・未着手 3 件で、
// 最後の 1 件だけが未完了の blocker を持つので blocked のまま残る。
var sampleTaskStatuses = []domain.TaskStatus{
	domain.TaskStatusCompleted,
	domain.TaskStatusCompleted,
	domain.TaskStatusInProgress,
	domain.TaskStatusNotStarted,
	domain.TaskStatusNotStarted,
	domain.TaskStatusNotStarted,
}

// sampleDependencyEdges は blocker と blocked の添字の組。
var sampleDependencyEdges = [][2]int{{0, 2}, {1, 3}, {1, 4}, {2, 5}, {3, 5}, {4, 5}}

func sampleTextFor(language prompt.Language) sampleDataText {
	if language == prompt.LanguageJapanese {
		return japaneseSampleText()
	}
	return englishSampleText()
}

func englishSampleText() sampleDataText {
	return sampleDataText{
		projectTitle:       "Sample project",
		projectDescription: "A first look at how PRX groups features and their work.",
		featureTitle:       "Sample feature",
		featureDescription: "Six tasks and the dependencies between them.",
		documentTitle:      "Getting started",
		documentContent:    englishSampleDocument,
		planContent:        englishSamplePlan,
		tasks: []sampleTaskText{
			{title: "Sketch the plan", scope: "Agree on what this feature delivers"},
			{title: "Prepare the workspace", scope: "Get the tools and the repository ready"},
			{title: "Build the core flow", scope: "Implement the path that everything else needs"},
			{title: "Write the guide", scope: "Describe the feature for the people who use it"},
			{title: "Polish the UI", scope: "Finish the screens on top of the core flow"},
			{title: "Ship the release", scope: "Cut the release once the work above lands"},
		},
	}
}

const englishSampleDocument = `# Getting started

PRX was just installed, so this sample project shows what it tracks.

- A task carries a status, and its display state also follows its plan and its pull request.
- A dependency says which task has to land first; the task that waits stays blocked.
- Sample feature has six tasks: two are completed, one is in progress, three are not started.
- Write the guide has an implementation plan, so it shows as designed while staying ready.
- Ship the release waits on everything else, so it stays blocked.

Delete this project once you no longer need it, or start the next setup with
prx setup --no-sample-data to skip it.
`

const englishSamplePlan = `1. List the steps the feature needs.
2. Write them down where the reader looks first.
3. Check the result against the graph.
`
