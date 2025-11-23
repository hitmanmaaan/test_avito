DROP INDEX IF EXISTS idx_pr_reviewers_pr;
DROP INDEX IF EXISTS idx_pr_reviewers_user;

DROP TABLE IF EXISTS pr_reviewers;
DROP TABLE IF EXISTS pull_requests;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS teams;

DROP TYPE IF EXISTS pr_status;
