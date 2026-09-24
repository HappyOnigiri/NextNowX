// Package datadir は設定・データベース・稼働記録を置く既定のディレクトリを解決する。
// 改名前の prx が使っていた場所からの移行もここだけで行う。docs/design/persistence.md を参照。
package datadir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	// name はユーザー設定ディレクトリの下に作るディレクトリ名。
	name = "nnx"
	// legacyName は改名前の prx が使っていたディレクトリ名。
	legacyName = "prx"
	// DatabaseFile は既定のデータベースのファイル名。
	DatabaseFile = "nnx.db"
	// legacyDatabaseFile は改名前の prx が使っていたデータベースのファイル名。
	legacyDatabaseFile = "prx.db"
)

// Dir は既定のディレクトリを返す。新しい場所が無く旧 prx の場所だけがあれば、先に移す。
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return migrate(base)
}

// DatabasePath は既定のデータベースの位置を返す。移したディレクトリに旧名のファイルが
// 残っていれば、sidecar ごと新しい名前へ変える。
func DatabasePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := renameDatabase(dir); err != nil {
		return "", err
	}
	return filepath.Join(dir, DatabaseFile), nil
}

// migrate は両方があるときは新しい方を使い、旧 prx の場所には触れない。
func migrate(base string) (string, error) {
	dir := filepath.Join(base, name)
	if exists, err := pathExists(dir); err != nil || exists {
		return dir, err
	}
	legacy := filepath.Join(base, legacyName)
	info, err := os.Lstat(legacy)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && !info.IsDir()) {
		return dir, nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect legacy directory %s: %w", legacy, err)
	}
	if err := os.Rename(legacy, dir); err != nil {
		// 同時に起動した別のプロセスが先に移したなら、その結果を使う。
		if exists, _ := pathExists(dir); exists {
			return dir, nil
		}
		return "", fmt.Errorf("move legacy directory %s to %s: %w", legacy, dir, err)
	}
	return dir, nil
}

// renameDatabase は WAL の内容を失わないよう、sidecar を先に、本体を最後に移す。
// 途中で止まっても本体が旧名のまま残るので、次の起動で続きから移せる。
func renameDatabase(dir string) error {
	target := filepath.Join(dir, DatabaseFile)
	if exists, err := pathExists(target); err != nil || exists {
		return err
	}
	legacy := filepath.Join(dir, legacyDatabaseFile)
	if exists, err := pathExists(legacy); err != nil || !exists {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm", ""} {
		err := os.Rename(legacy+suffix, target+suffix)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("rename legacy database %s: %w", legacy+suffix, err)
		}
	}
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("inspect %s: %w", path, err)
}
