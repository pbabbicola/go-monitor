-- +goose Up
-- +goose StatementBegin
create table logs (
    ts timestamp with time zone,
    url varchar,
    duration_milliseconds bigint,
    status_code integer,
    regexp_matches boolean,
    error varchar
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table logs;
-- +goose StatementEnd
