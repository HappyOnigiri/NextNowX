package prompt_test

import (
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/prompt"
)

func TestResolveLanguagePrefersTheSettingOverTheEnvironment(t *testing.T) {
	japaneseEnvironment := map[string]string{"LANG": "ja_JP.UTF-8"}
	for name, test := range map[string]struct {
		setting     string
		environment map[string]string
		want        prompt.Language
	}{
		"explicit english over a japanese locale": {
			setting: "en", environment: japaneseEnvironment, want: prompt.LanguageEnglish,
		},
		"explicit japanese without any locale": {
			setting: "ja", environment: nil, want: prompt.LanguageJapanese,
		},
		"auto reads the locale": {
			setting: "auto", environment: japaneseEnvironment, want: prompt.LanguageJapanese,
		},
		"a missing key behaves like auto": {
			setting: "", environment: japaneseEnvironment, want: prompt.LanguageJapanese,
		},
		// LC_ALL は LANG を覆う。POSIX の優先順に従わないと、ロケールを一時的に
		// 切り替えた実行でプロンプトの言語だけが元のままになる。
		"LC_ALL wins over LANG": {
			setting:     "auto",
			environment: map[string]string{"LC_ALL": "en_US.UTF-8", "LANG": "ja_JP.UTF-8"},
			want:        prompt.LanguageEnglish,
		},
		"LC_MESSAGES wins over LANG": {
			setting:     "auto",
			environment: map[string]string{"LC_MESSAGES": "ja_JP.UTF-8", "LANG": "en_US.UTF-8"},
			want:        prompt.LanguageJapanese,
		},
		// LaunchAgent 経由の起動ではロケール変数が渡らないことがある。
		"an empty environment falls back to english": {
			setting: "auto", environment: nil, want: prompt.LanguageEnglish,
		},
		"an untranslatable locale falls back to english": {
			setting:     "auto",
			environment: map[string]string{"LANG": "C.UTF-8"},
			want:        prompt.LanguageEnglish,
		},
		"an unsupported language falls back to english": {
			setting:     "auto",
			environment: map[string]string{"LANG": "fr_FR.UTF-8"},
			want:        prompt.LanguageEnglish,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := prompt.ResolveLanguageWith(test.setting, func(key string) string {
				return test.environment[key]
			})
			if got != test.want {
				t.Fatalf("language=%q, want %q", got, test.want)
			}
		})
	}
}

func TestParseLanguageRejectsWhatIsNotABuiltInLanguage(t *testing.T) {
	if language, ok := prompt.ParseLanguage(" JA "); !ok || language != prompt.LanguageJapanese {
		t.Fatalf("language=%q ok=%v", language, ok)
	}
	for _, value := range []string{"auto", "", "fr", "japanese"} {
		if _, ok := prompt.ParseLanguage(value); ok {
			t.Fatalf("ParseLanguage accepted %q", value)
		}
	}
}

// 実効言語は組み込みテンプレートの言語そのものである。設定が ja のとき、
// レンダリングされたプロンプトは日本語で出なければならない。
func TestDefaultTemplatesFollowTheEffectiveLanguage(t *testing.T) {
	_, body, err := prompt.Render(designTask(), "", prompt.Templates{}, prompt.LanguageJapanese)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "設計する") || strings.Contains(body, "Design PRX task") {
		t.Fatalf("the japanese design prompt is not in japanese: %q", body)
	}
	if !strings.Contains(body, "T-7") || strings.Contains(body, "{{") {
		t.Fatalf("the japanese design prompt did not expand its placeholders: %q", body)
	}
	_, batch, err := prompt.RenderBatch(
		"F-3", batchTasks(), prompt.KindBatch, prompt.Templates{}, prompt.LanguageJapanese,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(batch, "SubAgent") || !strings.Contains(batch, "prx prompt TASK_ID") {
		t.Fatalf("the japanese batch prompt lost its instructions: %q", batch)
	}
	if strings.Contains(batch, "Implement the PRX tasks") {
		t.Fatalf("the japanese batch prompt kept the english text: %q", batch)
	}
}
