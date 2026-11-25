-- A pull request that no task owns used to be deleted once GitHub stopped
-- returning it. It is now kept, so the log of what was shipped and reviewed
-- stays whole, and marked here so that nothing live shows it again.
ALTER TABLE change ADD COLUMN archived_at TEXT;
