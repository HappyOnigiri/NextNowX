-- 一括設計プロンプトの上書き列。空値はアプリケーション層で NULL にする。
ALTER TABLE projects ADD COLUMN prompt_batch_design TEXT;
ALTER TABLE features ADD COLUMN prompt_batch_design TEXT;
