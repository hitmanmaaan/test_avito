DROP TRIGGER IF EXISTS trg_pull_requests_set_merged_at ON pull_requests;
DROP FUNCTION IF EXISTS fn_pull_requests_set_merged_at();

DROP TRIGGER IF EXISTS trg_pr_reviewers_limit_two ON pr_reviewers;
DROP FUNCTION IF EXISTS fn_pr_reviewers_limit_two();

DROP TRIGGER IF EXISTS trg_pr_reviewers_block_if_merged ON pr_reviewers;
DROP FUNCTION IF EXISTS fn_pr_reviewers_block_if_merged();
