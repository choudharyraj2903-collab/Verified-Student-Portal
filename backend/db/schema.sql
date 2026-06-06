-- =========================================================================
-- SECURITY-FIRST AUTH SYSTEM DATABASE SCHEMA
-- Optimized Version
-- =========================================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Clean up existing tables and types if recreating
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS device_trust CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS invalidation_tokens CASCADE;
DROP TABLE IF EXISTS magic_tokens CASCADE;
DROP TABLE IF EXISTS verification_requests CASCADE;
DROP TABLE IF EXISTS profiles CASCADE;
DROP TABLE IF EXISTS user_council_scopes CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS councils CASCADE;
DROP TYPE IF EXISTS verification_status_enum CASCADE;
DROP TYPE IF EXISTS user_role_enum CASCADE;
DROP TYPE IF EXISTS audit_severity_enum CASCADE;
DROP TYPE IF EXISTS audit_event_enum CASCADE;

-- =========================================================================
-- ENUMS
-- =========================================================================

CREATE TYPE user_role_enum AS ENUM (
    'STUDENT',
    'COUNCIL_ADMIN',
    'SUPER_ADMIN'
);

CREATE TYPE audit_severity_enum AS ENUM (
    'INFO',
    'WARN',
    'CRITICAL'
);

CREATE TYPE audit_event_enum AS ENUM (
    'MAGIC_LINK_REQUESTED',
    'MAGIC_LINK_CLICKED',
    'MAGIC_LINK_EXPIRED',
    'MAGIC_LINK_REUSE_ATTEMPT',
    'LOGIN_SUCCESS',
    'LOGIN_FAILED',
    'SESSION_INVALIDATED',
    'SESSION_INVALIDATED_BY_USER',
    'SESSION_CONFIRMED_BY_USER',
    'TOKEN_ROTATION',
    'TOKEN_REUSE_DETECTED',
    'DEVICE_TRUSTED',
    'DEVICE_REVOKED',
    'ROLE_CHANGED',
    'COUNCIL_SCOPE_ASSIGNED',
    'COUNCIL_SCOPE_REMOVED',
    'LOGOUT',
    'LOGOUT_ALL_DEVICES',
    'REPEATED_INVALIDATION_FLAGGED',
    'UNAUTHORIZED_SCOPE_ACCESS',
    'UNAUTHORIZED_ROLE_ACCESS',
    'RATE_LIMIT_HIT',
    'RATE_LIMIT_IP_BLOCK',
    'CAMPUS_IP_GUARD_BLOCK',
    'PROFILE_CREATED',
    'PROFILE_UPDATED',
    'VERIFICATION_SUBMITTED',
    'VERIFICATION_APPROVED',
    'VERIFICATION_REJECTED',
    'VERIFICATION_REQUEST_SUBMITTED',
    'VERIFICATION_REQUEST_APPROVED',
    'VERIFICATION_REQUEST_REJECTED',
    'ACCOUNT_DEACTIVATED'
);

CREATE TYPE verification_status_enum AS ENUM (
    'PENDING',
    'APPROVED',
    'REJECTED'
);

-- =========================================================================
-- 1. COUNCILS TABLE
-- =========================================================================

CREATE TABLE councils (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    code       VARCHAR(10)  UNIQUE NOT NULL,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- =========================================================================
-- 2. USERS TABLE
-- =========================================================================

CREATE TABLE users (
    id         UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    email      VARCHAR(255)   UNIQUE NOT NULL,
    role       user_role_enum NOT NULL DEFAULT 'STUDENT',
    is_active  BOOLEAN        NOT NULL DEFAULT TRUE,
    deactivation_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    CONSTRAINT check_normalized_email CHECK (email = LOWER(TRIM(email))),
    CONSTRAINT check_email_domain     CHECK (email LIKE '%@iitk.ac.in')
);

CREATE INDEX idx_users_email  ON users(email);
CREATE INDEX idx_users_role   ON users(role);
CREATE INDEX idx_users_active ON users(is_active);

-- =========================================================================
-- 3. PROFILES TABLE
-- =========================================================================

CREATE TABLE profiles (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID         UNIQUE REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    full_name   VARCHAR(255) NOT NULL,
    roll_number VARCHAR(50)  UNIQUE NOT NULL,
    year        INTEGER      NOT NULL CHECK (year BETWEEN 1 AND 5),
    branch      VARCHAR(100) NOT NULL,
    phone       VARCHAR(30),
    avatar_url  TEXT,
    bio         TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_profiles_user_id     ON profiles(user_id);
CREATE INDEX idx_profiles_roll_number ON profiles(roll_number);

-- =========================================================================
-- 4. USER COUNCIL SCOPES
-- =========================================================================

CREATE TABLE user_council_scopes (
    user_id     UUID REFERENCES users(id)    ON DELETE CASCADE,
    council_id  UUID REFERENCES councils(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id)    ON DELETE SET NULL,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at  TIMESTAMP WITH TIME ZONE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (user_id, council_id)
);

CREATE INDEX idx_council_scopes_user   ON user_council_scopes(user_id);
CREATE INDEX idx_council_scopes_expiry ON user_council_scopes(expires_at)
    WHERE expires_at IS NOT NULL;

-- =========================================================================
-- 5. VERIFICATION REQUESTS TABLE
-- =========================================================================

CREATE TABLE verification_requests (
    id          UUID                     PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID                     REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    council_id  UUID                     REFERENCES councils(id) ON DELETE RESTRICT NOT NULL,
    title       VARCHAR(255)             NOT NULL,
    description TEXT                     NOT NULL,
    proof_link  TEXT                     NOT NULL,
    por_date    DATE                     NOT NULL,
    status      verification_status_enum NOT NULL DEFAULT 'PENDING',
    remarks     TEXT,
    reviewed_by UUID                     REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    CONSTRAINT check_proof_link_not_blank CHECK (LENGTH(TRIM(proof_link)) > 0)
);

CREATE INDEX idx_verification_user_status    ON verification_requests(user_id, status);
CREATE INDEX idx_verification_council_status ON verification_requests(council_id, status);
CREATE INDEX idx_verification_created_at     ON verification_requests(created_at DESC);

-- =========================================================================
-- 6. MAGIC TOKENS TABLE
-- =========================================================================

CREATE TABLE magic_tokens (
    id              UUID     PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID     REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    token_hash      CHAR(64) UNIQUE NOT NULL,
    user_agent_hash CHAR(64) NOT NULL,
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    is_used         BOOLEAN  NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_magic_tokens_verification
    ON magic_tokens(token_hash, is_used, expires_at);

CREATE INDEX idx_magic_tokens_user
    ON magic_tokens(user_id, created_at);

-- =========================================================================
-- 7. INVALIDATION TOKENS TABLE
-- =========================================================================

CREATE TABLE invalidation_tokens (
    id                   UUID     PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id              UUID     REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    refresh_token_family UUID     NOT NULL,
    token_hash           CHAR(64) UNIQUE NOT NULL,
    is_used              BOOLEAN  NOT NULL DEFAULT FALSE,
    expires_at           TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at           TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_invalidation_tokens_lookup
    ON invalidation_tokens(token_hash, is_used, expires_at);

CREATE INDEX idx_invalidation_tokens_family
    ON invalidation_tokens(refresh_token_family);

-- =========================================================================
-- 8. REFRESH TOKENS TABLE
-- =========================================================================

CREATE TABLE refresh_tokens (
    id               UUID     PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID     REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    token_hash       CHAR(64) UNIQUE NOT NULL,
    family_id        UUID     NOT NULL,
    fingerprint_hash CHAR(64) NOT NULL,
    is_revoked       BOOLEAN  NOT NULL DEFAULT FALSE,
    expires_at       TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_refresh_tokens_lookup
    ON refresh_tokens(token_hash, is_revoked);

CREATE INDEX idx_refresh_tokens_family
    ON refresh_tokens(family_id);

CREATE INDEX idx_refresh_tokens_user
    ON refresh_tokens(user_id, is_revoked);

-- =========================================================================
-- 9. DEVICE TRUST TABLE
-- =========================================================================

CREATE TABLE device_trust (
    id                UUID     PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id           UUID     REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    device_token_hash CHAR(64) UNIQUE NOT NULL,
    fingerprint_hash  CHAR(64) NOT NULL,
    is_revoked        BOOLEAN  NOT NULL DEFAULT FALSE,
    expires_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX idx_user_device_fingerprint
    ON device_trust(user_id, fingerprint_hash)
    WHERE is_revoked = FALSE;

CREATE INDEX idx_device_trust_token
    ON device_trust(device_token_hash, is_revoked);

CREATE INDEX idx_device_trust_expiry
    ON device_trust(expires_at)
    WHERE is_revoked = FALSE;

-- =========================================================================
-- 10. AUDIT LOGS TABLE
-- =========================================================================

CREATE TABLE audit_logs (
    id         UUID                PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID                REFERENCES users(id) ON DELETE SET NULL,
    event_type audit_event_enum    NOT NULL,
    severity   audit_severity_enum NOT NULL,
    metadata   JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_audit_user_time
    ON audit_logs(user_id, created_at DESC);

CREATE INDEX idx_audit_event_time
    ON audit_logs(event_type, created_at DESC);

CREATE INDEX idx_audit_severity_time
    ON audit_logs(severity, created_at DESC);

-- =========================================================================
-- TRIGGERS
-- =========================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_profiles_updated_at
    BEFORE UPDATE ON profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_verification_requests_updated_at
    BEFORE UPDATE ON verification_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
