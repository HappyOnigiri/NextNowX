package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"

	nnxv1 "github.com/HappyOnigiri/nnx/gen/nnx/v1"
	"github.com/HappyOnigiri/nnx/internal/domain"
)

// GetUpdateStatus は保存済みの確認結果を返す。間引きが切れていれば application 層が
// その場で確認するので、クライアントは頻繁に呼んでよい。
func (h *Handler) GetUpdateStatus(
	ctx context.Context,
	_ *connect.Request[nnxv1.GetUpdateStatusRequest],
) (*connect.Response[nnxv1.GetUpdateStatusResponse], error) {
	status, err := h.service.GetUpdateStatus(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&nnxv1.GetUpdateStatusResponse{Status: protoUpdateStatus(status)}), nil
}

func (h *Handler) SkipUpdateVersion(
	ctx context.Context,
	req *connect.Request[nnxv1.SkipUpdateVersionRequest],
) (*connect.Response[nnxv1.SkipUpdateVersionResponse], error) {
	status, err := h.service.SkipUpdateVersion(ctx, req.Msg.GetVersion())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&nnxv1.SkipUpdateVersionResponse{Status: protoUpdateStatus(status)}), nil
}

// ApplyUpdate は配布元のインストーラーを実行する。信頼境界の扱いは
// docs/design/security.md にある。
func (h *Handler) ApplyUpdate(
	ctx context.Context,
	req *connect.Request[nnxv1.ApplyUpdateRequest],
) (*connect.Response[nnxv1.ApplyUpdateResponse], error) {
	result, err := h.service.ApplyUpdate(ctx, req.Msg.GetVersion())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&nnxv1.ApplyUpdateResponse{
		Version:         result.Version,
		InstalledPath:   result.InstalledPath,
		RestartRequired: result.RestartRequired,
	}), nil
}

func protoUpdateStatus(status domain.UpdateStatus) *nnxv1.UpdateStatus {
	result := &nnxv1.UpdateStatus{
		Enabled:         status.Enabled,
		DisabledReason:  protoUpdateDisabledReason(status.DisabledReason),
		CurrentVersion:  status.CurrentVersion,
		UpdateAvailable: status.UpdateAvailable,
		ShouldNotify:    status.ShouldNotify,
		LatestVersion:   status.LatestVersion,
		Releases:        protoUpdateReleases(status.Releases),
		LastError:       status.LastError,
		SkippedVersion:  status.SkippedVersion,
	}
	if status.LastCheckedAt != nil {
		checked := status.LastCheckedAt.UTC().Format(time.RFC3339)
		result.LastCheckedAt = &checked
	}
	return result
}

func protoUpdateReleases(releases []domain.ReleaseNote) []*nnxv1.UpdateRelease {
	result := make([]*nnxv1.UpdateRelease, 0, len(releases))
	for _, release := range releases {
		value := &nnxv1.UpdateRelease{Version: release.Version, Body: release.Body, Url: release.URL}
		if release.PublishedAt != nil {
			value.PublishedAt = release.PublishedAt.UTC().Format(time.RFC3339)
		}
		result = append(result, value)
	}
	return result
}

func protoUpdateDisabledReason(reason domain.UpdateDisabledReason) nnxv1.UpdateDisabledReason {
	switch reason {
	case domain.UpdateDisabledDevelopmentBuild:
		return nnxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_DEVELOPMENT_BUILD
	case domain.UpdateDisabledDemo:
		return nnxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_DEMO
	case domain.UpdateDisabledExcludedFromBuild:
		return nnxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_EXCLUDED_FROM_BUILD
	case domain.UpdateEnabled:
		return nnxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_UNSPECIFIED
	default:
		return nnxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_UNSPECIFIED
	}
}
