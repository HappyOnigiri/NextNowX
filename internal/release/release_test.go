package release_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/release"
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) *release.LiveProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return release.NewWithOptions(server.URL, server.Client())
}

// 確認はタグ・本文・公開時刻を読み、案内の材料にならない draft と prerelease、
// および semver でないタグを落とす。
func TestReleasesKeepsPublishedSemverTagsOnly(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("per_page"); got == "" {
			t.Errorf("per_page=%q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept=%q", got)
		}
		if _, ok := r.Header["Authorization"]; ok {
			t.Error("the update check sent an Authorization header")
		}
		_, _ = w.Write([]byte(`[
			{"tag_name":"v0.4.0","body":"Notes","html_url":"https://example.test/v0.4.0",
			 "published_at":"2026-09-01T10:00:00Z"},
			{"tag_name":"v0.5.0","draft":true},
			{"tag_name":"v0.6.0","prerelease":true},
			{"tag_name":"nightly"}
		]`))
	})
	releases, err := provider.Releases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 {
		t.Fatalf("releases=%+v", releases)
	}
	got := releases[0]
	if got.Version != "v0.4.0" || got.Body != "Notes" || got.URL != "https://example.test/v0.4.0" {
		t.Fatalf("release=%+v", got)
	}
	if got.PublishedAt == nil || got.PublishedAt.Format("2006-01-02") != "2026-09-01" {
		t.Fatalf("published=%v", got.PublishedAt)
	}
}

// 未認証の呼び出し上限は時間が解決するので、権限の失敗とは別に伝える必要がある。
func TestReleasesClassifiesFailures(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		headers map[string]string
		want    release.ErrorClass
	}{
		{
			name:    "exhausted rate limit",
			status:  http.StatusForbidden,
			headers: map[string]string{"X-RateLimit-Remaining": "0"},
			want:    release.ErrorClassRateLimit,
		},
		{name: "too many requests", status: http.StatusTooManyRequests, want: release.ErrorClassRateLimit},
		{name: "missing feed", status: http.StatusNotFound, want: release.ErrorClassNotFound},
		{name: "server failure", status: http.StatusBadGateway, want: release.ErrorClassTransient},
		{name: "forbidden", status: http.StatusForbidden, want: release.ErrorClassOther},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := newTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
				for key, value := range test.headers {
					w.Header().Set(key, value)
				}
				w.WriteHeader(test.status)
			})
			_, err := provider.Releases(context.Background())
			if err == nil {
				t.Fatal("expected a failure")
			}
			if got := release.ClassOf(err); got != test.want {
				t.Fatalf("class=%q, want %q: %v", got, test.want, err)
			}
		})
	}
}

func TestReleasesReportsUnreadableResponses(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})
	_, err := provider.Releases(context.Background())
	if err == nil || release.ClassOf(err) != release.ErrorClassOther {
		t.Fatalf("error=%v class=%q", err, release.ClassOf(err))
	}
}

// 到達できないホストは時間をおけば直りうるので transient に寄せる。
func TestReleasesTreatsTransportFailuresAsTransient(t *testing.T) {
	provider := release.NewWithOptions("http://127.0.0.1:0/releases", nil)
	_, err := provider.Releases(context.Background())
	if err == nil || release.ClassOf(err) != release.ErrorClassTransient {
		t.Fatalf("error=%v class=%q", err, release.ClassOf(err))
	}
}

func TestClassOfUnclassifiedErrorIsOther(t *testing.T) {
	if got := release.ClassOf(errors.New("boom")); got != release.ErrorClassOther {
		t.Fatalf("class=%q", got)
	}
}

func TestNewUsesTheFixedDistributionFeed(t *testing.T) {
	if release.DefaultBaseURL != "https://api.github.com/repos/"+release.Repository+"/releases" {
		t.Fatalf("base url=%q", release.DefaultBaseURL)
	}
	if release.New() == nil {
		t.Fatal("provider is nil")
	}
}

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
