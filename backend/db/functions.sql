-- =========================================================================
-- PURGE & MAINTENANCE FUNCTIONS
-- Called by Go job scheduler (go-cron) — never called directly by app logic
-- =========================================================================

-- -------------------------------------------------------------------------
-- Purge expired and used magic link tokens
-- Schedule: every hour
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_expired_magic_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM magic_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
       OR is_used = TRUE;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Purge expired and used invalidation tokens
-- Schedule: every hour
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_expired_invalidation_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM invalidation_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
       OR is_used = TRUE;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Purge expired refresh tokens
-- Schedule: daily
-- 24h buffer kept for non-revoked tokens to avoid race conditions on slow clients
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_expired_refresh_tokens()
RETURNS void AS $$
BEGIN
    -- Revoked tokens: delete immediately on expiry
    DELETE FROM refresh_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
      AND is_revoked = TRUE;

    -- Non-revoked tokens: hard delete after 24h grace buffer
    DELETE FROM refresh_tokens
    WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '24 hours';
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Purge expired device trust entries
-- Schedule: daily
-- Same 24h grace buffer pattern as refresh tokens
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION purge_expired_device_trust()
RETURNS void AS $$
BEGIN
    DELETE FROM device_trust
    WHERE expires_at < CURRENT_TIMESTAMP
      AND is_revoked = TRUE;

    DELETE FROM device_trust
    WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '24 hours';
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------------------
-- Soft-expire council scopes past their expiry date
-- Schedule: daily
-- Does not hard delete — preserves history, marks is_active = FALSE only
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION expire_council_scopes()
RETURNS void AS $$
BEGIN
    UPDATE user_council_scopes
    SET is_active = FALSE
    WHERE expires_at IS NOT NULL
      AND expires_at < CURRENT_TIMESTAMP
      AND is_active = TRUE;
END;
$$ LANGUAGE plpgsql;