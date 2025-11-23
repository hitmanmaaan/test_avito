package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hitmanmaaan/test_avito/internal/model"
	"github.com/jmoiron/sqlx"
)

// PostgresRepository реализует доступ к БД.
type PostgresRepository struct {
	DB *sqlx.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		DB: sqlx.NewDb(db, "pgx"),
	}
}

/* Teams & Users */

func (r *PostgresRepository) CreateTeamWithMembers(ctx context.Context, team model.Team) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO teams (team_name) VALUES ($1)`, team.TeamName)
	if err != nil {
		return err
	}

	for _, m := range team.Members {
		_, err = tx.ExecContext(ctx, `
INSERT INTO users (user_id, username, team_name, is_active)
VALUES ($1,$2,$3,$4)
ON CONFLICT (user_id) DO UPDATE
  SET username = EXCLUDED.username,
      team_name = EXCLUDED.team_name,
      is_active = EXCLUDED.is_active
`, m.UserID, m.Username, team.TeamName, m.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetTeam(ctx context.Context, teamName string) (model.Team, error) {
	t := model.Team{TeamName: teamName}
	var members []model.TeamMember
	err := r.DB.SelectContext(ctx, &members, `SELECT user_id, username, is_active FROM users WHERE team_name = $1`, teamName)
	if err != nil && err != sql.ErrNoRows {
		return t, err
	}
	t.Members = members
	return t, nil
}

func (r *PostgresRepository) SetUserIsActive(ctx context.Context, userID string, isActive bool) (model.User, error) {
	_, err := r.DB.ExecContext(ctx, `UPDATE users SET is_active=$2 WHERE user_id=$1`, userID, isActive)
	if err != nil {
		return model.User{}, err
	}
	var u model.User
	err = r.DB.GetContext(ctx, &u, `SELECT user_id, username, team_name, is_active FROM users WHERE user_id=$1`, userID)
	if err != nil {
		return model.User{}, err
	}
	return u, nil
}

/* Pull Requests */

func (r *PostgresRepository) CreatePR(ctx context.Context, pr model.PullRequest) error {
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
VALUES ($1,$2,$3,$4)`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status)
	return err
}

func (r *PostgresRepository) GetPR(ctx context.Context, prID string) (model.PullRequest, error) {
	var pr model.PullRequest
	err := r.DB.GetContext(ctx, &pr, `SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at FROM pull_requests WHERE pull_request_id=$1`, prID)
	if err != nil {
		return pr, err
	}
	var reviewers []string
	err = r.DB.SelectContext(ctx, &reviewers, `SELECT user_id FROM pr_reviewers WHERE pr_id=$1 ORDER BY assigned_at`, prID)
	if err != nil && err != sql.ErrNoRows {
		return pr, err
	}
	pr.AssignedReviewers = reviewers
	return pr, nil
}

func (r *PostgresRepository) AddPRReviewer(ctx context.Context, prID string, userID string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1,$2)`, prID, userID)
	return err
}

func (r *PostgresRepository) RemovePRReviewer(ctx context.Context, prID string, userID string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM pr_reviewers WHERE pr_id=$1 AND user_id=$2`, prID, userID)
	return err
}

func (r *PostgresRepository) SetPRMerged(ctx context.Context, prID string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE pull_requests SET status='MERGED' WHERE pull_request_id=$1`, prID)
	return err
}

func (r *PostgresRepository) GetActiveTeamMembersExcept(ctx context.Context, teamName string, excludes []string) ([]string, error) {
	q := `SELECT user_id FROM users WHERE team_name=$1 AND is_active = true`
	var args []interface{}
	args = append(args, teamName)

	if len(excludes) > 0 {
		placeholders := make([]string, len(excludes))
		for i := range excludes {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args = append(args, excludes[i])
		}
		q += " AND user_id NOT IN (" + strings.Join(placeholders, ",") + ")"
	}
	q += " ORDER BY random()"
	var ids []string
	err := r.DB.SelectContext(ctx, &ids, q, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return ids, nil
}

func (r *PostgresRepository) GetUser(ctx context.Context, userID string) (model.User, error) {
	var u model.User
	err := r.DB.GetContext(ctx, &u, `SELECT user_id, username, team_name, is_active FROM users WHERE user_id=$1`, userID)
	return u, err
}

func (r *PostgresRepository) GetAssignedPRsForUser(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	var prs []model.PullRequestShort
	err := r.DB.SelectContext(ctx, &prs, `
SELECT p.pull_request_id, p.pull_request_name, p.author_id, p.status
FROM pull_requests p
JOIN pr_reviewers r ON r.pr_id = p.pull_request_id
WHERE r.user_id = $1
ORDER BY p.created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	return prs, nil
}
