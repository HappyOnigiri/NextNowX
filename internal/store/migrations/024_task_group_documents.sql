CREATE TABLE documents_new (
  id TEXT PRIMARY KEY,
  project_id TEXT REFERENCES projects(id) ON DELETE RESTRICT,
  feature_id TEXT REFERENCES features(id) ON DELETE RESTRICT,
  task_id TEXT REFERENCES tasks(id) ON DELETE RESTRICT,
  group_id TEXT REFERENCES task_groups(public_id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK(kind IN ('url','local_file','markdown')),
  title TEXT NOT NULL DEFAULT '',
  locator TEXT,
  content TEXT,
  is_implementation_plan INTEGER NOT NULL DEFAULT 0 CHECK(is_implementation_plan IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK(
    (project_id IS NOT NULL) + (feature_id IS NOT NULL) +
    (task_id IS NOT NULL) + (group_id IS NOT NULL) = 1
  ),
  CHECK(is_implementation_plan = 0 OR task_id IS NOT NULL),
  CHECK(
    (kind IN ('url','local_file') AND locator IS NOT NULL AND length(trim(locator)) > 0 AND content IS NULL) OR
    (kind = 'markdown' AND locator IS NULL AND content IS NOT NULL AND length(trim(content)) > 0)
  )
);

INSERT INTO documents_new (
  id, project_id, feature_id, task_id, kind, title, locator, content,
  is_implementation_plan, created_at, updated_at
)
SELECT
  id, project_id, feature_id, task_id, kind, title, locator, content,
  is_implementation_plan, created_at, updated_at
FROM documents;

DROP TABLE documents;
ALTER TABLE documents_new RENAME TO documents;

CREATE INDEX documents_project_idx ON documents(project_id, created_at, id);
CREATE INDEX documents_feature_idx ON documents(feature_id, created_at, id);
CREATE INDEX documents_task_idx ON documents(task_id, is_implementation_plan DESC, created_at, id);

CREATE INDEX documents_group_idx ON documents(group_id);

CREATE UNIQUE INDEX documents_one_plan_per_task_idx
ON documents(task_id) WHERE is_implementation_plan = 1;
