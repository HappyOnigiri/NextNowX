package prompt

import (
	"os"
	"strings"
)

// Language は組み込みテンプレートを書き分ける言語。上書きテンプレートはユーザーが
// 書いた文字列なので、この型の対象にはならない。
type Language string

const (
	// LanguageEnglish は組み込みテンプレートの英語版を選ぶ。
	LanguageEnglish Language = "en"
	// LanguageJapanese は組み込みテンプレートの日本語版を選ぶ。
	LanguageJapanese Language = "ja"
)

// LanguageAutoValue は設定ファイルと CLI が共有する「環境から判定する」ことを
// 表す語彙。設定ファイルのキーがないときもこの値として扱う。
const LanguageAutoValue = "auto"

// localeVariables は auto のときに読むロケール変数。POSIX の優先順で、先に
// 値のあるものが勝つ。
var localeVariables = []string{"LC_ALL", "LC_MESSAGES", "LANG"}

// ParseLanguage は設定ファイルと CLI が受け付ける言語の語彙を解釈する。auto は
// ここでは解決せず、呼び出し元が実効言語を決めるまで保留する。
func ParseLanguage(value string) (Language, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(LanguageEnglish):
		return LanguageEnglish, true
	case string(LanguageJapanese):
		return LanguageJapanese, true
	default:
		return "", false
	}
}

// SupportedLanguages は組み込みテンプレートを持つ言語を返す。
func SupportedLanguages() []Language {
	return []Language{LanguageEnglish, LanguageJapanese}
}

// ResolveLanguage は設定値から実効言語を決める。プロセスの環境を読むのは設定が
// auto のときだけである。
func ResolveLanguage(setting string) Language {
	return ResolveLanguageWith(setting, os.Getenv)
}

// ResolveLanguageWith は環境変数の取得方法を差し替えられる ResolveLanguage。
// テストがプロセスの環境に依存せず実効言語を固定できるようにするためにある。
func ResolveLanguageWith(setting string, lookup func(string) string) Language {
	if language, ok := ParseLanguage(setting); ok {
		return language
	}
	for _, name := range localeVariables {
		if language, ok := languageOfLocale(lookup(name)); ok {
			return language
		}
	}
	return LanguageEnglish
}

// languageOfLocale は ja_JP.UTF-8 のようなロケールから言語を取り出す。C や
// POSIX のように言語を表さない値、および未対応の言語は判定できないものとして扱う。
func languageOfLocale(value string) (Language, bool) {
	base := strings.TrimSpace(value)
	for _, separator := range []string{".", "@", "_", "-"} {
		if index := strings.Index(base, separator); index >= 0 {
			base = base[:index]
		}
	}
	return ParseLanguage(base)
}
