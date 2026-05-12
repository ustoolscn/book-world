package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"book-world/backend/internal/contextbuilder"
	"book-world/backend/internal/llm"
	"book-world/backend/internal/model"
)

type streamChatRequest struct {
	StorySlug      string          `json:"storySlug"`
	Message        string          `json:"message"`
	Model          string          `json:"model"`
	ThinkingEffort string          `json:"thinkingEffort"`
	Messages       []model.Message `json:"messages"`
	Summary        string          `json:"summary"`
	UserProfile    string          `json:"userProfile"`
}

func (s *Server) StreamChat(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req streamChatRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.StorySlug == "" || req.Message == "" {
		writeError(w, http.StatusBadRequest, "storySlug and message are required")
		return
	}

	story, err := s.storyBySlug(r.Context(), req.StorySlug)
	if err != nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	characters, err := s.charactersByStory(r.Context(), story.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load characters")
		return
	}
	worldInfo, err := s.worldInfoByStory(r.Context(), story.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load world info")
		return
	}

	messages := contextbuilder.Build(contextbuilder.Input{
		Story:            story,
		Characters:       characters,
		WorldInfo:        worldInfo,
		Summary:          req.Summary,
		UserProfile:      req.UserProfile,
		RecentMessages:   normalizedRecentMessages(req.Messages),
		CurrentUserInput: req.Message,
		CharBudget:       s.Config.ContextCharBudget,
		ReplyReserve:     s.Config.ReplyCharReserve,
	})

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	modelName := req.Model
	if modelName == "" {
		modelName = s.Config.DefaultModel
	}
	_, usage, err := s.LLM.StreamChat(r.Context(), llm.StreamRequest{
		BaseURL:        user.BaseURL,
		APIKey:         user.APIKey,
		Model:          modelName,
		ThinkingEffort: req.ThinkingEffort,
		Messages:       messages,
	}, func(delta string) error {
		return writeSSE(w, flusher, "delta", map[string]string{"content": delta})
	})
	if err != nil {
		_ = writeSSE(w, flusher, "error", map[string]string{"message": err.Error()})
		return
	}
	_ = writeSSE(w, flusher, "usage", usage)
	_ = writeSSE(w, flusher, "done", map[string]bool{})
}

func normalizedRecentMessages(messages []model.Message) []model.Message {
	recent := make([]model.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
			continue
		}
		if msg.Content == "" {
			continue
		}
		if msg.CreatedAt.IsZero() {
			msg.CreatedAt = time.Now()
		}
		msg.TokenEstimate = len(msg.Content)
		recent = append(recent, msg)
	}
	return recent
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
