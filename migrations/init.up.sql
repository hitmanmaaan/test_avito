-- migrations/0001_init.up.sql
-- Инициализация схемы: teams, users, pull_requests, pr_reviewers, индексы

-- Статус PR
CREATE TYPE pr_status AS ENUM ('OPEN', 'MERGED');

-- Teams
CREATE TABLE teams (
                       team_name TEXT PRIMARY KEY,
                       created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Users
CREATE TABLE users (
                       user_id TEXT PRIMARY KEY,
                       username TEXT NOT NULL,
                       team_name TEXT NOT NULL REFERENCES teams(team_name) ON DELETE RESTRICT,
                       is_active BOOLEAN NOT NULL DEFAULT TRUE,
                       created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Pull requests
CREATE TABLE pull_requests (
                               pull_request_id TEXT PRIMARY KEY,
                               pull_request_name TEXT NOT NULL,
                               author_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
                               status pr_status NOT NULL DEFAULT 'OPEN',
                               created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                               merged_at TIMESTAMPTZ
);

-- PR reviewers
CREATE TABLE pr_reviewers (
                              pr_id TEXT NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
                              user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
                              assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                              PRIMARY KEY (pr_id, user_id)
);

-- Индексы
CREATE INDEX idx_pr_reviewers_user ON pr_reviewers(user_id);
CREATE INDEX idx_pr_reviewers_pr ON pr_reviewers(pr_id);

-- 7) Небольшая целостность:
-- нельзя указать в pull_requests автора, которого нет в users — FK гарантирует это.
