ALTER TABLE club_members DROP CONSTRAINT IF EXISTS club_members_role_check;

ALTER TABLE club_members
    ADD CONSTRAINT club_members_role_check CHECK (role IN ('owner', 'admin', 'member'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_club_members_one_owner
    ON club_members(club_id) WHERE role = 'owner';
