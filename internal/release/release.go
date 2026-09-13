// Package release は PRX の配布元である GitHub Release を読む。取得は未認証で行い、
// 同期対象のリポジトリ向けに設定された資格情報を配布元へ送らない。
// 方針は docs/design/updates.md にある。
package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// Repository は配布元。利用者が同期するリポジトリとは無関係なので設定で変えられない。
const Repository = "HappyOnigiri/PRX"

// DefaultBaseURL は未認証で読む Releases API の位置。
const DefaultBaseURL = "https://api.github.com/repos/" + Repository + "/releases"

const (
	// requestPageSize は 1 回の確認で読むリリース件数。表示に使う件数より少し多く読み、
	// draft と prerelease を落としても十分な数が残るようにする。
	requestPageSize = 30
	// maxResponseBytes は本文を含む応答の読み取り上限。
	maxResponseBytes = 4 << 20
	// requestTimeout は 1 回の取得にかける上限。internal/github の慣習に揃える。
	requestTimeout = 30 * time.Second
)

// Provider は配布元のリリース一覧を読む境界。
type Provider interface {
	// Releases は draft と prerelease を除いたリリースを返す。並び順は呼び出し側が決める。
	Releases(ctx context.Context) ([]domain.ReleaseNote, error)
}

// LiveProvider は GitHub Releases API を未認証で読む。
type LiveProvider struct {
	baseURL string
	client  *http.Client
}

// New は既定の配布元を読む provider を返す。
func New() *LiveProvider { return NewWithOptions(DefaultBaseURL, nil) }

// NewWithOptions は取得元と HTTP クライアントを差し替えられる provider を返す。
// テストは httptest のアドレスを渡す。
func NewWithOptions(baseURL string, client *http.Client) *LiveProvider {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	return &LiveProvider{baseURL: baseURL, client: client}
}

// apiRelease は Releases API の応答のうち、更新の案内に必要なフィールド。
type apiRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
}

func (p *LiveProvider) Releases(ctx context.Context) ([]domain.ReleaseNote, error) {
	url := fmt.Sprintf("%s?per_page=%d", p.baseURL, requestPageSize)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build release request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("read releases: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, statusError(response)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read releases: %w", err)
	}
	var decoded []apiRelease
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode releases: %w", err)
	}
	return releaseNotes(decoded), nil
}

// statusError は未認証の呼び出し上限を、権限やその他の失敗と区別する。
// 上限は時間が解決するので、利用者に伝えるべき次の行動が違う。
func statusError(response *http.Response) error {
	status := response.StatusCode
	switch {
	case status == http.StatusTooManyRequests,
		status == http.StatusForbidden && response.Header.Get("X-RateLimit-Remaining") == "0":
		return errors.New(
			"GitHub refused the update check because the unauthenticated rate limit is exhausted; try again later",
		)
	case status == http.StatusNotFound:
		return fmt.Errorf("the release feed for %s is unavailable", Repository)
	default:
		return fmt.Errorf("GitHub returned status %d for the update check", status)
	}
}

func releaseNotes(values []apiRelease) []domain.ReleaseNote {
	result := make([]domain.ReleaseNote, 0, len(values))
	for _, value := range values {
		if value.Draft || value.Prerelease {
			continue
		}
		if domain.CanonicalVersion(value.TagName) == "" {
			continue
		}
		result = append(result, domain.ReleaseNote{
			Version:     value.TagName,
			PublishedAt: parsePublishedAt(value.PublishedAt),
			Body:        value.Body,
			URL:         value.HTMLURL,
		})
	}
	return result
}

func parsePublishedAt(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	utc := parsed.UTC()
	return &utc
}

// StaticProvider は固定のリリース一覧を返す。テストと、実ネットワークへ出てはならない
// 経路のために置く。
type StaticProvider struct {
	releases []domain.ReleaseNote
	err      error
}

// NewStaticProvider は与えた一覧をそのまま返す provider を作る。
func NewStaticProvider(releases []domain.ReleaseNote) *StaticProvider {
	return &StaticProvider{releases: releases}
}

// NewFailingProvider は必ず失敗する provider を作る。確認の失敗経路の検証に使う。
func NewFailingProvider(err error) *StaticProvider { return &StaticProvider{err: err} }

func (p *StaticProvider) Releases(context.Context) ([]domain.ReleaseNote, error) {
	if p.err != nil {
		return nil, p.err
	}
	return append([]domain.ReleaseNote{}, p.releases...), nil
}
