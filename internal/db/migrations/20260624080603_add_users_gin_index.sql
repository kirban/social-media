-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

CREATE INDEX IF NOT EXISTS idx_users_fname_sname_gin
ON public.users USING gin (first_name, second_name public.gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS idx_users_fname_sname_gin;
