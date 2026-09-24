//go:build noupdate

package app_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/nnx/internal/app"
	"github.com/HappyOnigiri/nnx/internal/config"
	"github.com/HappyOnigiri/nnx/internal/domain"
	"github.com/HappyOnigiri/nnx/internal/store"
)

// 更新機能を外したビルドでは、注入があっても確認へ出ず、理由付きで無効を返す。
type unreachableProvider struct{ calls int }

func (p *unreachableProvider) Releases(context.Context) ([]domain.ReleaseNote, error) {
	p.calls++
	return nil, nil
}

func TestUpdateIsDisabledWhenExcludedFromTheBuild(t *testing.T) {
	app.StubUpdateBuildVersionForTest(t, "0.3.0")
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "update.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	configStore, err := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := configStore.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	provider := &unreachableProvider{}
	service := app.NewWithConfig(database, nil, configStore)
	service.SetUpdateSources(provider, nil)

	status, err := service.GetUpdateStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Enabled || status.ShouldNotify || status.DisabledReason != domain.UpdateDisabledExcludedFromBuild {
		t.Fatalf("status=%+v", status)
	}
	if provider.calls != 0 {
		t.Fatalf("calls=%d", provider.calls)
	}
	if _, err := service.ApplyUpdate(ctx, "v0.4.0"); err == nil {
		t.Fatal("expected apply to fail")
	}
}
