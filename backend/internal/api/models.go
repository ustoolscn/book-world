package api

import "net/http"

type modelsResponse struct {
	Models []string `json:"models"`
}

func (s *Server) ListModels(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	models, err := s.LLM.ListModels(r.Context(), user.BaseURL, user.APIKey)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, modelsResponse{Models: models})
}
