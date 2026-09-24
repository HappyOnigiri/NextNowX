package rpc

import (
	"sort"

	nnxv1 "github.com/HappyOnigiri/nnx/gen/nnx/v1"
	"github.com/HappyOnigiri/nnx/internal/domain"
)

func protoProject(v domain.Project) *nnxv1.Project {
	return &nnxv1.Project{
		Id:                 v.ID,
		Title:              v.Title,
		Description:        v.Description,
		Archived:           v.Archived,
		CreatedAt:          v.CreatedAt.Format(timeFormat),
		UpdatedAt:          v.UpdatedAt.Format(timeFormat),
		PromptOverrides:    protoPromptTemplateOverrides(v.PromptOverrides),
		TaskLabelOverrides: protoTaskLabelOverrides(v.TaskLabelOverrides),
	}
}

func protoFeature(v domain.Feature) *nnxv1.Feature {
	return &nnxv1.Feature{
		Id:                   v.ID,
		ProjectId:            v.ProjectID,
		ReadOnly:             v.ReadOnly,
		Title:                v.Title,
		Description:          v.Description,
		Status:               protoFeatureStatus(v.Status),
		Archived:             v.Archived,
		CreatedAt:            v.CreatedAt.Format(timeFormat),
		UpdatedAt:            v.UpdatedAt.Format(timeFormat),
		TaskCount:            int32(v.TaskCount),
		ReadyCount:           int32(v.ReadyCount),
		ReviewWaitingCount:   int32(v.ReviewWaitingCount),
		ConflictCount:        int32(v.ConflictCount),
		MergedCount:          int32(v.MergedCount),
		DisplayStatus:        protoFeatureStatus(v.DisplayStatus),
		FinishedCount:        int32(v.FinishedCount),
		PromptOverrides:      protoPromptTemplateOverrides(v.PromptOverrides),
		TaskLabelOverrides:   protoTaskLabelOverrides(v.TaskLabelOverrides),
		TaskLabelAppearances: protoTaskLabelAppearances(v.TaskLabelAppearances),
	}
}

func protoTaskLabelOverrides(values domain.TaskLabelOverrides) *nnxv1.TaskLabelOverrides {
	result := &nnxv1.TaskLabelOverrides{Values: map[string]*nnxv1.TaskLabelOverride{}}
	for key, value := range values {
		result.Values[string(key)] = &nnxv1.TaskLabelOverride{Text: value.Text, Color: value.Color}
	}
	return result
}

func domainTaskLabelOverridesUpdate(value *nnxv1.TaskLabelOverridesUpdate) *domain.TaskLabelOverridesUpdate {
	if value == nil {
		return nil
	}
	result := domain.TaskLabelOverridesUpdate{}
	for key, item := range value.GetValues() {
		if item == nil {
			result[domain.TaskLabelKey(key)] = domain.TaskLabelOverrideUpdate{}
			continue
		}
		result[domain.TaskLabelKey(key)] = domain.TaskLabelOverrideUpdate{
			Text: item.Text, Color: item.Color,
		}
	}
	return &result
}

func protoTaskLabelAppearances(values domain.TaskLabelAppearances) *nnxv1.TaskLabelAppearances {
	if len(values) == 0 {
		return nil
	}
	keys := make([]domain.TaskLabelKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	result := &nnxv1.TaskLabelAppearances{Values: make([]*nnxv1.TaskLabelAppearance, 0, len(keys))}
	for _, key := range keys {
		value := values[key]
		result.Values = append(result.Values, &nnxv1.TaskLabelAppearance{
			Key: string(key), Text: value.Text, Color: value.Color,
			TextOverridden: value.TextOverridden, ColorOverridden: value.ColorOverridden,
		})
	}
	return result
}

func protoPromptTemplateOverrides(v domain.PromptTemplateOverrides) *nnxv1.PromptTemplateOverrides {
	return &nnxv1.PromptTemplateOverrides{
		Design:         v.Design,
		Implementation: v.Implementation,
		Batch:          v.Batch,
		BatchDesign:    v.BatchDesign,
	}
}

func domainPromptTemplateOverridesUpdate(
	value *nnxv1.PromptTemplateOverridesUpdate,
) *domain.PromptTemplateOverridesUpdate {
	if value == nil {
		return nil
	}
	result := &domain.PromptTemplateOverridesUpdate{}
	if value.Design != nil {
		v := value.GetDesign()
		result.Design = &v
	}
	if value.Implementation != nil {
		v := value.GetImplementation()
		result.Implementation = &v
	}
	if value.Batch != nil {
		v := value.GetBatch()
		result.Batch = &v
	}
	if value.BatchDesign != nil {
		v := value.GetBatchDesign()
		result.BatchDesign = &v
	}
	return result
}

func protoTask(v domain.Task) *nnxv1.Task {
	return &nnxv1.Task{
		Id:                    v.ID,
		FeatureId:             v.FeatureID,
		Title:                 v.Title,
		Scope:                 v.Scope,
		Status:                protoTaskStatus(v.Status),
		Assignee:              v.Assignee,
		HasImplementationPlan: v.HasImplementationPlan,
		CreatedAt:             v.CreatedAt.Format(timeFormat),
		UpdatedAt:             v.UpdatedAt.Format(timeFormat),
		Ready:                 v.Ready,
		DisplayState:          protoTaskDisplayState(v.DisplayState),
		BlockedReason:         protoBlockedReason(v),
		PendingBlockerTaskIds: v.PendingBlockerTaskIDs,
		BlockLabels:           protoTaskBlockLabels(v.BlockLabels),
	}
}

func protoDependency(v domain.Dependency) *nnxv1.Dependency {
	return &nnxv1.Dependency{
		BlockerTaskId: v.BlockerTaskID,
		BlockedTaskId: v.BlockedTaskID,
		CreatedAt:     v.CreatedAt.Format(timeFormat),
	}
}

func protoPullRequest(v domain.PullRequest) *nnxv1.PullRequest {
	result := &nnxv1.PullRequest{
		TaskId:       v.TaskID,
		Host:         v.Host,
		Owner:        v.Owner,
		Repository:   v.Repository,
		Number:       v.Number,
		Url:          v.URL,
		NodeId:       v.NodeID,
		Author:       v.Author,
		Assignees:    v.Assignees,
		State:        protoPullRequestState(v.State),
		Draft:        v.Draft,
		ReviewState:  protoReviewState(v.ReviewState),
		Mergeability: protoMergeability(v.Mergeability),
		CheckState:   protoCheckState(v.CheckState),
		SyncError:    v.SyncError,
		Stale:        v.Stale,
		DisplayState: protoPullRequestDisplayState(v.DisplayState),

		ReviewRequestPending: v.ReviewRequestPending,
	}
	if v.GitHubUpdatedAt != nil {
		result.GithubUpdatedAt = v.GitHubUpdatedAt.Format(timeFormat)
	}
	if v.LastSyncedAt != nil {
		result.LastSyncedAt = v.LastSyncedAt.Format(timeFormat)
	}
	if v.ChangesRequestedAt != nil {
		result.ChangesRequestedAt = v.ChangesRequestedAt.Format(timeFormat)
	}
	if v.LastPushedAt != nil {
		result.LastPushedAt = v.LastPushedAt.Format(timeFormat)
	}
	return result
}

func protoDocument(v domain.Document) *nnxv1.Document {
	return &nnxv1.Document{
		Id:                   v.ID,
		ProjectId:            v.ProjectID,
		FeatureId:            v.FeatureID,
		TaskId:               v.TaskID,
		Kind:                 protoDocumentKind(v.Kind),
		Title:                v.Title,
		Locator:              v.Locator,
		CreatedAt:            v.CreatedAt.Format(timeFormat),
		UpdatedAt:            v.UpdatedAt.Format(timeFormat),
		IsImplementationPlan: v.IsImplementationPlan,
	}
}

func protoSnapshot(v domain.Snapshot) *nnxv1.Snapshot {
	result := &nnxv1.Snapshot{}
	for _, item := range v.Projects {
		result.Projects = append(result.Projects, protoProject(item))
	}
	for _, item := range v.Features {
		result.Features = append(result.Features, protoFeature(item))
	}
	for _, item := range v.Tasks {
		result.Tasks = append(result.Tasks, protoTask(item))
	}
	for _, item := range v.Dependencies {
		result.Dependencies = append(result.Dependencies, protoDependency(item))
	}
	for _, item := range v.PullRequests {
		result.PullRequests = append(result.PullRequests, protoPullRequest(item))
	}
	for _, item := range v.Documents {
		result.Documents = append(result.Documents, protoDocument(item))
	}
	for _, item := range v.ReadyTasks {
		result.ReadyTasks = append(result.ReadyTasks, protoTask(item))
	}
	for _, item := range v.ReviewWaitingTasks {
		result.ReviewWaitingTasks = append(result.ReviewWaitingTasks, protoTask(item))
	}
	for _, item := range v.ConflictTasks {
		result.ConflictTasks = append(result.ConflictTasks, protoTask(item))
	}
	for _, item := range v.StaleTasks {
		result.StaleTasks = append(result.StaleTasks, protoTask(item))
	}
	return result
}

func protoFeatureStatus(value domain.FeatureStatus) nnxv1.FeatureStatus {
	switch value {
	case domain.FeatureStatusAuto:
		return nnxv1.FeatureStatus_FEATURE_STATUS_AUTO
	case domain.FeatureStatusActive:
		return nnxv1.FeatureStatus_FEATURE_STATUS_ACTIVE
	case domain.FeatureStatusPaused:
		return nnxv1.FeatureStatus_FEATURE_STATUS_PAUSED
	case domain.FeatureStatusCompleted:
		return nnxv1.FeatureStatus_FEATURE_STATUS_COMPLETED
	case domain.FeatureStatusCancelled:
		return nnxv1.FeatureStatus_FEATURE_STATUS_CANCELLED
	default:
		return nnxv1.FeatureStatus_FEATURE_STATUS_UNSPECIFIED
	}
}

// domainFeatureStatus はサーバーがマップできない値を、空文字列にフォールバック
// せず拒否する。空文字列はサービス層が「フィールド省略」と解釈するため。
func domainFeatureStatus(value *nnxv1.FeatureStatus) (*domain.FeatureStatus, error) {
	if value == nil {
		return nil, nil
	}
	var result domain.FeatureStatus
	switch *value {
	case nnxv1.FeatureStatus_FEATURE_STATUS_AUTO:
		result = domain.FeatureStatusAuto
	case nnxv1.FeatureStatus_FEATURE_STATUS_ACTIVE:
		result = domain.FeatureStatusActive
	case nnxv1.FeatureStatus_FEATURE_STATUS_PAUSED:
		result = domain.FeatureStatusPaused
	case nnxv1.FeatureStatus_FEATURE_STATUS_COMPLETED:
		result = domain.FeatureStatusCompleted
	case nnxv1.FeatureStatus_FEATURE_STATUS_CANCELLED:
		result = domain.FeatureStatusCancelled
	case nnxv1.FeatureStatus_FEATURE_STATUS_UNSPECIFIED:
		return nil, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid feature status")
	default:
		return nil, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid feature status")
	}
	return &result, nil
}

func protoTaskStatus(value domain.TaskStatus) nnxv1.TaskStatus {
	switch value {
	case domain.TaskStatusNotStarted:
		return nnxv1.TaskStatus_TASK_STATUS_NOT_STARTED
	case domain.TaskStatusDesigning:
		return nnxv1.TaskStatus_TASK_STATUS_DESIGNING
	case domain.TaskStatusInProgress:
		return nnxv1.TaskStatus_TASK_STATUS_IN_PROGRESS
	case domain.TaskStatusCompleted:
		return nnxv1.TaskStatus_TASK_STATUS_COMPLETED
	case domain.TaskStatusClosed:
		return nnxv1.TaskStatus_TASK_STATUS_CLOSED
	default:
		return nnxv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// domainTaskStatus はサーバーがマップできない値を、空文字列にフォールバック
// せず拒否する。空文字列はサービス層が「フィールド省略」と解釈するため。
func domainTaskStatus(value *nnxv1.TaskStatus) (*domain.TaskStatus, error) {
	if value == nil {
		return nil, nil
	}
	var result domain.TaskStatus
	switch *value {
	case nnxv1.TaskStatus_TASK_STATUS_NOT_STARTED:
		result = domain.TaskStatusNotStarted
	case nnxv1.TaskStatus_TASK_STATUS_DESIGNING:
		result = domain.TaskStatusDesigning
	case nnxv1.TaskStatus_TASK_STATUS_IN_PROGRESS:
		result = domain.TaskStatusInProgress
	case nnxv1.TaskStatus_TASK_STATUS_COMPLETED:
		result = domain.TaskStatusCompleted
	case nnxv1.TaskStatus_TASK_STATUS_CLOSED:
		result = domain.TaskStatusClosed
	case nnxv1.TaskStatus_TASK_STATUS_UNSPECIFIED:
		return nil, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid task status")
	default:
		return nil, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid task status")
	}
	return &result, nil
}

func protoTaskDisplayState(value domain.TaskDisplayState) nnxv1.TaskDisplayState {
	states := map[domain.TaskDisplayState]nnxv1.TaskDisplayState{
		domain.TaskDisplayStateNotStarted:  nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_NOT_STARTED,
		domain.TaskDisplayStateDesigning:   nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_DESIGNING,
		domain.TaskDisplayStateDesigned:    nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_DESIGNED,
		domain.TaskDisplayStateInProgress:  nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_IN_PROGRESS,
		domain.TaskDisplayStateImplemented: nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_IMPLEMENTED,
		domain.TaskDisplayStateInReview:    nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_IN_REVIEW,
		domain.TaskDisplayStateApproved:    nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_APPROVED,
		domain.TaskDisplayStateMerged:      nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_MERGED,
		domain.TaskDisplayStateCompleted:   nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_COMPLETED,
		domain.TaskDisplayStateClosed:      nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_CLOSED,
		domain.TaskDisplayStateUnknown:     nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNKNOWN,
	}
	if state, ok := states[value]; ok {
		return state
	}
	return nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNSPECIFIED
}

func protoPullRequestState(value domain.PullRequestState) nnxv1.PullRequestState {
	switch value {
	case domain.PullRequestStateOpen:
		return nnxv1.PullRequestState_PULL_REQUEST_STATE_OPEN
	case domain.PullRequestStateClosed:
		return nnxv1.PullRequestState_PULL_REQUEST_STATE_CLOSED
	case domain.PullRequestStateMerged:
		return nnxv1.PullRequestState_PULL_REQUEST_STATE_MERGED
	case domain.PullRequestStateUnknown:
		return nnxv1.PullRequestState_PULL_REQUEST_STATE_UNKNOWN
	default:
		return nnxv1.PullRequestState_PULL_REQUEST_STATE_UNSPECIFIED
	}
}

func protoReviewState(value domain.ReviewState) nnxv1.ReviewState {
	switch value {
	case domain.ReviewStateNone:
		return nnxv1.ReviewState_REVIEW_STATE_NONE
	case domain.ReviewStateRequired:
		return nnxv1.ReviewState_REVIEW_STATE_REQUIRED
	case domain.ReviewStateApproved:
		return nnxv1.ReviewState_REVIEW_STATE_APPROVED
	case domain.ReviewStateChangesRequested:
		return nnxv1.ReviewState_REVIEW_STATE_CHANGES_REQUESTED
	case domain.ReviewStateUnknown:
		return nnxv1.ReviewState_REVIEW_STATE_UNKNOWN
	default:
		return nnxv1.ReviewState_REVIEW_STATE_UNSPECIFIED
	}
}

func protoMergeability(value domain.Mergeability) nnxv1.Mergeability {
	switch value {
	case domain.MergeabilityMergeable:
		return nnxv1.Mergeability_MERGEABILITY_MERGEABLE
	case domain.MergeabilityConflicting:
		return nnxv1.Mergeability_MERGEABILITY_CONFLICTING
	case domain.MergeabilityUnknown:
		return nnxv1.Mergeability_MERGEABILITY_UNKNOWN
	default:
		return nnxv1.Mergeability_MERGEABILITY_UNSPECIFIED
	}
}

func protoCheckState(value domain.CheckState) nnxv1.CheckState {
	switch value {
	case domain.CheckStateUnknown:
		return nnxv1.CheckState_CHECK_STATE_UNKNOWN
	case domain.CheckStateNone:
		return nnxv1.CheckState_CHECK_STATE_NONE
	case domain.CheckStatePending:
		return nnxv1.CheckState_CHECK_STATE_PENDING
	case domain.CheckStateSuccess:
		return nnxv1.CheckState_CHECK_STATE_SUCCESS
	case domain.CheckStateFailure:
		return nnxv1.CheckState_CHECK_STATE_FAILURE
	default:
		return nnxv1.CheckState_CHECK_STATE_UNSPECIFIED
	}
}

func protoPullRequestDisplayState(value domain.PullRequestDisplayState) nnxv1.PullRequestDisplayState {
	const (
		changesRequestedState = nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CHANGES_REQUESTED
		reviewWaitingState    = nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_REVIEW_WAITING
	)
	states := map[domain.PullRequestDisplayState]nnxv1.PullRequestDisplayState{
		domain.PullRequestDisplayStateMerged:           nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_MERGED,
		domain.PullRequestDisplayStateClosed:           nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CLOSED,
		domain.PullRequestDisplayStateDraft:            nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_DRAFT,
		domain.PullRequestDisplayStateConflict:         nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CONFLICT,
		domain.PullRequestDisplayStateChangesRequested: changesRequestedState,
		domain.PullRequestDisplayStateApproved:         nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_APPROVED,
		domain.PullRequestDisplayStateReviewWaiting:    reviewWaitingState,
		domain.PullRequestDisplayStateOpen:             nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_OPEN,
		domain.PullRequestDisplayStateUnknown:          nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_UNKNOWN,
	}
	if state, ok := states[value]; ok {
		return state
	}
	return nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_UNSPECIFIED
}

func protoDocumentKind(value domain.DocumentKind) nnxv1.DocumentKind {
	switch value {
	case domain.DocumentKindURL:
		return nnxv1.DocumentKind_DOCUMENT_KIND_URL
	case domain.DocumentKindLocalFile:
		return nnxv1.DocumentKind_DOCUMENT_KIND_LOCAL_FILE
	case domain.DocumentKindMarkdown:
		return nnxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN
	default:
		return nnxv1.DocumentKind_DOCUMENT_KIND_UNSPECIFIED
	}
}

func protoAddDocumentSource(value *nnxv1.AddDocumentRequest) domain.Document {
	switch value.GetSource().(type) {
	case *nnxv1.AddDocumentRequest_Url:
		return domain.Document{Kind: domain.DocumentKindURL, Locator: value.GetUrl()}
	case *nnxv1.AddDocumentRequest_LocalFile:
		return domain.Document{Kind: domain.DocumentKindLocalFile, Locator: value.GetLocalFile()}
	case *nnxv1.AddDocumentRequest_Markdown:
		return domain.Document{Kind: domain.DocumentKindMarkdown, Content: value.GetMarkdown()}
	default:
		return domain.Document{}
	}
}

func protoUpdateDocumentSource(value *nnxv1.UpdateDocumentRequest) domain.Document {
	switch value.GetSource().(type) {
	case *nnxv1.UpdateDocumentRequest_Url:
		return domain.Document{Kind: domain.DocumentKindURL, Locator: value.GetUrl()}
	case *nnxv1.UpdateDocumentRequest_LocalFile:
		return domain.Document{Kind: domain.DocumentKindLocalFile, Locator: value.GetLocalFile()}
	case *nnxv1.UpdateDocumentRequest_Markdown:
		return domain.Document{Kind: domain.DocumentKindMarkdown, Content: value.GetMarkdown()}
	default:
		return domain.Document{}
	}
}

// protoTaskBlockLabels はドメインの並び順をそのまま保つ。未知の値は落とす。
func protoTaskBlockLabels(values []domain.TaskBlockLabel) []nnxv1.TaskBlockLabel {
	labels := map[domain.TaskBlockLabel]nnxv1.TaskBlockLabel{
		domain.TaskBlockLabelDependencyUnresolved: nnxv1.TaskBlockLabel_TASK_BLOCK_LABEL_DEPENDENCY_UNRESOLVED,
		domain.TaskBlockLabelConflict:             nnxv1.TaskBlockLabel_TASK_BLOCK_LABEL_CONFLICT,
		domain.TaskBlockLabelChangesRequested:     nnxv1.TaskBlockLabel_TASK_BLOCK_LABEL_CHANGES_REQUESTED,
		domain.TaskBlockLabelCIFailed:             nnxv1.TaskBlockLabel_TASK_BLOCK_LABEL_CI_FAILED,
	}
	result := make([]nnxv1.TaskBlockLabel, 0, len(values))
	for _, value := range values {
		if label, ok := labels[value]; ok {
			result = append(result, label)
		}
	}
	return result
}

func protoBlockedReason(task domain.Task) *nnxv1.BlockedReason {
	code := nnxv1.BlockedReasonCode_BLOCKED_REASON_CODE_UNSPECIFIED
	switch task.BlockedCode {
	case domain.BlockedReasonCodeDependencyDataIncomplete:
		code = nnxv1.BlockedReasonCode_BLOCKED_REASON_CODE_DEPENDENCY_DATA_INCOMPLETE
	case domain.BlockedReasonCodeWaitingForBlocker:
		code = nnxv1.BlockedReasonCode_BLOCKED_REASON_CODE_WAITING_FOR_BLOCKER
	}
	if code == nnxv1.BlockedReasonCode_BLOCKED_REASON_CODE_UNSPECIFIED {
		return nil
	}
	return &nnxv1.BlockedReason{Code: code, BlockerTaskId: task.BlockerTaskID}
}

func protoDomainErrorCode(value domain.DomainErrorCode) nnxv1.DomainErrorCode {
	switch value {
	case domain.DomainErrorCodeCrossFeatureDependency:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CROSS_FEATURE_DEPENDENCY
	case domain.DomainErrorCodeCycle:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CYCLE
	case domain.DomainErrorCodeDuplicateDependency:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_DEPENDENCY
	case domain.DomainErrorCodeDuplicatePullRequest:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_PULL_REQUEST
	case domain.DomainErrorCodeDocumentReadFailed:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_READ_FAILED
	case domain.DomainErrorCodeDocumentTooLarge:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_TOO_LARGE
	case domain.DomainErrorCodeDocumentNotText:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_NOT_TEXT
	case domain.DomainErrorCodeDuplicateImplementationPlan:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_IMPLEMENTATION_PLAN
	case domain.DomainErrorCodeInvalidImplementationPlan:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_IMPLEMENTATION_PLAN
	case domain.DomainErrorCodeImplementationPlanTooLarge:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_IMPLEMENTATION_PLAN_TOO_LARGE
	case domain.DomainErrorCodeInvalidConfig:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_CONFIG
	case domain.DomainErrorCodeArchivedReadOnly:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_ARCHIVED_READ_ONLY
	case domain.DomainErrorCodeGitHubAuth:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_GITHUB_AUTH
	case domain.DomainErrorCodeInvalidDatabase:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DATABASE
	case domain.DomainErrorCodeInvalidDocument:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT
	case domain.DomainErrorCodeInvalidDocumentKind:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_KIND
	case domain.DomainErrorCodeInvalidDocumentURL:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_URL
	case domain.DomainErrorCodeInvalidParent:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PARENT
	case domain.DomainErrorCodeInvalidPullRequestURL:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PULL_REQUEST_URL
	case domain.DomainErrorCodeInvalidStatus:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS
	case domain.DomainErrorCodeInvalidTitle:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_TITLE
	case domain.DomainErrorCodeInvalidPromptTemplate:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PROMPT_TEMPLATE
	case domain.DomainErrorCodeInvalidTaskLabel:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_TASK_LABEL
	case domain.DomainErrorCodeNotFound:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_NOT_FOUND
	case domain.DomainErrorCodeReferencesExist:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_REFERENCES_EXIST
	case domain.DomainErrorCodeUpdateUnavailable:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UPDATE_UNAVAILABLE
	case domain.DomainErrorCodeUpdateCheckFailed:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UPDATE_CHECK_FAILED
	case domain.DomainErrorCodeUpdateFailed:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UPDATE_FAILED
	// daemon 系と address_in_use は CLI だけが返す。RPC には常駐の操作がないので、
	// proto の enum には持たせず内部エラー相当として扱う。
	case domain.DomainErrorCodeInternal,
		domain.DomainErrorCodeAddressInUse,
		domain.DomainErrorCodeDaemonUnsupported,
		domain.DomainErrorCodeDaemonNotInstalled,
		domain.DomainErrorCodeDaemonNotRunning,
		domain.DomainErrorCodeDaemonFailed:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UNSPECIFIED
	default:
		return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UNSPECIFIED
	}
}

// optionalValue は proto3 の optional フィールドの有無をサービス層向けに保つ。
// nil ポインタはフィールドの省略を、ゼロ値へのポインタはクリア要求を意味する。
func optionalValue[T any](present bool, value T) *T {
	if !present {
		return nil
	}
	return &value
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"
