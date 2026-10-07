-- +goose Up
CREATE SEQUENCE member_code_seq START 1;

-- +goose Down
DROP SEQUENCE member_code_seq;
