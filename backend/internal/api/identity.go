package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"book-world/backend/internal/model"

	"github.com/google/uuid"
)

type enterIdentityRequest struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
}

type enterIdentityResponse struct {
	SessionID string `json:"sessionId"`
}

func (s *Server) EnterIdentity(w http.ResponseWriter, r *http.Request) {
	var req enterIdentityRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	baseURL := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	apiKey := strings.TrimSpace(req.APIKey)
	if baseURL == "" || apiKey == "" {
		writeError(w, http.StatusBadRequest, "baseUrl and apiKey are required")
		return
	}
	identityHash := hashIdentity(baseURL, apiKey)
	var userID string
	err := s.DB.Pool.QueryRow(r.Context(), `
		INSERT INTO users (id, identity_hash, base_url, api_key)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (identity_hash) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key = EXCLUDED.api_key,
			last_seen_at = now()
		RETURNING id::text
	`, uuid.NewString(), identityHash, baseURL, apiKey).Scan(&userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save identity")
		return
	}
	token := uuid.NewString()
	_, err = s.DB.Pool.Exec(r.Context(), `
		INSERT INTO auth_sessions (id, user_id, token)
		VALUES ($1, $2, $3)
	`, uuid.NewString(), userID, token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	writeJSON(w, http.StatusOK, enterIdentityResponse{SessionID: token})
}

func (s *Server) userByToken(ctx context.Context, token string) (model.User, error) {
	var user model.User
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT u.id::text, u.base_url, u.api_key, u.created_at
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND (s.expires_at IS NULL OR s.expires_at > now())
	`, token).Scan(&user.ID, &user.BaseURL, &user.APIKey, &user.CreatedAt)
	return user, err
}

func hashIdentity(baseURL, apiKey string) string {
	sum := sha256.Sum256([]byte(normalizeIdentityBaseURL(baseURL) + ":" + strings.TrimSpace(apiKey)))
	return hex.EncodeToString(sum[:])
}

func normalizeIdentityBaseURL(baseURL string) string {
	value := strings.TrimRight(strings.ToLower(strings.TrimSpace(baseURL)), "/")
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	return value
}
