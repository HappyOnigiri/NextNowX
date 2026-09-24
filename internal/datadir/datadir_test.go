package datadir

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s should not exist: %v", path, err)
	}
}

// 旧 prx のディレクトリだけがあれば、設定・稼働記録・データベースと sidecar を新しい名前へ移す。
func TestDatabasePathMigratesLegacyDirectory(t *testing.T) {
	base := t.TempDir()
	t.Setenv("HOME", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	configBase, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(configBase, "prx")
	writeFile(t, filepath.Join(legacy, "config.yaml"), "config")
	writeFile(t, filepath.Join(legacy, "run", "serve.json"), "run")
	writeFile(t, filepath.Join(legacy, "prx.db"), "db")
	writeFile(t, filepath.Join(legacy, "prx.db-wal"), "wal")

	path, err := DatabasePath()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(configBase, "nnx")
	if path != filepath.Join(dir, "nnx.db") {
		t.Fatalf("DatabasePath()=%q", path)
	}
	assertMissing(t, legacy)
	assertMissing(t, filepath.Join(dir, "prx.db"))
	assertMissing(t, filepath.Join(dir, "nnx.db-shm"))
	for file, want := range map[string]string{
		"config.yaml": "config", filepath.Join("run", "serve.json"): "run", "nnx.db": "db", "nnx.db-wal": "wal",
	} {
		if got := readFile(t, filepath.Join(dir, file)); got != want {
			t.Fatalf("%s=%q, want %q", file, got, want)
		}
	}
}

// 新しいディレクトリがあれば、旧 prx のディレクトリには触れない。
func TestMigrateKeepsLegacyWhenNewDirectoryExists(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "prx", "prx.db"), "legacy")
	writeFile(t, filepath.Join(base, "nnx", "nnx.db"), "current")

	dir, err := migrate(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := renameDatabase(dir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(base, "prx", "prx.db")); got != "legacy" {
		t.Fatalf("legacy database changed: %q", got)
	}
	if got := readFile(t, filepath.Join(dir, "nnx.db")); got != "current" {
		t.Fatalf("current database changed: %q", got)
	}
}

// どちらも無ければ何も作らない。作成は各ストレージの初回書き込みに任せる。
func TestMigrateWithoutLegacyCreatesNothing(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "prx"), "not a directory")

	dir, err := migrate(base)
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(base, "nnx") {
		t.Fatalf("migrate()=%q", dir)
	}
	assertMissing(t, dir)
	if err := renameDatabase(dir); err != nil {
		t.Fatal(err)
	}
}
