package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

// GetUpdateStatus は保存済みの確認結果を返す。間引きが切れていれば application 層が
// その場で確認するので、クライアントは頻繁に呼んでよい。
func (h *Handler) GetUpdateStatus(
	ctx context.Context,
	_ *connect.Request[prxv1.GetUpdateStatusRequest],
) (*connect.Response[prxv1.GetUpdateStatusResponse], error) {
	status, err := h.service.GetUpdateStatus(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.GetUpdateStatusResponse{Status: protoUpdateStatus(status)}), nil
}

func (h *Handler) SkipUpdateVersion(
	ctx context.Context,
	req *connect.Request[prxv1.SkipUpdateVersionRequest],
) (*connect.Response[prxv1.SkipUpdateVersionResponse], error) {
	status, err := h.service.SkipUpdateVersion(ctx, req.Msg.GetVersion())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.SkipUpdateVersionResponse{Status: protoUpdateStatus(status)}), nil
}

// ApplyUpdate は配布元のインストーラーを実行する。信頼境界の扱いは
// docs/design/security.md にある。
func (h *Handler) ApplyUpdate(
	ctx context.Context,
	req *connect.Request[prxv1.ApplyUpdateRequest],
) (*connect.Response[prxv1.ApplyUpdateResponse], error) {
	result, err := h.service.ApplyUpdate(ctx, req.Msg.GetVersion())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.ApplyUpdateResponse{
		Version:         result.Version,
		InstalledPath:   result.InstalledPath,
		RestartRequired: result.RestartRequired,
	}), nil
}

func protoUpdateStatus(status domain.UpdateStatus) *prxv1.UpdateStatus {
	result := &prxv1.UpdateStatus{
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

func protoUpdateReleases(releases []domain.ReleaseNote) []*prxv1.UpdateRelease {
	result := make([]*prxv1.UpdateRelease, 0, len(releases))
	for _, release := range releases {
		value := &prxv1.UpdateRelease{Version: release.Version, Body: release.Body, Url: release.URL}
		if release.PublishedAt != nil {
			value.PublishedAt = release.PublishedAt.UTC().Format(time.RFC3339)
		}
		result = append(result, value)
	}
	return result
}

func protoUpdateDisabledReason(reason domain.UpdateDisabledReason) prxv1.UpdateDisabledReason {
	switch reason {
	case domain.UpdateDisabledDevelopmentBuild:
		return prxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_DEVELOPMENT_BUILD
	case domain.UpdateDisabledDemo:
		return prxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_DEMO
	case domain.UpdateEnabled:
		return prxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_UNSPECIFIED
	default:
		return prxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_UNSPECIFIED
	}
}
