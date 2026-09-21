package rpc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/rpc"
)

// updateService は更新の 3 つだけを差し替える。他の RPC は埋め込んだ interface が担う。
type updateService struct {
	rpc.Service
	status  domain.UpdateStatus
	result  domain.UpdateResult
	skipped string
	applied string
	err     error
}

func (s *updateService) GetUpdateStatus(context.Context) (domain.UpdateStatus, error) {
	if s.err != nil {
		return domain.UpdateStatus{}, s.err
	}
	return s.status, nil
}

func (s *updateService) SkipUpdateVersion(_ context.Context, version string) (domain.UpdateStatus, error) {
	if s.err != nil {
		return domain.UpdateStatus{}, s.err
	}
	s.skipped = version
	s.status.ShouldNotify = false
	s.status.SkippedVersion = version
	return s.status, nil
}

func (s *updateService) ApplyUpdate(_ context.Context, version string) (domain.UpdateResult, error) {
	if s.err != nil {
		return domain.UpdateResult{}, s.err
	}
	s.applied = version
	return s.result, nil
}

func TestUpdateStatusRPCCarriesReleasesAndNotice(t *testing.T) {
	published := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	checked := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	service := &updateService{status: domain.UpdateStatus{
		Enabled:         true,
		CurrentVersion:  "0.3.0",
		UpdateAvailable: true,
		ShouldNotify:    true,
		LatestVersion:   "v0.4.0",
		Releases: []domain.ReleaseNote{
			{Version: "v0.4.0", PublishedAt: &published, Body: "Notes", URL: "https://example.test"},
		},
		LastCheckedAt: &checked,
	}}
	client := newTestClientForService(t, service)
	response, err := client.GetUpdateStatus(
		context.Background(), connect.NewRequest(&prxv1.GetUpdateStatusRequest{}),
	)
	if err != nil {
		t.Fatal(err)
	}
	status := response.Msg.GetStatus()
	if !status.GetEnabled() || !status.GetShouldNotify() || status.GetLatestVersion() != "v0.4.0" {
		t.Fatalf("status=%+v", status)
	}
	if status.GetLastCheckedAt() != checked.Format(time.RFC3339) {
		t.Fatalf("checked=%q", status.GetLastCheckedAt())
	}
	releases := status.GetReleases()
	if len(releases) != 1 || releases[0].GetBody() != "Notes" ||
		releases[0].GetPublishedAt() != published.Format(time.RFC3339) {
		t.Fatalf("releases=%+v", releases)
	}
}

func TestUpdateStatusRPCReportsTheDisabledReason(t *testing.T) {
	for _, test := range []struct {
		name   string
		reason domain.UpdateDisabledReason
		want   prxv1.UpdateDisabledReason
	}{
		{
			name:   "development build",
			reason: domain.UpdateDisabledDevelopmentBuild,
			want:   prxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_DEVELOPMENT_BUILD,
		},
		{
			name:   "demo",
			reason: domain.UpdateDisabledDemo,
			want:   prxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_DEMO,
		},
		{
			name:   "excluded from build",
			reason: domain.UpdateDisabledExcludedFromBuild,
			want:   prxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_EXCLUDED_FROM_BUILD,
		},
		{
			name:   "enabled",
			reason: domain.UpdateEnabled,
			want:   prxv1.UpdateDisabledReason_UPDATE_DISABLED_REASON_UNSPECIFIED,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &updateService{status: domain.UpdateStatus{DisabledReason: test.reason}}
			client := newTestClientForService(t, service)
			response, err := client.GetUpdateStatus(
				context.Background(), connect.NewRequest(&prxv1.GetUpdateStatusRequest{}),
			)
			if err != nil {
				t.Fatal(err)
			}
			if got := response.Msg.GetStatus().GetDisabledReason(); got != test.want {
				t.Fatalf("reason=%v, want %v", got, test.want)
			}
		})
	}
}

func TestSkipUpdateVersionRPCSilencesTheNotice(t *testing.T) {
	service := &updateService{status: domain.UpdateStatus{Enabled: true, ShouldNotify: true}}
	client := newTestClientForService(t, service)
	response, err := client.SkipUpdateVersion(
		context.Background(), connect.NewRequest(&prxv1.SkipUpdateVersionRequest{Version: "v0.4.0"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if service.skipped != "v0.4.0" || response.Msg.GetStatus().GetShouldNotify() {
		t.Fatalf("skipped=%q status=%+v", service.skipped, response.Msg.GetStatus())
	}
}

func TestApplyUpdateRPCReportsTheInstalledBinary(t *testing.T) {
	service := &updateService{result: domain.UpdateResult{
		Version: "v0.4.0", InstalledPath: "/home/example/.local/bin/prx", RestartRequired: true,
	}}
	client := newTestClientForService(t, service)
	response, err := client.ApplyUpdate(
		context.Background(), connect.NewRequest(&prxv1.ApplyUpdateRequest{Version: "v0.4.0"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if service.applied != "v0.4.0" || !response.Msg.GetRestartRequired() {
		t.Fatalf("applied=%q response=%+v", service.applied, response.Msg)
	}
	if response.Msg.GetInstalledPath() != "/home/example/.local/bin/prx" {
		t.Fatalf("path=%q", response.Msg.GetInstalledPath())
	}
}

// 無効なビルドでの依頼は、クライアントが区別できるコードで返す。
func TestUpdateRPCsMapDomainErrorsToDetailCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want prxv1.DomainErrorCode
		code connect.Code
	}{
		{
			name: "disabled",
			err:  domain.NewError(domain.DomainErrorCodeUpdateUnavailable, "updates are disabled"),
			want: prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UPDATE_UNAVAILABLE,
			code: connect.CodeFailedPrecondition,
		},
		{
			name: "check failed",
			err:  domain.NewError(domain.DomainErrorCodeUpdateCheckFailed, "the feed is unreachable"),
			want: prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UPDATE_CHECK_FAILED,
			code: connect.CodeUnavailable,
		},
		{
			name: "apply failed",
			err:  domain.NewError(domain.DomainErrorCodeUpdateFailed, "checksum verification failed"),
			want: prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UPDATE_FAILED,
			code: connect.CodeInternal,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClientForService(t, &updateService{err: test.err})
			_, err := client.GetUpdateStatus(
				context.Background(), connect.NewRequest(&prxv1.GetUpdateStatusRequest{}),
			)
			if err == nil {
				t.Fatal("expected a failure")
			}
			if got := errorDetailCode(t, err); got != test.want {
				t.Fatalf("detail=%v, want %v", got, test.want)
			}
			var connectErr *connect.Error
			if !errors.As(err, &connectErr) || connectErr.Code() != test.code {
				t.Fatalf("code=%v, want %v", err, test.code)
			}
		})
	}
}
