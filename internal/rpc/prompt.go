package rpc

import (
	"context"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

func (h *Handler) GetPromptTemplates(
	_ context.Context,
	_ *connect.Request[prxv1.GetPromptTemplatesRequest],
) (*connect.Response[prxv1.GetPromptTemplatesResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Load()
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.GetPromptTemplatesResponse{
		Templates:             protoPromptTemplates(settings.Prompts),
		SupportedPlaceholders: prompt.SupportedPlaceholders(),
		RequiredPlaceholder:   prompt.RequiredPlaceholder(),
		BuiltIn:               protoPromptTemplates(prompt.DefaultTemplates(settings.EffectiveLanguage())),

		BatchSupportedPlaceholders: prompt.BatchSupportedPlaceholders(),
		BatchRequiredPlaceholder:   prompt.BatchRequiredPlaceholder(),
	}), nil
}

func (h *Handler) UpdatePromptTemplates(
	_ context.Context,
	req *connect.Request[prxv1.UpdatePromptTemplatesRequest],
) (*connect.Response[prxv1.UpdatePromptTemplatesResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Update(func(settings *config.Config) error {
		return settings.SetPrompts(prompt.Templates{
			Design:         req.Msg.GetDesign(),
			Implementation: req.Msg.GetImplementation(),
			Batch:          req.Msg.GetBatch(),
			BatchDesign:    req.Msg.GetBatchDesign(),
		})
	})
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.UpdatePromptTemplatesResponse{
		Templates: protoPromptTemplates(settings.Prompts),
	}), nil
}

// GetTaskPrompt は呼び出し元が思っているタスクの姿ではなく、現在のサーバー状態から
// プロンプトを生成する。WebUI のスナップショットは計画の登録や削除より前の
// 時点のものかもしれないため。
func (h *Handler) GetTaskPrompt(
	ctx context.Context,
	req *connect.Request[prxv1.GetTaskPromptRequest],
) (*connect.Response[prxv1.GetTaskPromptResponse], error) {
	if _, err := h.requireConfig(); err != nil {
		return nil, err
	}
	kind, body, err := h.service.GetTaskPrompt(ctx, req.Msg.GetTaskId(), taskPromptKind(req.Msg.GetKind()))
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.GetTaskPromptResponse{
		TaskId: req.Msg.GetTaskId(), Kind: protoTaskPromptKind(kind), Prompt: body,
	}), nil
}

// GetBatchPrompt は複数タスクをまとめた 1 つのプロンプトを生成する。各タスクは
// 現在のサーバー状態から解決し、指定された feature に属するか検査する。
// docs/design/agent-prompts.md を参照。
func (h *Handler) GetBatchPrompt(
	ctx context.Context,
	req *connect.Request[prxv1.GetBatchPromptRequest],
) (*connect.Response[prxv1.GetBatchPromptResponse], error) {
	if _, err := h.requireConfig(); err != nil {
		return nil, err
	}
	featureID := req.Msg.GetFeatureId()
	taskIDs := req.Msg.GetTaskIds()
	kind, body, err := h.service.GetBatchPrompt(ctx, featureID, taskIDs, batchPromptKind(req.Msg.GetKind()))
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.GetBatchPromptResponse{
		FeatureId: featureID, TaskIds: taskIDs, Prompt: body, Kind: protoBatchPromptKind(kind),
	}), nil
}

func protoPromptTemplates(value prompt.Templates) *prxv1.PromptTemplates {
	return &prxv1.PromptTemplates{
		Design:         value.Design,
		Implementation: value.Implementation,
		Batch:          value.Batch,
		BatchDesign:    value.BatchDesign,
	}
}

func protoTaskPromptKind(value prompt.Kind) prxv1.TaskPromptKind {
	if value == prompt.KindImplementation {
		return prxv1.TaskPromptKind_TASK_PROMPT_KIND_IMPLEMENTATION
	}
	return prxv1.TaskPromptKind_TASK_PROMPT_KIND_DESIGN
}

// protoBatchPromptKind は batch 種別を 2 値の enum へ戻す。batch と batch_design の
// 違いは設計か実装かだけなので、単一 task と同じ語彙で足りる。
func protoBatchPromptKind(value prompt.Kind) prxv1.TaskPromptKind {
	if value == prompt.KindBatchDesign {
		return prxv1.TaskPromptKind_TASK_PROMPT_KIND_DESIGN
	}
	return prxv1.TaskPromptKind_TASK_PROMPT_KIND_IMPLEMENTATION
}

// taskPromptKind は要求の enum を単一 task のテンプレート種別へ写す。
// UNSPECIFIED は空の Kind にして、実装計画からの導出をサービス層へ残す。
func taskPromptKind(value prxv1.TaskPromptKind) prompt.Kind {
	switch value {
	case prxv1.TaskPromptKind_TASK_PROMPT_KIND_DESIGN:
		return prompt.KindDesign
	case prxv1.TaskPromptKind_TASK_PROMPT_KIND_IMPLEMENTATION:
		return prompt.KindImplementation
	case prxv1.TaskPromptKind_TASK_PROMPT_KIND_UNSPECIFIED:
		return ""
	}
	return ""
}

// batchPromptKind は要求の enum を batch のテンプレート種別へ写す。
// UNSPECIFIED は従来どおりの一括実装を意味する。
func batchPromptKind(value prxv1.TaskPromptKind) prompt.Kind {
	if value == prxv1.TaskPromptKind_TASK_PROMPT_KIND_DESIGN {
		return prompt.KindBatchDesign
	}
	return prompt.KindBatch
}
