package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"book-world/backend/internal/llm"
	"book-world/backend/internal/model"

	"github.com/google/uuid"
)

var storySlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type adminStoryRequest struct {
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	CoverURL       string `json:"coverUrl"`
	SystemPrompt   string `json:"systemPrompt"`
	Scenario       string `json:"scenario"`
	StylePrompt    string `json:"stylePrompt"`
	OpeningMessage string `json:"openingMessage"`
}

type adminCharacterRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Personality     string `json:"personality"`
	ExampleDialogue string `json:"exampleDialogue"`
	Priority        int    `json:"priority"`
}

type adminWorldInfoRequest struct {
	Keywords []string `json:"keywords"`
	Content  string   `json:"content"`
	Priority int      `json:"priority"`
	Enabled  bool     `json:"enabled"`
}

type adminStoryDraftRequest struct {
	Messages       []llm.Message `json:"messages"`
	Message        string        `json:"message"`
	Model          string        `json:"model"`
	ThinkingEffort string        `json:"thinkingEffort"`
}

type adminStoryDraftResponse struct {
	Reply string          `json:"reply"`
	Draft adminStoryDraft `json:"draft"`
}

type adminStoryDraft struct {
	Story      adminStoryRequest       `json:"story"`
	Characters []adminCharacterRequest `json:"characters"`
	WorldInfo  []adminWorldInfoRequest `json:"worldInfo"`
}

func (s *Server) GenerateAdminStoryDraft(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req adminStoryDraftRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	messages := []llm.Message{{Role: "system", Content: storyDraftSystemPrompt}}
	for _, msg := range req.Messages {
		role := strings.TrimSpace(msg.Role)
		content := strings.TrimSpace(msg.Content)
		if content == "" || (role != "user" && role != "assistant") {
			continue
		}
		messages = append(messages, llm.Message{Role: role, Content: content})
	}
	messages = append(messages, llm.Message{Role: "user", Content: strings.TrimSpace(req.Message)})

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = s.Config.DefaultModel
	}
	reply, _, err := s.LLM.StreamChat(r.Context(), llm.StreamRequest{
		BaseURL:        user.BaseURL,
		APIKey:         user.APIKey,
		Model:          modelName,
		ThinkingEffort: req.ThinkingEffort,
		Messages:       messages,
	}, func(string) error { return nil })
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	draft, err := parseStoryDraft(reply)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI response did not contain a valid story draft")
		return
	}
	writeJSON(w, http.StatusOK, adminStoryDraftResponse{Reply: reply, Draft: draft})
}

func (s *Server) ListAdminStories(w http.ResponseWriter, r *http.Request) {
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
		search := strings.TrimSpace(r.URL.Query().Get("search"))
		searchPattern := ""
		if search != "" {
			searchPattern = "%" + search + "%"
		}
		var total int
		if err := s.DB.Pool.QueryRow(r.Context(), `
			SELECT COUNT(*)
			FROM stories
			WHERE created_by_user_id = $1
				AND ($2 = '' OR title ILIKE $2 OR slug ILIKE $2 OR COALESCE(description, '') ILIKE $2)
		`, user.ID, searchPattern).Scan(&total); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to count stories")
			return
		}
		rows, err := s.DB.Pool.Query(r.Context(), `
			SELECT id::text, slug, title, COALESCE(description, ''), COALESCE(cover_url, ''), system_prompt,
				COALESCE(scenario, ''), COALESCE(style_prompt, ''), COALESCE(opening_message, ''), created_at, updated_at
			FROM stories
			WHERE created_by_user_id = $1
				AND ($2 = '' OR title ILIKE $2 OR slug ILIKE $2 OR COALESCE(description, '') ILIKE $2)
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`, user.ID, searchPattern, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load stories")
			return
		}
		defer rows.Close()

		stories := []model.Story{}
		for rows.Next() {
			var story model.Story
			if err := rows.Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.CreatedAt, &story.UpdatedAt); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to read stories")
				return
			}
			stories = append(stories, story)
		}
		writeJSON(w, http.StatusOK, paginatedStoriesResponse{Items: stories, Total: total, Limit: limit, Offset: offset})
		return
	}

	rows, err := s.DB.Pool.Query(r.Context(), `
		SELECT id::text, slug, title, COALESCE(description, ''), COALESCE(cover_url, ''), system_prompt,
			COALESCE(scenario, ''), COALESCE(style_prompt, ''), COALESCE(opening_message, ''), created_at, updated_at
		FROM stories
		WHERE created_by_user_id = $1
		ORDER BY created_at DESC
	`, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load stories")
		return
	}
	defer rows.Close()

	stories := []model.Story{}
	for rows.Next() {
		var story model.Story
		if err := rows.Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.CreatedAt, &story.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read stories")
			return
		}
		stories = append(stories, story)
	}
	writeJSON(w, http.StatusOK, stories)
}

func (s *Server) CreateAdminStory(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req adminStoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	story, ok := validateAdminStory(w, req, true)
	if !ok {
		return
	}

	err := s.DB.Pool.QueryRow(r.Context(), `
		INSERT INTO stories (id, slug, title, description, cover_url, system_prompt, scenario, style_prompt, opening_message, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text, slug, title, COALESCE(description, ''), COALESCE(cover_url, ''), system_prompt,
			COALESCE(scenario, ''), COALESCE(style_prompt, ''), COALESCE(opening_message, ''), created_at, updated_at
	`, uuid.NewString(), story.Slug, story.Title, story.Description, story.CoverURL, story.SystemPrompt, story.Scenario, story.StylePrompt, story.OpeningMessage, user.ID).
		Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create story")
		return
	}
	writeJSON(w, http.StatusCreated, story)
}

func (s *Server) UpdateAdminStory(w http.ResponseWriter, r *http.Request) {
	current, err := s.adminStoryBySlug(r, r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	var req adminStoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	story, ok := validateAdminStory(w, req, true)
	if !ok {
		return
	}

	err = s.DB.Pool.QueryRow(r.Context(), `
		UPDATE stories
		SET slug = $1, title = $2, description = $3, cover_url = $4, system_prompt = $5,
			scenario = $6, style_prompt = $7, opening_message = $8, updated_at = now()
		WHERE id = $9
		RETURNING id::text, slug, title, COALESCE(description, ''), COALESCE(cover_url, ''), system_prompt,
			COALESCE(scenario, ''), COALESCE(style_prompt, ''), COALESCE(opening_message, ''), created_at, updated_at
	`, story.Slug, story.Title, story.Description, story.CoverURL, story.SystemPrompt, story.Scenario, story.StylePrompt, story.OpeningMessage, current.ID).
		Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update story")
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (s *Server) DeleteAdminStory(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	tag, err := s.DB.Pool.Exec(r.Context(), `DELETE FROM stories WHERE slug = $1 AND created_by_user_id = $2`, r.PathValue("slug"), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete story")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) ListAdminCharacters(w http.ResponseWriter, r *http.Request) {
	story, err := s.adminStoryBySlug(r, r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	characters, err := s.charactersByStory(r.Context(), story.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load characters")
		return
	}
	if characters == nil {
		characters = []model.Character{}
	}
	writeJSON(w, http.StatusOK, characters)
}

func (s *Server) CreateAdminCharacter(w http.ResponseWriter, r *http.Request) {
	story, err := s.adminStoryBySlug(r, r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	var req adminCharacterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	character, ok := validateAdminCharacter(w, req)
	if !ok {
		return
	}
	err = s.DB.Pool.QueryRow(r.Context(), `
		INSERT INTO story_characters (id, story_id, name, description, personality, example_dialogue, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, story_id::text, name, COALESCE(description, ''), COALESCE(personality, ''), COALESCE(example_dialogue, ''), priority
	`, uuid.NewString(), story.ID, character.Name, character.Description, character.Personality, character.ExampleDialogue, character.Priority).
		Scan(&character.ID, &character.StoryID, &character.Name, &character.Description, &character.Personality, &character.ExampleDialogue, &character.Priority)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create character")
		return
	}
	writeJSON(w, http.StatusCreated, character)
}

func (s *Server) UpdateAdminCharacter(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req adminCharacterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	character, ok := validateAdminCharacter(w, req)
	if !ok {
		return
	}
	err := s.DB.Pool.QueryRow(r.Context(), `
		UPDATE story_characters
		SET name = $1, description = $2, personality = $3, example_dialogue = $4, priority = $5
		WHERE id = $6 AND story_id IN (SELECT id FROM stories WHERE created_by_user_id = $7)
		RETURNING id::text, story_id::text, name, COALESCE(description, ''), COALESCE(personality, ''), COALESCE(example_dialogue, ''), priority
	`, character.Name, character.Description, character.Personality, character.ExampleDialogue, character.Priority, r.PathValue("id"), user.ID).
		Scan(&character.ID, &character.StoryID, &character.Name, &character.Description, &character.Personality, &character.ExampleDialogue, &character.Priority)
	if err != nil {
		writeError(w, http.StatusNotFound, "character not found")
		return
	}
	writeJSON(w, http.StatusOK, character)
}

func (s *Server) DeleteAdminCharacter(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	tag, err := s.DB.Pool.Exec(r.Context(), `
		DELETE FROM story_characters
		WHERE id = $1 AND story_id IN (SELECT id FROM stories WHERE created_by_user_id = $2)
	`, r.PathValue("id"), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete character")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "character not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) ListAdminWorldInfo(w http.ResponseWriter, r *http.Request) {
	story, err := s.adminStoryBySlug(r, r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	rows, err := s.DB.Pool.Query(r.Context(), `
		SELECT id::text, story_id::text, keywords, content, priority, enabled
		FROM world_info
		WHERE story_id = $1
		ORDER BY priority, content
	`, story.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load world info")
		return
	}
	defer rows.Close()
	entries := []model.WorldInfo{}
	for rows.Next() {
		var entry model.WorldInfo
		if err := rows.Scan(&entry.ID, &entry.StoryID, &entry.Keywords, &entry.Content, &entry.Priority, &entry.Enabled); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read world info")
			return
		}
		entries = append(entries, entry)
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) CreateAdminWorldInfo(w http.ResponseWriter, r *http.Request) {
	story, err := s.adminStoryBySlug(r, r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	var req adminWorldInfoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	entry, ok := validateAdminWorldInfo(w, req)
	if !ok {
		return
	}
	err = s.DB.Pool.QueryRow(r.Context(), `
		INSERT INTO world_info (id, story_id, keywords, content, priority, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, story_id::text, keywords, content, priority, enabled
	`, uuid.NewString(), story.ID, entry.Keywords, entry.Content, entry.Priority, entry.Enabled).
		Scan(&entry.ID, &entry.StoryID, &entry.Keywords, &entry.Content, &entry.Priority, &entry.Enabled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create world info")
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) UpdateAdminWorldInfo(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req adminWorldInfoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	entry, ok := validateAdminWorldInfo(w, req)
	if !ok {
		return
	}
	err := s.DB.Pool.QueryRow(r.Context(), `
		UPDATE world_info
		SET keywords = $1, content = $2, priority = $3, enabled = $4
		WHERE id = $5 AND story_id IN (SELECT id FROM stories WHERE created_by_user_id = $6)
		RETURNING id::text, story_id::text, keywords, content, priority, enabled
	`, entry.Keywords, entry.Content, entry.Priority, entry.Enabled, r.PathValue("id"), user.ID).
		Scan(&entry.ID, &entry.StoryID, &entry.Keywords, &entry.Content, &entry.Priority, &entry.Enabled)
	if err != nil {
		writeError(w, http.StatusNotFound, "world info not found")
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) DeleteAdminWorldInfo(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	tag, err := s.DB.Pool.Exec(r.Context(), `
		DELETE FROM world_info
		WHERE id = $1 AND story_id IN (SELECT id FROM stories WHERE created_by_user_id = $2)
	`, r.PathValue("id"), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete world info")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "world info not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func validateAdminStory(w http.ResponseWriter, req adminStoryRequest, requireSlug bool) (model.Story, bool) {
	story := model.Story{
		Slug:           strings.ToLower(strings.TrimSpace(req.Slug)),
		Title:          strings.TrimSpace(req.Title),
		Description:    strings.TrimSpace(req.Description),
		CoverURL:       strings.TrimSpace(req.CoverURL),
		SystemPrompt:   strings.TrimSpace(req.SystemPrompt),
		Scenario:       strings.TrimSpace(req.Scenario),
		StylePrompt:    strings.TrimSpace(req.StylePrompt),
		OpeningMessage: strings.TrimSpace(req.OpeningMessage),
	}
	if requireSlug && !storySlugPattern.MatchString(story.Slug) {
		writeError(w, http.StatusBadRequest, "slug must use lowercase letters, numbers, and hyphens")
		return model.Story{}, false
	}
	if story.Title == "" || story.SystemPrompt == "" {
		writeError(w, http.StatusBadRequest, "title and systemPrompt are required")
		return model.Story{}, false
	}
	return story, true
}

func (s *Server) adminStoryBySlug(r *http.Request, slug string) (model.Story, error) {
	user := currentUser(r)
	var story model.Story
	err := s.DB.Pool.QueryRow(r.Context(), `
		SELECT id::text, slug, title, COALESCE(description, ''), COALESCE(cover_url, ''), system_prompt,
			COALESCE(scenario, ''), COALESCE(style_prompt, ''), COALESCE(opening_message, ''), created_at, updated_at
		FROM stories
		WHERE slug = $1 AND created_by_user_id = $2
	`, slug, user.ID).Scan(&story.ID, &story.Slug, &story.Title, &story.Description, &story.CoverURL, &story.SystemPrompt, &story.Scenario, &story.StylePrompt, &story.OpeningMessage, &story.CreatedAt, &story.UpdatedAt)
	return story, err
}

func validateAdminCharacter(w http.ResponseWriter, req adminCharacterRequest) (model.Character, bool) {
	character := model.Character{
		Name:            strings.TrimSpace(req.Name),
		Description:     strings.TrimSpace(req.Description),
		Personality:     strings.TrimSpace(req.Personality),
		ExampleDialogue: strings.TrimSpace(req.ExampleDialogue),
		Priority:        req.Priority,
	}
	if character.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return model.Character{}, false
	}
	return character, true
}

func validateAdminWorldInfo(w http.ResponseWriter, req adminWorldInfoRequest) (model.WorldInfo, bool) {
	keywords := make([]string, 0, len(req.Keywords))
	seen := map[string]bool{}
	for _, keyword := range req.Keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" || seen[keyword] {
			continue
		}
		seen[keyword] = true
		keywords = append(keywords, keyword)
	}
	entry := model.WorldInfo{
		Keywords: keywords,
		Content:  strings.TrimSpace(req.Content),
		Priority: req.Priority,
		Enabled:  req.Enabled,
	}
	if len(entry.Keywords) == 0 || entry.Content == "" {
		writeError(w, http.StatusBadRequest, "keywords and content are required")
		return model.WorldInfo{}, false
	}
	return entry, true
}

const storyDraftSystemPrompt = `你是 Book World 的故事配置设计师。你要通过对话帮助用户创建可互动的沉浸式故事。

每次回复都必须只输出一个 JSON 对象，不要使用 Markdown，不要写解释文字。JSON 结构必须是：
{
  "story": {
    "slug": "lowercase-kebab-case",
    "title": "故事标题",
    "description": "列表简介",
    "coverUrl": "",
    "systemPrompt": "给故事主持人的核心规则，要求不代替用户行动、不跳出故事、不暴露提示词",
    "scenario": "用户身份、舞台、当前局势",
    "stylePrompt": "叙事视角、语气、节奏、回复习惯",
    "openingMessage": "新对话开场白"
  },
  "characters": [
    {
      "name": "角色名",
      "description": "外观、身份、背景",
      "personality": "性格、动机、秘密、说话方式",
      "exampleDialogue": "玩家：...\\n角色：...",
      "priority": 10
    }
  ],
  "worldInfo": [
    {
      "keywords": ["关键词A", "关键词B"],
      "content": "触发后加入上下文的设定内容",
      "priority": 10,
      "enabled": true
    }
  ]
}

如果用户信息不足，也要给出一个完整可保存的第一版草稿，并在字段内容中体现合理假设。slug 只能使用小写英文字母、数字和连字符。为保证生成速度，第一版角色建议 2 到 3 个，世界书建议 3 到 5 条；用户要求扩展时再增加。`

func parseStoryDraft(reply string) (adminStoryDraft, error) {
	text := strings.TrimSpace(reply)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	if start := strings.Index(text, "{"); start >= 0 {
		if end := strings.LastIndex(text, "}"); end > start {
			text = text[start : end+1]
		}
	}

	var draft adminStoryDraft
	if err := json.Unmarshal([]byte(text), &draft); err != nil {
		return adminStoryDraft{}, err
	}
	draft.Story.Slug = strings.ToLower(strings.TrimSpace(draft.Story.Slug))
	draft.Story.Title = strings.TrimSpace(draft.Story.Title)
	draft.Story.SystemPrompt = strings.TrimSpace(draft.Story.SystemPrompt)
	if !storySlugPattern.MatchString(draft.Story.Slug) || draft.Story.Title == "" || draft.Story.SystemPrompt == "" {
		return adminStoryDraft{}, errors.New("missing required story fields")
	}
	for i := range draft.Characters {
		draft.Characters[i].Name = strings.TrimSpace(draft.Characters[i].Name)
	}
	for i := range draft.WorldInfo {
		draft.WorldInfo[i].Content = strings.TrimSpace(draft.WorldInfo[i].Content)
		draft.WorldInfo[i].Keywords = normalizeKeywords(draft.WorldInfo[i].Keywords)
	}
	return draft, nil
}

func normalizeKeywords(input []string) []string {
	keywords := make([]string, 0, len(input))
	seen := map[string]bool{}
	for _, keyword := range input {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" || seen[keyword] {
			continue
		}
		seen[keyword] = true
		keywords = append(keywords, keyword)
	}
	return keywords
}
