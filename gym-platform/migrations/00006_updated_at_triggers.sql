-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'roles', 'users', 'staff_profiles', 'members', 'plans', 'subscriptions',
    'payments', 'tickets', 'ticket_comments', 'posts', 'social_publications'
  ]
  LOOP
    EXECUTE format(
      'CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION set_updated_at()',
      t || '_set_updated_at', t
    );
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'roles', 'users', 'staff_profiles', 'members', 'plans', 'subscriptions',
    'payments', 'tickets', 'ticket_comments', 'posts', 'social_publications'
  ]
  LOOP
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', t || '_set_updated_at', t);
  END LOOP;
END $$;
-- +goose StatementEnd