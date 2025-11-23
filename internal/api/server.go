package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hitmanmaaan/test_avito/internal/model"
	"github.com/hitmanmaaan/test_avito/internal/repository"
	"github.com/hitmanmaaan/test_avito/internal/service"
)

type Server struct {
	repo *repository.PostgresRepository
	svc  *service.Service
	r    *chi.Mux
}

func NewServer(repo *repository.PostgresRepository) *Server {
	s := &Server{
		repo: repo,
		svc:  service.NewService(repo),
		r:    chi.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) Router() http.Handler {
	return s.r
}

func (s *Server) setupRoutes() {
	s.r.Use(JSONContentTypeMiddleware)

	s.r.Post("/team/add", s.handleTeamAdd)
	s.r.Get("/team/get", s.handleTeamGet)
	s.r.Post("/users/setIsActive", s.handleSetIsActive)
	s.r.Get("/users/getReview", s.handleGetUserReviews)
	s.r.Post("/pullRequest/create", s.handleCreatePR)
	s.r.Post("/pullRequest/merge", s.handleMergePR)
	s.r.Post("/pullRequest/reassign", s.handleReassign)

	s.r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
}

func (s *Server) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	var t model.Team
	if err := decodeJSON(r, &t); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID", err.Error())
		return
	}
	if t.TeamName == "" {
		writeError(w, http.StatusBadRequest, "INVALID", "team_name is required")
		return
	}
	if err := s.svc.CreateTeam(r.Context(), t); err != nil {
		mapServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"team": t})
}

func (s *Server) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		writeError(w, http.StatusBadRequest, "INVALID", "team_name query is required")
		return
	}

	// Check existence in teams table
	var tname string
	if err := s.repo.DB.GetContext(r.Context(), &tname, `SELECT team_name FROM teams WHERE team_name=$1`, teamName); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "team not found")
			return
		}
		mapServiceError(w, err)
		return
	}

	team, err := s.svc.GetTeam(r.Context(), teamName)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(team)
}

func (s *Server) handleSetIsActive(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID", err.Error())
		return
	}
	u, err := s.svc.SetUserIsActive(r.Context(), payload.UserID, payload.IsActive)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"user": u})
}

func (s *Server) handleGetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "INVALID", "user_id is required")
		return
	}
	prs, err := s.svc.GetAssignedPRsForUser(r.Context(), userID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	resp := map[string]any{
		"user_id":       userID,
		"pull_requests": prs,
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCreatePR(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID", err.Error())
		return
	}
	if payload.PullRequestID == "" || payload.PullRequestName == "" || payload.AuthorID == "" {
		writeError(w, http.StatusBadRequest, "INVALID", "missing required fields")
		return
	}
	pr := model.PullRequest{
		PullRequestID:   payload.PullRequestID,
		PullRequestName: payload.PullRequestName,
		AuthorID:        payload.AuthorID,
		Status:          "OPEN",
	}
	created, err := s.svc.CreatePullRequest(r.Context(), pr)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"pr": created})
}

func (s *Server) handleMergePR(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PullRequestID string `json:"pull_request_id"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID", err.Error())
		return
	}
	updated, err := s.svc.MergePullRequest(r.Context(), payload.PullRequestID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"pr": updated})
}

func (s *Server) handleReassign(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_reviewer_id"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID", err.Error())
		return
	}
	pr, newUser, err := s.svc.ReassignReviewer(r.Context(), payload.PullRequestID, payload.OldUserID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	resp := map[string]any{
		"pr":          pr,
		"replaced_by": newUser,
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
