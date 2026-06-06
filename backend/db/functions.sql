-- =========================================================================
-- PURGE AND MAINTENANCE FUNCTIONS
-- Called by Go job scheduler (go-cron) — never called by app request handlers
-- Each function is independently schedulable
-- =========================================================================

-- -------------------------------------------------------------------------
-- Purge used and expired magic link tokens
-- Schedule: every hour
-- Reason: magic tokens have 15 min expiry — hour-old used tokens have no value
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_expired_magic_tokens()
RETURNS void AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM magic_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
       OR is_used = TRUE;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RAISE NOTICE 'purge_expired_magic_tokens: removed % rows', deleted_count;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Purge used and expired invalidation tokens
-- Schedule: every hour
-- Reason: 1 hour expiry — anything past that is useless
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_expired_invalidation_tokens()
RETURNS void AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM invalidation_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
       OR is_used = TRUE;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RAISE NOTICE 'purge_expired_invalidation_tokens: removed % rows', deleted_count;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Purge expired refresh tokens
-- Schedule: daily at 02:00
-- Two-pass strategy:
--   Pass 1 — revoked tokens deleted immediately on expiry
--   Pass 2 — non-revoked tokens deleted after 24h grace buffer
--            (avoids race condition on slow clients mid-rotation)
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_expired_refresh_tokens()
RETURNS void AS $$
DECLARE
    deleted_revoked  INTEGER;
    deleted_grace    INTEGER;
BEGIN
    -- Pass 1 — revoked tokens past expiry
    DELETE FROM refresh_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
      AND is_revoked = TRUE;
    GET DIAGNOSTICS deleted_revoked = ROW_COUNT;

    -- Pass 2 — all tokens past 24h grace window
    DELETE FROM refresh_tokens
    WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '24 hours';
    GET DIAGNOSTICS deleted_grace = ROW_COUNT;

    RAISE NOTICE 'purge_expired_refresh_tokens: removed % revoked, % grace-expired',
        deleted_revoked, deleted_grace;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Purge expired device trust entries
-- Schedule: daily at 02:30
-- Same two-pass strategy as refresh tokens
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_expired_device_trust()
RETURNS void AS $$
DECLARE
    deleted_revoked INTEGER;
    deleted_grace   INTEGER;
BEGIN
    DELETE FROM device_trust
    WHERE expires_at < CURRENT_TIMESTAMP
      AND is_revoked = TRUE;
    GET DIAGNOSTICS deleted_revoked = ROW_COUNT;

    DELETE FROM device_trust
    WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '24 hours';
    GET DIAGNOSTICS deleted_grace = ROW_COUNT;

    RAISE NOTICE 'purge_expired_device_trust: removed % revoked, % grace-expired',
        deleted_revoked, deleted_grace;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Soft-expire council scopes past their expiry date
-- Schedule: daily at 03:00
-- Does NOT hard delete — preserves history for audit purposes
-- Sets is_active = FALSE only
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION expire_council_scopes()
RETURNS void AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    UPDATE user_council_scopes
    SET is_active = FALSE
    WHERE expires_at IS NOT NULL
      AND expires_at < CURRENT_TIMESTAMP
      AND is_active = TRUE;

    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RAISE NOTICE 'expire_council_scopes: soft-expired % scope rows', updated_count;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Purge old rejected verification requests
-- Schedule: weekly on Sunday at 04:00
-- Rejected requests older than 90 days are removed
-- Approved requests are NEVER purged — permanent record
-- Pending requests are NEVER purged — awaiting review
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_old_rejected_requests()
RETURNS void AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM verification_requests
    WHERE status = 'REJECTED'
      AND updated_at < CURRENT_TIMESTAMP - INTERVAL '90 days';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RAISE NOTICE 'purge_old_rejected_requests: removed % rows', deleted_count;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Utility — get current pool and table statistics
-- Called manually by Super Admin or monitoring scripts
-- Not scheduled — on-demand only
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION get_system_stats()
RETURNS TABLE (
    table_name  TEXT,
    row_count   BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 'users'::TEXT,                  COUNT(*) FROM users
    UNION ALL
    SELECT 'profiles'::TEXT,               COUNT(*) FROM profiles
    UNION ALL
    SELECT 'verification_requests'::TEXT,  COUNT(*) FROM verification_requests
    UNION ALL
    SELECT 'magic_tokens'::TEXT,           COUNT(*) FROM magic_tokens
    UNION ALL
    SELECT 'refresh_tokens'::TEXT,         COUNT(*) FROM refresh_tokens
    UNION ALL
    SELECT 'device_trust'::TEXT,           COUNT(*) FROM device_trust
    UNION ALL
    SELECT 'invalidation_tokens'::TEXT,    COUNT(*) FROM invalidation_tokens
    UNION ALL
    SELECT 'audit_logs'::TEXT,             COUNT(*) FROM audit_logs
    UNION ALL
    SELECT 'user_council_scopes'::TEXT,    COUNT(*) FROM user_council_scopes;
END;
$$ LANGUAGE plpgsql;