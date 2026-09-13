package store

// CloseDataVersionConnForTest は専用接続だけを返却し、以後の利用で ErrConnDone を
// 起こす。実際の接続死亡はドライバの事情でしか起きないため、テストから作る。
func (s *Store) CloseDataVersionConnForTest() error {
	state := &s.dataVersion
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.conn == nil {
		return nil
	}
	return state.conn.Close()
}
