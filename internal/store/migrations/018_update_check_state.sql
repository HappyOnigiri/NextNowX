CREATE TABLE update_check_state (
  singleton INTEGER PRIMARY KEY CHECK(singleton = 1),
  last_checked_unix INTEGER,
  check_error TEXT NOT NULL DEFAULT '',
  releases TEXT NOT NULL DEFAULT ''
);

INSERT INTO update_check_state(singleton) VALUES(1);
