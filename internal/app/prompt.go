package app

import (
	"context"
	"errors"
	"strings"

	"github.com/HappyOnigiri/nnx/internal/domain"
	"github.com/HappyOnigiri/nnx/internal/prompt"
)

// GetTaskPrompt は global、所属 project、feature のテンプレートを種類ごとに
// 解決してから、現在の task を描画する。CLI と RPC が同じ結果を使う入口である。
// kind が空なら実装計画の有無から導出する。
func (s *Service) GetTaskPrompt(
	ctx context.Context,
	taskID string,
	kind prompt.Kind,
) (prompt.Kind, string, error) {
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return "", "", err
	}
	task, ok := snapshotTask(snapshot, taskID)
	if !ok {
		return "", "", domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", taskID)
	}
	feature, err := s.ResolveFeature(ctx, task.FeatureID)
	if err != nil {
		return "", "", err
	}
	templates, language, err := s.resolvePromptTemplates(ctx, feature)
	if err != nil {
		return "", "", err
	}
	return prompt.Render(task, kind, templates, language)
}

// GetBatchPrompt は指定された feature の task 群を検証し、task prompt と同じ
// 種類別解決結果を使って一括 prompt を描画する。kind が batch 系でなければ
// 一括実装に落とす。
func (s *Service) GetBatchPrompt(
	ctx context.Context,
	featureID string,
	taskIDs []string,
	kind prompt.Kind,
) (prompt.Kind, string, error) {
	if len(taskIDs) == 0 {
		return "", "", domain.NewError(
			domain.DomainErrorCodeInvalidParent,
			"a batch prompt needs at least one task of feature %q",
			featureID,
		)
	}
	feature, err := s.ResolveFeature(ctx, featureID)
	if err != nil {
		return "", "", err
	}
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return "", "", err
	}
	tasks := make([]domain.Task, 0, len(taskIDs))
	for _, id := range taskIDs {
		task, ok := snapshotTask(snapshot, id)
		if !ok {
			return "", "", domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", id)
		}
		if task.FeatureID != feature.ID {
			return "", "", domain.NewError(
				domain.DomainErrorCodeInvalidParent,
				"task %q does not belong to feature %q",
				id,
				feature.ID,
			)
		}
		tasks = append(tasks, task)
	}
	if err := requirePromptBlockers(tasks); err != nil {
		return "", "", err
	}
	templates, language, err := s.resolvePromptTemplates(ctx, feature)
	if err != nil {
		return "", "", err
	}
	return prompt.RenderBatch(feature.ID, tasks, kind, templates, language)
}

func (s *Service) resolvePromptTemplates(
	ctx context.Context,
	feature domain.Feature,
) (prompt.Templates, prompt.Language, error) {
	if s.configStore == nil {
		return prompt.Templates{}, "", domain.NewError(
			domain.DomainErrorCodeInvalidConfig,
			"prompt templates require a configuration store",
		)
	}
	settings, err := s.configStore.Load()
	if err != nil {
		return prompt.Templates{}, "", configDomainError(err)
	}
	project, err := s.ResolveProject(ctx, feature.ProjectID)
	if err != nil {
		return prompt.Templates{}, "", err
	}
	language := settings.EffectiveLanguage()
	resolved, err := prompt.Resolve(language, settings.Prompts, project.PromptOverrides, feature.PromptOverrides)
	if err != nil {
		var typed *prompt.Error
		if errors.As(err, &typed) {
			if !strings.HasPrefix(typed.Field, "project.") && !strings.HasPrefix(typed.Field, "feature.") {
				return prompt.Templates{}, "", domain.NewError(
					domain.DomainErrorCodeInvalidConfig,
					"%s",
					typed.Error(),
				)
			}
			return prompt.Templates{}, "", domain.NewError(
				domain.DomainErrorCodeInvalidPromptTemplate,
				"%s",
				typed.Error(),
			)
		}
		return prompt.Templates{}, "", err
	}
	return resolved, language, nil
}

func snapshotTask(snapshot domain.Snapshot, id string) (domain.Task, bool) {
	for _, task := range snapshot.Tasks {
		if task.ID == id {
			return task, true
		}
	}
	return domain.Task{}, false
}

func requirePromptBlockers(tasks []domain.Task) error {
	included := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		included[task.ID] = struct{}{}
	}
	for _, task := range tasks {
		for _, blockerID := range task.PendingBlockerTaskIDs {
			if _, ok := included[blockerID]; ok {
				continue
			}
			return domain.NewError(
				domain.DomainErrorCodeInvalidParent,
				"task %q waits for task %q, which the batch does not include",
				task.ID,
				blockerID,
			)
		}
	}
	return nil
}
