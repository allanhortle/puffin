-- A task that lives in a directory it did not make and does not own: a clone
-- that is hacked on directly, with no worktree and no pull request to follow.
-- Nothing in such a directory is ever created, moved or removed by puffin.
ALTER TABLE task ADD COLUMN in_place INTEGER NOT NULL DEFAULT 0 CHECK (in_place IN (0, 1));
