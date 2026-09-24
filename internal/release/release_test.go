package release_test

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
	"github.com/HappyOnigiri/NextNowX/internal/release"
)

// 静的な provider は、実ネットワークへ出てはならない経路のために置く。
func TestStaticProviderReturnsConfiguredValues(t *testing.T) {
	notes := []domain.ReleaseNote{{Version: "v1.0.0"}}
	releases, err := release.NewStaticProvider(notes).Releases(context.Background())
	if err != nil || len(releases) != 1 || releases[0].Version != "v1.0.0" {
		t.Fatalf("releases=%+v err=%v", releases, err)
	}
	failure := errors.New("offline")
	if _, err := release.NewFailingProvider(failure).Releases(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("error=%v", err)
	}
}
