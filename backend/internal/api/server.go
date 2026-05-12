package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"book-world/backend/internal/config"
	"book-world/backend/internal/db"
	"book-world/backend/internal/llm"
	"book-world/backend/internal/model"
)

type contextKey string

const userContextKey contextKey = "user"

type Server struct {
	DB     *db.DB
	Config config.Config
	LLM    *llm.Client
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/identity/enter", s.EnterIdentity)
	mux.Handle("GET /api/models", s.Auth(http.HandlerFunc(s.ListModels)))
	mux.Handle("GET /api/stories", s.Auth(http.HandlerFunc(s.ListStories)))
	mux.Handle("GET /api/stories/{slug}", s.Auth(http.HandlerFunc(s.GetStory)))
	mux.Handle("POST /api/stories/{slug}/like", s.Auth(http.HandlerFunc(s.ToggleStoryLike)))
	mux.Handle("GET /api/stories/{slug}/settings", s.Auth(http.HandlerFunc(s.GetStorySettings)))
	mux.Handle("PATCH /api/stories/{slug}/settings", s.Auth(http.HandlerFunc(s.UpdateStorySettings)))
	mux.Handle("GET /api/stories/{slug}/sessions", s.Auth(http.HandlerFunc(s.ListStorySessions)))
	mux.Handle("GET /api/stories/{slug}/sessions/{id}", s.Auth(http.HandlerFunc(s.GetStorySession)))
	mux.Handle("POST /api/stories/{slug}/sessions", s.Auth(http.HandlerFunc(s.SaveStorySession)))
	mux.Handle("DELETE /api/stories/{slug}/sessions/{id}", s.Auth(http.HandlerFunc(s.DeleteStorySession)))
	mux.Handle("POST /api/admin/story-drafts", s.Auth(http.HandlerFunc(s.GenerateAdminStoryDraft)))
	mux.Handle("GET /api/admin/stories", s.Auth(http.HandlerFunc(s.ListAdminStories)))
	mux.Handle("POST /api/admin/stories", s.Auth(http.HandlerFunc(s.CreateAdminStory)))
	mux.Handle("PATCH /api/admin/stories/{slug}", s.Auth(http.HandlerFunc(s.UpdateAdminStory)))
	mux.Handle("DELETE /api/admin/stories/{slug}", s.Auth(http.HandlerFunc(s.DeleteAdminStory)))
	mux.Handle("GET /api/admin/stories/{slug}/characters", s.Auth(http.HandlerFunc(s.ListAdminCharacters)))
	mux.Handle("POST /api/admin/stories/{slug}/characters", s.Auth(http.HandlerFunc(s.CreateAdminCharacter)))
	mux.Handle("PATCH /api/admin/characters/{id}", s.Auth(http.HandlerFunc(s.UpdateAdminCharacter)))
	mux.Handle("DELETE /api/admin/characters/{id}", s.Auth(http.HandlerFunc(s.DeleteAdminCharacter)))
	mux.Handle("GET /api/admin/stories/{slug}/world-info", s.Auth(http.HandlerFunc(s.ListAdminWorldInfo)))
	mux.Handle("POST /api/admin/stories/{slug}/world-info", s.Auth(http.HandlerFunc(s.CreateAdminWorldInfo)))
	mux.Handle("PATCH /api/admin/world-info/{id}", s.Auth(http.HandlerFunc(s.UpdateAdminWorldInfo)))
	mux.Handle("DELETE /api/admin/world-info/{id}", s.Auth(http.HandlerFunc(s.DeleteAdminWorldInfo)))
	mux.Handle("POST /api/chat/stream", s.Auth(http.HandlerFunc(s.StreamChat)))
	if s.Config.StaticDir != "" {
		mux.Handle("GET /", http.HandlerFunc(s.ServeFrontend))
	}
	return s.Middleware(mux)
}

func (s *Server) ServeFrontend(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if path == "." || path == "" {
		path = "index.html"
	}
	fullPath := filepath.Join(s.Config.StaticDir, path)
	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		http.ServeFile(w, r, fullPath)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.Config.StaticDir, "index.html"))
}

func (s *Server) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "frame-ancestors *")
		w.Header().Set("Access-Control-Allow-Origin", s.Config.FrontendOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		user, err := s.userByToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid session")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUser(r *http.Request) model.User {
	return r.Context().Value(userContextKey).(model.User)
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}
