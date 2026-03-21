-- +goose Up
-- Historical migration retained as a no-op because the base schema already contains
-- the final structure from earlier in-code migrations.
SELECT 1;

-- +goose Down
SELECT 1;
