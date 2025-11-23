package service

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/yourname/pr-reviewer/internal/apperrors"
	"github.com/yourname/pr-reviewer/internal/model"
	"github.com/yourname/pr-reviewer/internal/repository"
)

type Service struct {
	repo *repository.PostgresRepository
}

func NewService(repo *repository.PostgresRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateTeam(ctx context.Context, t model.Team) error {
	err := s.repo.CreateTeamWithMembers(ctx, t)
	if err != nil {
		// detect unique violation roughly
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "violates unique constraint") {
			return apperrors.ErrTeamExists
		}
		return err
	}
	return nil
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (model.Team, error) {
	return s.repo.GetTeam(ctx, teamName)
}

func (s *Service) SetUserIsActive(ctx context.Context, userID string, isActive bool) (model.User, error) {
	u, err := s.repo.SetUserIsActive(ctx, userID, isActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, apperrors.ErrNotFound
		}
		return model.User{}, err
	}
	return u, nil
}

func (s *Service) CreatePullRequest(ctx context.Context, pr model.PullRequest) (model.PullRequest, error) {
	_, err := s.repo.GetPR(ctx, pr.PullRequestID)
	if err == nil {
		return model.PullRequest{}, apperrors.ErrPRExists
	}
	if err != nil && err != sql.ErrNoRows {
		return model.PullRequest{}, err
	}

	author, err := s.repo.GetUser(ctx, pr.AuthorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.PullRequest{}, apperrors.ErrNotFound
		}
		return model.PullRequest{}, err
	}
	teamName := author.TeamName

	candidates, err := s.repo.GetActiveTeamMembersExcept(ctx, teamName, []string{author.UserID})
	if err != nil {
		return model.PullRequest{}, err
	}
	assign := candidates
	if len(assign) > 2 {
		assign = assign[:2]
	}

	tx, err := s.repo.DB.BeginTxx(ctx, nil)
	if err != nil {
		return model.PullRequest{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES ($1,$2,$3,$4)`,
		pr.PullRequestID, pr.PullRequestName, pr.AuthorID, "OPEN")
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "violates unique constraint") {
			return model.PullRequest{}, apperrors.ErrPRExists
		}
		return model.PullRequest{}, err
	}

	for _, u := range assign {
		if _, err := tx.ExecContext(ctx, `INSERT INTO pr_reviewers (pr_id,user_id) VALUES ($1,$2)`, pr.PullRequestID, u); err != nil {
			return model.PullRequest{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return model.PullRequest{}, err
	}

	resPr, err := s.repo.GetPR(ctx, pr.PullRequestID)
	if err != nil {
		return model.PullRequest{}, err
	}
	return resPr, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (model.PullRequest, error) {
	tx, err := s.repo.DB.BeginTxx(ctx, nil)
	if err != nil {
		return model.PullRequest{}, err
	}
	defer tx.Rollback()

	var status string
	if err := tx.GetContext(ctx, &status, `SELECT status FROM pull_requests WHERE pull_request_id=$1 FOR UPDATE`, prID); err != nil {
		if err == sql.ErrNoRows {
			return model.PullRequest{}, apperrors.ErrNotFound
		}
		return model.PullRequest{}, err
	}

	if status == "MERGED" {
		if err := tx.Commit(); err != nil {
			return model.PullRequest{}, err
		}
		return s.repo.GetPR(ctx, prID)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE pull_requests SET status='MERGED', merged_at = now() WHERE pull_request_id=$1`, prID); err != nil {
		return model.PullRequest{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.PullRequest{}, err
	}

	return s.repo.GetPR(ctx, prID)
}

func (s *Service) ReassignReviewer(ctx context.Context, prID string, oldUserID string) (model.PullRequest, string, error) {
	tx, err := s.repo.DB.BeginTxx(ctx, nil)
	if err != nil {
		return model.PullRequest{}, "", err
	}
	defer tx.Rollback()

	var status string
	if err := tx.GetContext(ctx, &status, `SELECT status FROM pull_requests WHERE pull_request_id=$1 FOR UPDATE`, prID); err != nil {
		if err == sql.ErrNoRows {
			return model.PullRequest{}, "", apperrors.ErrNotFound
		}
		return model.PullRequest{}, "", err
	}
	if status == "MERGED" {
		return model.PullRequest{}, "", apperrors.ErrPRMerged
	}

	var exists int
	if err := tx.GetContext(ctx, &exists, `SELECT 1 FROM pr_reviewers WHERE pr_id=$1 AND user_id=$2`, prID, oldUserID); err != nil {
		return model.PullRequest{}, "", err
	}
	if exists == 0 {
		return model.PullRequest{}, "", apperrors.ErrNotAssigned
	}

	oldUser, err := s.repo.GetUser(ctx, oldUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.PullRequest{}, "", apperrors.ErrNotFound
		}
		return model.PullRequest{}, "", err
	}
	teamName := oldUser.TeamName

	pr, err := s.repo.GetPR(ctx, prID)
	if err != nil {
		return model.PullRequest{}, "", err
	}
	excludes := append([]string{pr.AuthorID}, pr.AssignedReviewers...)

	candidates, err := s.repo.GetActiveTeamMembersExcept(ctx, teamName, excludes)
	if err != nil {
		return model.PullRequest{}, "", err
	}
	if len(candidates) == 0 {
		return model.PullRequest{}, "", apperrors.ErrNoCandidate
	}
	newUser := candidates[0]

	if _, err := tx.ExecContext(ctx, `DELETE FROM pr_reviewers WHERE pr_id=$1 AND user_id=$2`, prID, oldUserID); err != nil {
		return model.PullRequest{}, "", err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1,$2)`, prID, newUser); err != nil {
		return model.PullRequest{}, "", err
	}

	if err := tx.Commit(); err != nil {
		return model.PullRequest{}, "", err
	}

	updatedPr, err := s.repo.GetPR(ctx, prID)
	if err != nil {
		return model.PullRequest{}, "", err
	}
	return updatedPr, newUser, nil
}

func (s *Service) GetAssignedPRsForUser(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	return s.repo.GetAssignedPRsForUser(ctx, userID)
}
