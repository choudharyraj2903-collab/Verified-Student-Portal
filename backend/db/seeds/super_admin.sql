-- =========================================================================
-- SUPER ADMIN SEED
-- =========================================================================
-- INSTRUCTIONS:
-- 1. Run this file ONCE manually, directly against the database
-- 2. Never expose this through any API endpoint
-- 3. Replace ALL placeholder values before running
-- 4. Delete or vault this file after first run
-- 5. Run AFTER schema.sql and functions.sql
--
-- Command to run:
--   psql -U <db_user> -d <db_name> -f super_admin.sql
-- =========================================================================

BEGIN;

-- -------------------------------------------------------------------------
-- Safety Check — Abort if a SUPER_ADMIN already exists
-- Prevents accidental double-seeding
-- -------------------------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM users WHERE role = 'SUPER_ADMIN'
    ) THEN
        RAISE EXCEPTION 'ABORT: A SUPER_ADMIN already exists in the database. This seed must only run once.';
    END IF;
END;
$$;

-- -------------------------------------------------------------------------
-- Insert Super Admin
-- Replace email with the actual super admin institutional email
-- -------------------------------------------------------------------------
INSERT INTO users (
    id,
    email,
    role,
    is_active,
    created_at,
    updated_at
)
VALUES (
    uuid_generate_v4(),
    'super_admin@iitk.ac.in',     -- REPLACE with actual institutional email
    'SUPER_ADMIN',
    TRUE,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
);

-- -------------------------------------------------------------------------
-- Verify insertion before committing
-- -------------------------------------------------------------------------
DO $$
DECLARE
    admin_count INT;
BEGIN
    SELECT COUNT(*) INTO admin_count
    FROM users
    WHERE role = 'SUPER_ADMIN'
      AND is_active = TRUE;

    IF admin_count != 1 THEN
        RAISE EXCEPTION 'ABORT: Super admin insertion verification failed. Rolling back.';
    END IF;

    RAISE NOTICE 'SUCCESS: Super admin seeded successfully. Email: %',
        (SELECT email FROM users WHERE role = 'SUPER_ADMIN');
END;
$$;

COMMIT;

-- =========================================================================
-- Post-run checklist (verify manually after running):
-- [ ] Confirm exactly one SUPER_ADMIN exists:
--       SELECT id, email, role, is_active, created_at FROM users WHERE role = 'SUPER_ADMIN';
-- [ ] Confirm no other roles were affected:
--       SELECT role, COUNT(*) FROM users GROUP BY role;
-- [ ] Vault or delete this file — it has no use after first run
-- =========================================================================