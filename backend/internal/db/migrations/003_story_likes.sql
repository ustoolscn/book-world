CREATE TABLE IF NOT EXISTS story_likes (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  story_id UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, story_id)
);

CREATE INDEX IF NOT EXISTS story_likes_story_idx ON story_likes(story_id);
