-- +goose Up
ALTER TABLE subscriptions ADD COLUMN frozen_on DATE;
ALTER TABLE payments ADD COLUMN note TEXT;

-- Safety net behind the application rules: one current, one queued.
CREATE UNIQUE INDEX subscriptions_one_current_idx
  ON subscriptions (member_id) WHERE status IN ('active', 'frozen');
CREATE UNIQUE INDEX subscriptions_one_queued_idx
  ON subscriptions (member_id) WHERE status = 'pending';

-- +goose Down
DROP INDEX subscriptions_one_queued_idx;
DROP INDEX subscriptions_one_current_idx;
ALTER TABLE payments DROP COLUMN note;
ALTER TABLE subscriptions DROP COLUMN frozen_on;
