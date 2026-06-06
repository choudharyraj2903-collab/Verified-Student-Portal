-- =========================================================================
-- COUNCILS SEED
-- =========================================================================
-- INSTRUCTIONS:
-- 1. Run AFTER schema.sql and functions.sql
-- 2. Run BEFORE super_admin.sql (super admin seed has no council dependency
--    but council admins assigned later require these rows to exist)
-- 3. Safe to run multiple times — ON CONFLICT DO NOTHING prevents duplicates
-- 4. These four councils are fixed by the system design and must never be
--    created, renamed, or deleted through the application API
--
-- Command to run:
--   psql -U <db_user> -d <db_name> -f db/seeds/councils.sql
-- =========================================================================

BEGIN;

-- -------------------------------------------------------------------------
-- Safety check — warn if councils already exist
-- Does not abort — ON CONFLICT handles duplicates gracefully
-- -------------------------------------------------------------------------
DO $$
DECLARE
    existing_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO existing_count FROM councils;
    IF existing_count > 0 THEN
        RAISE NOTICE 'councils table already has % rows — duplicate inserts will be skipped', existing_count;
    END IF;
END;
$$;

-- -------------------------------------------------------------------------
-- Insert the four fixed councils
-- Codes must match exactly what the application uses in validation logic
-- and what domain admins are assigned to
-- -------------------------------------------------------------------------
INSERT INTO councils (id, code, name, created_at) VALUES
    (uuid_generate_v4(), 'GNS', 'Gymkhana and Sports',      CURRENT_TIMESTAMP),
    (uuid_generate_v4(), 'ANC', 'Arts and Cultural',         CURRENT_TIMESTAMP),
    (uuid_generate_v4(), 'SNT', 'Science and Technology',    CURRENT_TIMESTAMP),
    (uuid_generate_v4(), 'MNC', 'Music and Cultural',        CURRENT_TIMESTAMP)
ON CONFLICT (code) DO NOTHING;

-- -------------------------------------------------------------------------
-- Verify all four councils exist before committing
-- -------------------------------------------------------------------------
DO $$
DECLARE
    council_count INTEGER;
    missing_codes TEXT;
BEGIN
    SELECT COUNT(*) INTO council_count
    FROM councils
    WHERE code IN ('GNS', 'ANC', 'SNT', 'MNC');

    IF council_count < 4 THEN
        -- Find which ones are missing
        SELECT STRING_AGG(expected.code, ', ')
        INTO missing_codes
        FROM (VALUES ('GNS'), ('ANC'), ('SNT'), ('MNC')) AS expected(code)
        LEFT JOIN councils c ON c.code = expected.code
        WHERE c.code IS NULL;

        RAISE EXCEPTION 'ABORT: Missing councils after insert: %. Rolling back.', missing_codes;
    END IF;

    RAISE NOTICE 'SUCCESS: All 4 councils verified:';
    RAISE NOTICE '  GNS — Gymkhana and Sports';
    RAISE NOTICE '  ANC — Arts and Cultural';
    RAISE NOTICE '  SNT — Science and Technology';
    RAISE NOTICE '  MNC — Music and Cultural';
END;
$$;

COMMIT;

-- =========================================================================
-- Post-run verification query:
--   SELECT id, code, name, created_at FROM councils ORDER BY code;
-- =========================================================================