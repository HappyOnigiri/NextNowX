-- 作業範囲の本文を通常の添付資料へ移し、タスクの専用列を削除する。
INSERT INTO documents (
  id, task_id, kind, title, content, is_implementation_plan, created_at, updated_at
)
SELECT lower(hex(randomblob(16))), id, 'markdown', 'Scope', scope, 0, created_at, updated_at
FROM tasks
WHERE length(trim(scope)) > 0;

ALTER TABLE tasks DROP COLUMN scope;
