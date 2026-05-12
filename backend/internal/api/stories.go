package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"book-world/backend/internal/model"

	"github.com/google/uuid"
)

type storySessionResponse struct {
	ChatSessionID string          `json:"chatSessionId"`
	Title         string          `json:"title"`
	Summary       string          `json:"summary"`
	Messages      []model.Message `json:"messages"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type saveStorySessionRequest struct {
	Title    string          `json:"title"`
	Summary  string          `json:"summary"`
	Messages []model.Message `json:"messages"`
}

type storySettingsRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	CoverURL       string `json:"coverUrl"`
	SystemPrompt   string `json:"systemPrompt"`
	Scenario       string `json:"scenario"`
	StylePrompt    string `json:"stylePrompt"`
	OpeningMessage string `json:"openingMessage"`
}

type likeStoryResponse struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"likeCount"`
}

type paginatedStoriesResponse struct {
	Items  []model.Story `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

func (s *Server) ListStories(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	limit, hasPagination := parsePositiveInt(r.URL.Query().Get("limit"))
	offset, hasOffset := parsePositiveInt(r.URL.Query().Get("offset"))
	if !hasOffset {
		offset = 0
	}
	if hasPagination {
		if limit > 60 {
			limit = 60
		}
		s.listStoriesPage(w, r, user.ID, limit, offset)
		return
	}

	rows, err := s.DB.Pool.Query(r.Context(), `
		SELECT st.id::text, st.slug, st.title, COALESCE(st.description, ''), COALESCE(st.cover_url, ''), st.system_prompt,
			COALESCE(st.scenario, ''), COALESCE(st.style_prompt, ''), COALESCE(st.opening_message, ''),
			COUNT(sl.story_id)::int AS like_count,
			COUNT(my_like.story_id) > 0 AS liked,
			st.created_at, st.updated_at
		FROM stories st
		LEFT JOIN story_likes sl ON sl.story_id = st.id
		LEFT JOIN story_likes my_like ON my_like.story_id = st.id AND my_like.user_id = $1
		GROUP BY st.id
		ORDER BY st.created_at DESC
	`, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load stories")
		return
	}
	defer rows.Close()

	var stories []model.Story
	for rows.Next() {
		var story model.Story
		if err := rows.Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.LikeCount, &story.Liked, &story.CreatedAt, &story.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read stories")
			return
		}
		stories = append(stories, story)
	}
	writeJSON(w, http.StatusOK, stories)
}

func (s *Server) listStoriesPage(w http.ResponseWriter, r *http.Request, userID string, limit, offset int) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	sortMode := strings.TrimSpace(r.URL.Query().Get("sort"))
	whereSQL := ""
	args := []any{}
	if search != "" {
		whereSQL = "WHERE st.title ILIKE $1 OR COALESCE(st.description, '') ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	var total int
	countSQL := `SELECT COUNT(*) FROM stories st ` + whereSQL
	if err := s.DB.Pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count stories")
		return
	}

	orderSQL := "ORDER BY st.created_at DESC"
	if sortMode == "likes" {
		orderSQL = "ORDER BY like_count DESC, st.created_at DESC"
	}
	userParam := len(args) + 1
	limitParam := len(args) + 2
	offsetParam := len(args) + 3
	args = append(args, userID, limit, offset)
	rows, err := s.DB.Pool.Query(r.Context(), `
		SELECT st.id::text, st.slug, st.title, COALESCE(st.description, ''), COALESCE(st.cover_url, ''), st.system_prompt,
			COALESCE(st.scenario, ''), COALESCE(st.style_prompt, ''), COALESCE(st.opening_message, ''),
			COUNT(sl.story_id)::int AS like_count,
			COUNT(my_like.story_id) > 0 AS liked,
			st.created_at, st.updated_at
		FROM stories st
		LEFT JOIN story_likes sl ON sl.story_id = st.id
		LEFT JOIN story_likes my_like ON my_like.story_id = st.id AND my_like.user_id = $`+strconv.Itoa(userParam)+`
		`+whereSQL+`
		GROUP BY st.id
		`+orderSQL+`
		LIMIT $`+strconv.Itoa(limitParam)+` OFFSET $`+strconv.Itoa(offsetParam)+`
	`, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load stories")
		return
	}
	defer rows.Close()

	var stories []model.Story
	for rows.Next() {
		var story model.Story
		if err := rows.Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.LikeCount, &story.Liked, &story.CreatedAt, &story.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read stories")
			return
		}
		stories = append(stories, story)
	}
	writeJSON(w, http.StatusOK, paginatedStoriesResponse{Items: stories, Total: total, Limit: limit, Offset: offset})
}

func (s *Server) ToggleStoryLike(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	story, err := s.storyBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	var liked bool
	err = s.DB.Pool.QueryRow(r.Context(), `
		WITH deleted AS (
			DELETE FROM story_likes WHERE user_id = $1 AND story_id = $2 RETURNING 1
		),
		inserted AS (
			INSERT INTO story_likes (user_id, story_id)
			SELECT $1, $2
			WHERE NOT EXISTS (SELECT 1 FROM deleted)
			RETURNING 1
		)
		SELECT EXISTS (SELECT 1 FROM inserted)
	`, user.ID, story.ID).Scan(&liked)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update like")
		return
	}
	var likeCount int
	if err := s.DB.Pool.QueryRow(r.Context(), `SELECT COUNT(*)::int FROM story_likes WHERE story_id = $1`, story.ID).Scan(&likeCount); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count likes")
		return
	}
	writeJSON(w, http.StatusOK, likeStoryResponse{Liked: liked, LikeCount: likeCount})
}

func parsePositiveInt(value string) (int, bool) {
	if strings.TrimSpace(value) == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func (s *Server) GetStory(w http.ResponseWriter, r *http.Request) {
	story, err := s.storyBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (s *Server) GetStorySettings(w http.ResponseWriter, r *http.Request) {
	story, err := s.storyBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (s *Server) UpdateStorySettings(w http.ResponseWriter, r *http.Request) {
	story, err := s.storyBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}

	var req storySettingsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	title := strings.TrimSpace(req.Title)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if title == "" || systemPrompt == "" {
		writeError(w, http.StatusBadRequest, "title and systemPrompt are required")
		return
	}

	err = s.DB.Pool.QueryRow(r.Context(), `
		UPDATE stories
		SET title = $1, description = $2, cover_url = $3, system_prompt = $4,
			scenario = $5, style_prompt = $6, opening_message = $7, updated_at = now()
		WHERE id = $8
		RETURNING id::text, slug, title, COALESCE(description, ''), COALESCE(cover_url, ''), system_prompt,
			COALESCE(scenario, ''), COALESCE(style_prompt, ''), COALESCE(opening_message, ''), created_at, updated_at
	`, title, strings.TrimSpace(req.Description), strings.TrimSpace(req.CoverURL), systemPrompt, strings.TrimSpace(req.Scenario), strings.TrimSpace(req.StylePrompt), strings.TrimSpace(req.OpeningMessage), story.ID).
		Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update story settings")
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (s *Server) ListStorySessions(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	story, err := s.storyBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}

	rows, err := s.DB.Pool.Query(r.Context(), `
			SELECT cs.id::text, cs.user_id::text, cs.story_id::text, COALESCE(cs.title, ''), cs.summary,
				COUNT(m.id)::int, cs.created_at, cs.updated_at
			FROM chat_sessions cs
			LEFT JOIN messages m ON m.chat_session_id = cs.id
			WHERE cs.user_id = $1 AND cs.story_id = $2
			GROUP BY cs.id
			ORDER BY cs.updated_at DESC
		`, user.ID, story.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load saved sessions")
		return
	}
	defer rows.Close()

	sessions := []model.ChatSession{}
	for rows.Next() {
		var session model.ChatSession
		if err := rows.Scan(&session.ID, &session.UserID, &session.StoryID, &session.Title, &session.Summary, &session.MessageCount, &session.CreatedAt, &session.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read saved sessions")
			return
		}
		sessions = append(sessions, session)
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *Server) GetStorySession(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	story, session, ok := s.sessionForStory(w, r, user.ID)
	if !ok {
		return
	}
	_ = story

	messages, err := s.messagesBySession(r.Context(), session.ID, 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}
	writeJSON(w, http.StatusOK, storySessionResponse{
		ChatSessionID: session.ID,
		Title:         session.Title,
		Summary:       session.Summary,
		Messages:      messages,
		CreatedAt:     session.CreatedAt,
		UpdatedAt:     session.UpdatedAt,
	})
}

func (s *Server) SaveStorySession(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	story, err := s.storyBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}

	var req saveStorySessionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages are required")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = defaultSessionTitle(story.Title, req.Messages)
	}

	tx, err := s.DB.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin save")
		return
	}
	defer tx.Rollback(r.Context())

	sessionID := uuid.NewString()
	var session model.ChatSession
	err = tx.QueryRow(r.Context(), `
			INSERT INTO chat_sessions (id, user_id, story_id, title, summary)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id::text, user_id::text, story_id::text, COALESCE(title, ''), summary, created_at, updated_at
		`, sessionID, user.ID, story.ID, title, req.Summary).Scan(&session.ID, &session.UserID, &session.StoryID, &session.Title, &session.Summary, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create saved session")
		return
	}

	for _, msg := range req.Messages {
		role := strings.TrimSpace(msg.Role)
		if role != "user" && role != "assistant" && role != "system" {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		if _, err := tx.Exec(r.Context(), `
				INSERT INTO messages (id, chat_session_id, role, content, token_estimate)
				VALUES ($1, $2, $3, $4, $5)
			`, uuid.NewString(), session.ID, role, content, len(content)); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save messages")
			return
		}
		session.MessageCount++
	}
	if session.MessageCount == 0 {
		writeError(w, http.StatusBadRequest, "no valid messages to save")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit save")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (s *Server) DeleteStorySession(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	_, session, ok := s.sessionForStory(w, r, user.ID)
	if !ok {
		return
	}

	_, err := s.DB.Pool.Exec(r.Context(), `DELETE FROM chat_sessions WHERE id = $1 AND user_id = $2`, session.ID, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete saved session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) sessionForStory(w http.ResponseWriter, r *http.Request, userID string) (model.Story, model.ChatSession, bool) {
	story, err := s.storyBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return model.Story{}, model.ChatSession{}, false
	}

	var session model.ChatSession
	err = s.DB.Pool.QueryRow(r.Context(), `
			SELECT id::text, user_id::text, story_id::text, COALESCE(title, ''), summary, created_at, updated_at
			FROM chat_sessions
			WHERE id = $1 AND user_id = $2 AND story_id = $3
		`, r.PathValue("id"), userID, story.ID).Scan(&session.ID, &session.UserID, &session.StoryID, &session.Title, &session.Summary, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "saved session not found")
		return model.Story{}, model.ChatSession{}, false
	}
	return story, session, true
}

func defaultSessionTitle(storyTitle string, messages []model.Message) string {
	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		if len([]rune(content)) > 24 {
			return string([]rune(content)[:24]) + "..."
		}
		return content
	}
	return storyTitle + " 记录"
}
