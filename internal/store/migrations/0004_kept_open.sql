-- A task reopened by hand after everything it shipped had merged: there is
-- more to do, with no pull request for it yet. It stays open until one
-- opens, and from then on finishes as any other task does.
ALTER TABLE task ADD COLUMN kept_open INTEGER NOT NULL DEFAULT 0 CHECK (kept_open IN (0, 1));
