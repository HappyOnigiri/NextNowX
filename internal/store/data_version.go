package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrDataVersionUnavailable は、この store では変更検知を行えないことを表す。
// 専用接続を確保するとプールを使い切る構成で返る。
var ErrDataVersionUnavailable = errors.New("data version tracking is unavailable for this database")

// ErrStoreClosed は閉じた store への呼び出しを表す。
var ErrStoreClosed = errors.New("store is closed")

// DataVersion は PRAGMA data_version を専用接続から読む。値は接続ローカルで、
// 別接続や別プロセスのコミットがあると変化する。意味と制約は
// docs/design/persistence.md を参照。
func (s *Store) DataVersion(ctx context.Context) (int64, error) {
	conn, err := s.dataVersionConn(ctx)
	if err != nil {
		return 0, err
	}
	version, err := readDataVersion(ctx, conn)
	if err == nil {
		return version, nil
	}
	// *sql.Conn はドライバ接続が壊れると以後すべて同じエラーを返す。張り直さないと
	// 一過性の失敗で変更検知だけが永久に止まる。
	if !errors.Is(err, sql.ErrConnDone) {
		return 0, err
	}
	conn, err = s.resetDataVersionConn(ctx)
	if err != nil {
		return 0, err
	}
	return readDataVersion(ctx, conn)
}

// readDataVersion は 1 行を読み切る。この接続で rows を開いたまま残したり別のクエリを
// 走らせたりすると read transaction が張られ、値が凍結して無言で検知が止まる。
func readDataVersion(ctx context.Context, conn *sql.Conn) (int64, error) {
	var version int64
	if err := conn.QueryRowContext(ctx, `PRAGMA data_version`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read data version: %w", err)
	}
	return version, nil
}

func (s *Store) dataVersionConn(ctx context.Context) (*sql.Conn, error) {
	state := &s.dataVersion
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.ensureConn(ctx, s.db)
}

func (s *Store) resetDataVersionConn(ctx context.Context) (*sql.Conn, error) {
	state := &s.dataVersion
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.conn != nil {
		_ = state.conn.Close()
		state.conn = nil
	}
	return state.ensureConn(ctx, s.db)
}

func (state *dataVersionState) ensureConn(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	if state.unavailable {
		return nil, ErrDataVersionUnavailable
	}
	if state.closed {
		return nil, ErrStoreClosed
	}
	if state.conn != nil {
		return state.conn, nil
	}
	conn, err := database.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("open data version connection: %w", err)
	}
	state.conn = conn
	return conn, nil
}

func (s *Store) closeDataVersion() error {
	state := &s.dataVersion
	state.mu.Lock()
	defer state.mu.Unlock()
	state.closed = true
	if state.conn == nil {
		return nil
	}
	conn := state.conn
	state.conn = nil
	return conn.Close()
}
