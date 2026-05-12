ALTER TABLE stories
  ADD COLUMN IF NOT EXISTS created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS stories_created_by_user_idx ON stories(created_by_user_id, created_at);
