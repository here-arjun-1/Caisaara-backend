DROP INDEX IF EXISTS idx_admin_actions_target;

DELETE FROM admin_actions WHERE target_id !~ '^[0-9]+$';

ALTER TABLE admin_actions
ALTER COLUMN target_id TYPE BIGINT USING target_id::bigint;
