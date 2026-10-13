DROP INDEX IF EXISTS idx_club_members_one_owner;

UPDATE club_members SET role = 'member' WHERE role = 'admin';

ALTER TABLE club_members DROP CONSTRAINT IF EXISTS club_members_role_check;

ALTER TABLE club_members
    ADD CONSTRAINT club_members_role_check CHECK (role IN ('owner', 'member'));
