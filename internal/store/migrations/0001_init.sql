-- Timestamps are RFC 3339 text in UTC.

-- A repository puffin knows about. A row appears as soon as a pull request in
-- it does, which for a review can be long before there is a clone, so the two
-- local columns stay NULL until scanning the configured roots finds one.
CREATE TABLE repo (
  id              INTEGER PRIMARY KEY,
  name_with_owner TEXT NOT NULL UNIQUE,
  main_clone_path TEXT,
  default_branch  TEXT
) STRICT;

-- What you sit down to do. Owns a tmux session and a directory, and belongs to
-- no repository: a task with no worktree is an exploration and nothing more.
CREATE TABLE task (
  id         INTEGER PRIMARY KEY,
  slug       TEXT NOT NULL,            -- the tmux session name and directory name
  dir        TEXT NOT NULL,
  title      TEXT NOT NULL DEFAULT '',
  flow       TEXT NOT NULL CHECK (flow IN ('work', 'review')),
  state      TEXT NOT NULL DEFAULT 'open'
             CHECK (state IN ('open', 'done', 'discarded')),
  linear_id  TEXT,
  prompt     TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  closed_at  TEXT,                     -- when state left 'open'
  swept_at   TEXT                      -- when the session and directory were removed
) STRICT;

-- A slug names a directory, so it only has to be unique among the tasks that
-- still have one.
CREATE UNIQUE INDEX task_slug_live ON task (slug) WHERE swept_at IS NULL;
CREATE INDEX task_state ON task (state);

-- The main clones a task's root agent may read, named when the task is made.
CREATE TABLE task_repo (
  task_id INTEGER NOT NULL REFERENCES task (id) ON DELETE CASCADE,
  repo_id INTEGER NOT NULL REFERENCES repo (id) ON DELETE CASCADE,
  PRIMARY KEY (task_id, repo_id)
) STRICT, WITHOUT ROWID;

-- One per stack, mounted inside the task's directory.
CREATE TABLE worktree (
  id          INTEGER PRIMARY KEY,
  task_id     INTEGER NOT NULL REFERENCES task (id) ON DELETE CASCADE,
  repo_id     INTEGER NOT NULL REFERENCES repo (id),
  path        TEXT NOT NULL,
  current_ref TEXT,                    -- NULL while HEAD is detached
  created_at  TEXT NOT NULL,
  removed_at  TEXT
) STRICT;

CREATE UNIQUE INDEX worktree_path_live ON worktree (path) WHERE removed_at IS NULL;
CREATE INDEX worktree_task ON worktree (task_id);

-- A branch and, once it exists, its pull request. Keyed by branch because the
-- work exists before the pull request does, and GitHub allows one open pull
-- request per head branch. A branch reused after its pull request closed keeps
-- this row and takes the new number.
--
-- Stacks are not stored: they are the chains base_ref -> head_ref within a
-- repository, and restacking rewrites them.
CREATE TABLE change (
  id                INTEGER PRIMARY KEY,
  repo_id           INTEGER NOT NULL REFERENCES repo (id),
  head_ref          TEXT NOT NULL,
  base_ref          TEXT NOT NULL,
  task_id           INTEGER REFERENCES task (id) ON DELETE SET NULL,
  pr_number         INTEGER,
  role              TEXT NOT NULL CHECK (role IN ('author', 'reviewer')),
  author            TEXT,
  title             TEXT NOT NULL DEFAULT '',
  state             TEXT NOT NULL DEFAULT 'LOCAL'
                    CHECK (state IN ('LOCAL', 'OPEN', 'MERGED', 'CLOSED')),
  is_draft          INTEGER NOT NULL DEFAULT 0 CHECK (is_draft IN (0, 1)),
  status            TEXT NOT NULL DEFAULT 'UNKNOWN',  -- derived: whose move it is
  snapshot          TEXT NOT NULL DEFAULT '{}',       -- JSON: checks, threads, queue
  last_reviewed_sha TEXT,
  fetched_at        TEXT,
  UNIQUE (repo_id, head_ref)
) STRICT;

CREATE INDEX change_task ON change (task_id);
CREATE INDEX change_pr ON change (repo_id, pr_number);

-- One agent process. A task has one at its root and one per worktree window.
CREATE TABLE agent_run (
  id          INTEGER PRIMARY KEY,
  task_id     INTEGER NOT NULL REFERENCES task (id) ON DELETE CASCADE,
  worktree_id INTEGER REFERENCES worktree (id) ON DELETE SET NULL,  -- NULL: root agent
  agent       TEXT NOT NULL,
  session_id  TEXT,                    -- the agent's own id, for resuming
  state       TEXT NOT NULL DEFAULT 'working'
              CHECK (state IN ('working', 'waiting', 'done')),
  prompt      TEXT NOT NULL DEFAULT '',
  started_at  TEXT NOT NULL,
  ended_at    TEXT
) STRICT;

CREATE INDEX agent_run_task ON agent_run (task_id);

-- Append-only. Flows and notifications react to rows here rather than
-- comparing one poll with the next.
CREATE TABLE event (
  id           INTEGER PRIMARY KEY,
  at           TEXT NOT NULL,
  task_id      INTEGER REFERENCES task (id) ON DELETE SET NULL,
  change_id    INTEGER REFERENCES change (id) ON DELETE SET NULL,
  agent_run_id INTEGER REFERENCES agent_run (id) ON DELETE SET NULL,
  kind         TEXT NOT NULL,
  from_value   TEXT,
  to_value     TEXT
) STRICT;

CREATE INDEX event_at ON event (at);
CREATE INDEX event_change ON event (change_id);

-- Daemon bookkeeping: rate limit, notification etag, the author's login.
CREATE TABLE meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
) STRICT, WITHOUT ROWID;
