-- project と feature ごとの prompt override。空値はアプリケーション層で NULL にする。
ALTER TABLE projects ADD COLUMN prompt_design TEXT;
ALTER TABLE projects ADD COLUMN prompt_implementation TEXT;
ALTER TABLE projects ADD COLUMN prompt_batch TEXT;
ALTER TABLE features ADD COLUMN prompt_design TEXT;
ALTER TABLE features ADD COLUMN prompt_implementation TEXT;
ALTER TABLE features ADD COLUMN prompt_batch TEXT;
