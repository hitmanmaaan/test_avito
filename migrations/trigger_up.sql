-- запрет изменений pr_reviewers если PR уже MERGED
CREATE OR REPLACE FUNCTION fn_pr_reviewers_block_if_merged() RETURNS trigger
    LANGUAGE plpgsql AS $$
DECLARE
    p_status pr_status;
BEGIN
    -- Получаем статус PR
    SELECT status INTO p_status FROM pull_requests WHERE pull_request_id = COALESCE(NEW.pr_id, OLD.pr_id) FOR SHARE;
    IF p_status = 'MERGED' THEN
        RAISE EXCEPTION 'PR_MERGED: cannot modify reviewers on merged PR';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_pr_reviewers_block_if_merged
    BEFORE INSERT OR UPDATE OR DELETE ON pr_reviewers
    FOR EACH ROW EXECUTE FUNCTION fn_pr_reviewers_block_if_merged();


-- ограничение — не более 2 ревьюверов на PR

CREATE OR REPLACE FUNCTION fn_pr_reviewers_limit_two() RETURNS trigger
    LANGUAGE plpgsql AS $$
DECLARE
    cnt INTEGER;
BEGIN
    IF (TG_OP = 'INSERT') THEN
        SELECT COUNT(*) INTO cnt FROM pr_reviewers WHERE pr_id = NEW.pr_id;
        IF cnt >= 2 THEN
            RAISE EXCEPTION 'PR_REVIEWERS_LIMIT: cannot assign more than 2 reviewers to a PR';
        END IF;
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        -- при удалении всегда OK
        RETURN OLD;
    ELSIF (TG_OP = 'UPDATE') THEN
        RETURN NEW;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_pr_reviewers_limit_two
    BEFORE INSERT ON pr_reviewers
    FOR EACH ROW EXECUTE FUNCTION fn_pr_reviewers_limit_two();


-- при переводе PR в MERGED — устанавливаем merged_at

CREATE OR REPLACE FUNCTION fn_pull_requests_set_merged_at() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF OLD.status IS DISTINCT FROM NEW.status THEN
            IF NEW.status = 'MERGED' THEN
                -- Если merged_at не задан, ставим now
                IF NEW.merged_at IS NULL THEN
                    NEW.merged_at := now();
                END IF;
            ELSE
                -- При переводе обратно в OPEN — очищаем merged_at
                NEW.merged_at := NULL;
            END IF;
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_pull_requests_set_merged_at
    BEFORE UPDATE ON pull_requests
    FOR EACH ROW EXECUTE FUNCTION fn_pull_requests_set_merged_at();
