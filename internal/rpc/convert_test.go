package rpc

import (
	"errors"
	"slices"
	"testing"

	"connectrpc.com/connect"

	nnxv1 "github.com/HappyOnigiri/nnx/gen/nnx/v1"
	"github.com/HappyOnigiri/nnx/internal/domain"
)

func TestProtoFeatureStatusMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.FeatureStatus
		want  nnxv1.FeatureStatus
	}{
		{"auto", domain.FeatureStatusAuto, nnxv1.FeatureStatus_FEATURE_STATUS_AUTO},
		{"active", domain.FeatureStatusActive, nnxv1.FeatureStatus_FEATURE_STATUS_ACTIVE},
		{"paused", domain.FeatureStatusPaused, nnxv1.FeatureStatus_FEATURE_STATUS_PAUSED},
		{"completed", domain.FeatureStatusCompleted, nnxv1.FeatureStatus_FEATURE_STATUS_COMPLETED},
		{"cancelled", domain.FeatureStatusCancelled, nnxv1.FeatureStatus_FEATURE_STATUS_CANCELLED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoFeatureStatus(test.value); got != test.want {
				t.Fatalf("protoFeatureStatus(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

// 導出された status と完了数はまとめてサーバーから出ていく。クライアントが
// タスクを数え直さずに feature へラベルを付け、残りを説明できるようにするため。
func TestProtoFeatureCarriesTheDerivedStatusAndFinishedCount(t *testing.T) {
	got := protoFeature(domain.Feature{
		ID:            "F-1",
		Status:        domain.FeatureStatusAuto,
		DisplayStatus: domain.FeatureStatusCompleted,
		TaskCount:     3,
		FinishedCount: 3,
	})
	if got.GetStatus() != nnxv1.FeatureStatus_FEATURE_STATUS_AUTO ||
		got.GetDisplayStatus() != nnxv1.FeatureStatus_FEATURE_STATUS_COMPLETED ||
		got.GetFinishedCount() != 3 {
		t.Fatalf("converted feature=%+v", got)
	}
}

func TestDomainFeatureStatusAcceptsAutoAndRejectsUnknownValues(t *testing.T) {
	auto := nnxv1.FeatureStatus_FEATURE_STATUS_AUTO
	got, err := domainFeatureStatus(&auto)
	if err != nil || got == nil || *got != domain.FeatureStatusAuto {
		t.Fatalf("domainFeatureStatus(auto)=%v err=%v", got, err)
	}
	unknown := nnxv1.FeatureStatus(999)
	if _, err := domainFeatureStatus(&unknown); domain.ErrorCode(err) != domain.DomainErrorCodeInvalidStatus {
		t.Fatalf("domainFeatureStatus(unknown) err=%v", err)
	}
}

func TestProtoTaskStatusMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.TaskStatus
		want  nnxv1.TaskStatus
	}{
		{"not started", domain.TaskStatusNotStarted, nnxv1.TaskStatus_TASK_STATUS_NOT_STARTED},
		{"designing", domain.TaskStatusDesigning, nnxv1.TaskStatus_TASK_STATUS_DESIGNING},
		{"in progress", domain.TaskStatusInProgress, nnxv1.TaskStatus_TASK_STATUS_IN_PROGRESS},
		{"completed", domain.TaskStatusCompleted, nnxv1.TaskStatus_TASK_STATUS_COMPLETED},
		{"closed", domain.TaskStatusClosed, nnxv1.TaskStatus_TASK_STATUS_CLOSED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoTaskStatus(test.value); got != test.want {
				t.Fatalf("protoTaskStatus(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoTaskDisplayStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.TaskDisplayState
		want  nnxv1.TaskDisplayState
	}{
		{"not started", domain.TaskDisplayStateNotStarted, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_NOT_STARTED},
		{"designing", domain.TaskDisplayStateDesigning, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_DESIGNING},
		{"designed", domain.TaskDisplayStateDesigned, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_DESIGNED},
		{"in progress", domain.TaskDisplayStateInProgress, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_IN_PROGRESS},
		{"completed", domain.TaskDisplayStateCompleted, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_COMPLETED},
		{"closed", domain.TaskDisplayStateClosed, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_CLOSED},
		{"merged", domain.TaskDisplayStateMerged, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_MERGED},
		{
			"implemented",
			domain.TaskDisplayStateImplemented,
			nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_IMPLEMENTED,
		},
		{"in review", domain.TaskDisplayStateInReview, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_IN_REVIEW},
		{"approved", domain.TaskDisplayStateApproved, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_APPROVED},
		{"unknown", domain.TaskDisplayStateUnknown, nnxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoTaskDisplayState(test.value); got != test.want {
				t.Fatalf("protoTaskDisplayState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoPullRequestStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.PullRequestState
		want  nnxv1.PullRequestState
	}{
		{"open", domain.PullRequestStateOpen, nnxv1.PullRequestState_PULL_REQUEST_STATE_OPEN},
		{"closed", domain.PullRequestStateClosed, nnxv1.PullRequestState_PULL_REQUEST_STATE_CLOSED},
		{"merged", domain.PullRequestStateMerged, nnxv1.PullRequestState_PULL_REQUEST_STATE_MERGED},
		{"unknown", domain.PullRequestStateUnknown, nnxv1.PullRequestState_PULL_REQUEST_STATE_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoPullRequestState(test.value); got != test.want {
				t.Fatalf("protoPullRequestState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoReviewStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.ReviewState
		want  nnxv1.ReviewState
	}{
		{"none", domain.ReviewStateNone, nnxv1.ReviewState_REVIEW_STATE_NONE},
		{"required", domain.ReviewStateRequired, nnxv1.ReviewState_REVIEW_STATE_REQUIRED},
		{"approved", domain.ReviewStateApproved, nnxv1.ReviewState_REVIEW_STATE_APPROVED},
		{"changes requested", domain.ReviewStateChangesRequested, nnxv1.ReviewState_REVIEW_STATE_CHANGES_REQUESTED},
		{"unknown", domain.ReviewStateUnknown, nnxv1.ReviewState_REVIEW_STATE_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoReviewState(test.value); got != test.want {
				t.Fatalf("protoReviewState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoMergeabilityMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.Mergeability
		want  nnxv1.Mergeability
	}{
		{"mergeable", domain.MergeabilityMergeable, nnxv1.Mergeability_MERGEABILITY_MERGEABLE},
		{"conflicting", domain.MergeabilityConflicting, nnxv1.Mergeability_MERGEABILITY_CONFLICTING},
		{"unknown", domain.MergeabilityUnknown, nnxv1.Mergeability_MERGEABILITY_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoMergeability(test.value); got != test.want {
				t.Fatalf("protoMergeability(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoCheckStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.CheckState
		want  nnxv1.CheckState
	}{
		{"unknown", domain.CheckStateUnknown, nnxv1.CheckState_CHECK_STATE_UNKNOWN},
		{"none", domain.CheckStateNone, nnxv1.CheckState_CHECK_STATE_NONE},
		{"pending", domain.CheckStatePending, nnxv1.CheckState_CHECK_STATE_PENDING},
		{"success", domain.CheckStateSuccess, nnxv1.CheckState_CHECK_STATE_SUCCESS},
		{"failure", domain.CheckStateFailure, nnxv1.CheckState_CHECK_STATE_FAILURE},
		{"unset", domain.CheckState(""), nnxv1.CheckState_CHECK_STATE_UNSPECIFIED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoCheckState(test.value); got != test.want {
				t.Fatalf("protoCheckState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

// ブロックラベルはドメインの並び順のまま送り、未知の値は落とす。
func TestProtoTaskBlockLabelsKeepsDomainOrder(t *testing.T) {
	got := protoTaskBlockLabels([]domain.TaskBlockLabel{
		domain.TaskBlockLabelDependencyUnresolved, domain.TaskBlockLabelConflict,
		domain.TaskBlockLabelChangesRequested, domain.TaskBlockLabelCIFailed,
		domain.TaskBlockLabel("mystery"),
	})
	want := []nnxv1.TaskBlockLabel{
		nnxv1.TaskBlockLabel_TASK_BLOCK_LABEL_DEPENDENCY_UNRESOLVED,
		nnxv1.TaskBlockLabel_TASK_BLOCK_LABEL_CONFLICT,
		nnxv1.TaskBlockLabel_TASK_BLOCK_LABEL_CHANGES_REQUESTED,
		nnxv1.TaskBlockLabel_TASK_BLOCK_LABEL_CI_FAILED,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("block labels=%v want %v", got, want)
	}
}

func TestProtoPullRequestDisplayStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.PullRequestDisplayState
		want  nnxv1.PullRequestDisplayState
	}{
		{
			"merged",
			domain.PullRequestDisplayStateMerged,
			nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_MERGED,
		},
		{
			"closed",
			domain.PullRequestDisplayStateClosed,
			nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CLOSED,
		},
		{"draft", domain.PullRequestDisplayStateDraft, nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_DRAFT},
		{
			"conflict",
			domain.PullRequestDisplayStateConflict,
			nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CONFLICT,
		},
		{
			"changes requested",
			domain.PullRequestDisplayStateChangesRequested,
			nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CHANGES_REQUESTED,
		},
		{
			"approved",
			domain.PullRequestDisplayStateApproved,
			nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_APPROVED,
		},
		{
			"review waiting",
			domain.PullRequestDisplayStateReviewWaiting,
			nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_REVIEW_WAITING,
		},
		{"open", domain.PullRequestDisplayStateOpen, nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_OPEN},
		{
			"unknown",
			domain.PullRequestDisplayStateUnknown,
			nnxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_UNKNOWN,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoPullRequestDisplayState(test.value); got != test.want {
				t.Fatalf("protoPullRequestDisplayState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoDocumentKindMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.DocumentKind
		want  nnxv1.DocumentKind
	}{
		{"URL", domain.DocumentKindURL, nnxv1.DocumentKind_DOCUMENT_KIND_URL},
		{"local file", domain.DocumentKindLocalFile, nnxv1.DocumentKind_DOCUMENT_KIND_LOCAL_FILE},
		{"Markdown", domain.DocumentKindMarkdown, nnxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoDocumentKind(test.value); got != test.want {
				t.Fatalf("protoDocumentKind(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoBlockedReasonMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.BlockedReasonCode
		want  nnxv1.BlockedReasonCode
	}{
		{
			"dependency data incomplete",
			domain.BlockedReasonCodeDependencyDataIncomplete,
			nnxv1.BlockedReasonCode_BLOCKED_REASON_CODE_DEPENDENCY_DATA_INCOMPLETE,
		},
		{
			"waiting for blocker",
			domain.BlockedReasonCodeWaitingForBlocker,
			nnxv1.BlockedReasonCode_BLOCKED_REASON_CODE_WAITING_FOR_BLOCKER,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := protoBlockedReason(domain.Task{BlockedCode: test.value, BlockerTaskID: "blocker"})
			if got == nil || got.GetCode() != test.want || got.GetBlockerTaskId() != "blocker" {
				t.Fatalf("protoBlockedReason(%q)=%+v, want code %s", test.value, got, test.want)
			}
		})
	}
}

func TestRPCErrorDetailsMapEveryKnownDomainErrorCode(t *testing.T) {
	tests := []struct {
		name  string
		value domain.DomainErrorCode
		want  nnxv1.DomainErrorCode
	}{
		{
			"cross feature dependency",
			domain.DomainErrorCodeCrossFeatureDependency,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CROSS_FEATURE_DEPENDENCY,
		},
		{"cycle", domain.DomainErrorCodeCycle, nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CYCLE},
		{
			"duplicate dependency",
			domain.DomainErrorCodeDuplicateDependency,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_DEPENDENCY,
		},
		{
			"duplicate pull request",
			domain.DomainErrorCodeDuplicatePullRequest,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_PULL_REQUEST,
		},
		{"GitHub auth", domain.DomainErrorCodeGitHubAuth, nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_GITHUB_AUTH},
		{
			"invalid database",
			domain.DomainErrorCodeInvalidDatabase,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DATABASE,
		},
		{
			"invalid document",
			domain.DomainErrorCodeInvalidDocument,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT,
		},
		{
			"invalid document kind",
			domain.DomainErrorCodeInvalidDocumentKind,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_KIND,
		},
		{"invalid parent", domain.DomainErrorCodeInvalidParent, nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PARENT},
		{
			"invalid pull request URL",
			domain.DomainErrorCodeInvalidPullRequestURL,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PULL_REQUEST_URL,
		},
		{"invalid status", domain.DomainErrorCodeInvalidStatus, nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS},
		{"invalid title", domain.DomainErrorCodeInvalidTitle, nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_TITLE},
		{"not found", domain.DomainErrorCodeNotFound, nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_NOT_FOUND},
		{
			"references exist",
			domain.DomainErrorCodeReferencesExist,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_REFERENCES_EXIST,
		},
		{
			"invalid document URL",
			domain.DomainErrorCodeInvalidDocumentURL,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_URL,
		},
		{
			"document read failed",
			domain.DomainErrorCodeDocumentReadFailed,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_READ_FAILED,
		},
		{
			"document too large",
			domain.DomainErrorCodeDocumentTooLarge,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_TOO_LARGE,
		},
		{
			"invalid implementation plan",
			domain.DomainErrorCodeInvalidImplementationPlan,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_IMPLEMENTATION_PLAN,
		},
		{
			"implementation plan too large",
			domain.DomainErrorCodeImplementationPlanTooLarge,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_IMPLEMENTATION_PLAN_TOO_LARGE,
		},
		{
			"document not text",
			domain.DomainErrorCodeDocumentNotText,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_NOT_TEXT,
		},
		{
			"duplicate implementation plan",
			domain.DomainErrorCodeDuplicateImplementationPlan,
			nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_IMPLEMENTATION_PLAN,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rpcError(domain.NewError(test.value, "domain error"))
			if got := errorDetailCode(t, err); got != test.want {
				t.Fatalf("rpcError(%q) detail code=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func errorDetailCode(t *testing.T, err error) nnxv1.DomainErrorCode {
	t.Helper()
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("error type=%T value=%v", err, err)
	}
	for _, detail := range connectErr.Details() {
		value, detailErr := detail.Value()
		if detailErr != nil {
			t.Fatal(detailErr)
		}
		if errorDetail, ok := value.(*nnxv1.ErrorDetail); ok {
			return errorDetail.GetCode()
		}
	}
	return nnxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UNSPECIFIED
}
