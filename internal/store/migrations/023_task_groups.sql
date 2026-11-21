CREATE TABLE id_sequences_new (
  entity TEXT PRIMARY KEY CHECK(entity IN ('feature', 'task', 'project', 'task_group')),
  next_value INTEGER NOT NULL CHECK(next_value > 0)
);

INSERT INTO id_sequences_new SELECT entity, next_value FROM id_sequences;
DROP TABLE id_sequences;
ALTER TABLE id_sequences_new RENAME TO id_sequences;

INSERT INTO id_sequences (entity, next_value) VALUES ('task_group', 1);

CREATE TABLE task_groups (
  public_id TEXT PRIMARY KEY NOT NULL,
  feature_id TEXT NOT NULL REFERENCES features(id) ON DELETE CASCADE,
  name TEXT NOT NULL CHECK(length(trim(name)) > 0),
  UNIQUE (feature_id, name)
);

-- NULL が未所属を表す。グループを削除しても、タスクとその依存は保持する。
ALTER TABLE tasks ADD COLUMN group_id TEXT REFERENCES task_groups(public_id) ON DELETE SET NULL;
CREATE INDEX tasks_group_idx ON tasks(group_id);
