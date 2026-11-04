ALTER TABLE admin_actions
ALTER COLUMN target_id TYPE VARCHAR(64) USING target_id::text;

CREATE INDEX IF NOT EXISTS idx_admin_actions_target
    ON admin_actions (target_type, target_id);
