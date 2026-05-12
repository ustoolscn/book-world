package api

import (
	"context"

	"book-world/backend/internal/model"

	"github.com/jackc/pgx/v5"
)

func (s *Server) storyBySlug(ctx context.Context, slug string) (model.Story, error) {
	var story model.Story
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT id::text, slug, title, COALESCE(description, ''), COALESCE(cover_url, ''), system_prompt,
			COALESCE(scenario, ''), COALESCE(style_prompt, ''), COALESCE(opening_message, ''), created_at, updated_at
		FROM stories
		WHERE slug = $1
	`, slug).Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.CreatedAt, &story.UpdatedAt)
	return story, err
}

func (s *Server) storyBySession(ctx context.Context, sessionID, userID string) (model.ChatSession, model.Story, error) {
	var session model.ChatSession
	var story model.Story
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT cs.id::text, cs.user_id::text, cs.story_id::text, COALESCE(cs.title, ''), cs.summary, cs.created_at, cs.updated_at,
			st.id::text, st.slug, st.title, COALESCE(st.description, ''), COALESCE(st.cover_url, ''), st.system_prompt,
			COALESCE(st.scenario, ''), COALESCE(st.style_prompt, ''), COALESCE(st.opening_message, ''), st.created_at, st.updated_at
		FROM chat_sessions cs
		JOIN stories st ON st.id = cs.story_id
		WHERE cs.id = $1 AND cs.user_id = $2
	`, sessionID, userID).Scan(
		&session.ID, &session.UserID, &session.StoryID, &session.Title, &session.Summary, &session.CreatedAt, &session.UpdatedAt,
		&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.CreatedAt, &story.UpdatedAt,
	)
	return session, story, err
}

func (s *Server) charactersByStory(ctx context.Context, storyID string) ([]model.Character, error) {
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT id::text, story_id::text, name, COALESCE(description, ''), COALESCE(personality, ''), COALESCE(example_dialogue, ''), priority
		FROM story_characters
		WHERE story_id = $1
		ORDER BY priority, name
	`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var characters []model.Character
	for rows.Next() {
		var character model.Character
		if err := rows.Scan(&character.ID, &character.StoryID, &character.Name, &character.Description, &character.Personality, &character.ExampleDialogue, &character.Priority); err != nil {
			return nil, err
		}
		characters = append(characters, character)
	}
	return characters, rows.Err()
}

func (s *Server) worldInfoByStory(ctx context.Context, storyID string) ([]model.WorldInfo, error) {
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT id::text, story_id::text, keywords, content, priority, enabled
		FROM world_info
		WHERE story_id = $1 AND enabled = true
		ORDER BY priority
	`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []model.WorldInfo
	for rows.Next() {
		var entry model.WorldInfo
		if err := rows.Scan(&entry.ID, &entry.StoryID, &entry.Keywords, &entry.Content, &entry.Priority, &entry.Enabled); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Server) messagesBySession(ctx context.Context, sessionID string, limit int) ([]model.Message, error) {
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT id::text, chat_session_id::text, role, content, token_estimate, created_at
		FROM (
			SELECT * FROM messages WHERE chat_session_id = $1 ORDER BY created_at DESC LIMIT $2
		) recent
		ORDER BY created_at ASC
	`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func scanMessages(rows pgx.Rows) ([]model.Message, error) {
	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		if err := rows.Scan(&msg.ID, &msg.ChatSessionID, &msg.Role, &msg.Content, &msg.TokenEstimate, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}
