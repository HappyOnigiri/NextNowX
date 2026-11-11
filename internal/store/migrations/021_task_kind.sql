-- 既存のタスクも開発として扱う。
ALTER TABLE tasks ADD COLUMN kind TEXT NOT NULL DEFAULT 'development' CHECK(kind IN ('development','work'));
