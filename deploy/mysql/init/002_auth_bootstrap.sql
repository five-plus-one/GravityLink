ALTER TABLE users
    ADD COLUMN IF NOT EXISTS auth_source ENUM('logto','local') NOT NULL DEFAULT 'logto' AFTER id,
    MODIFY COLUMN sso_id VARCHAR(128) NULL,
    MODIFY COLUMN email VARCHAR(128) NULL,
    ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NULL AFTER email,
    MODIFY COLUMN role ENUM('super_admin','admin','user') NOT NULL DEFAULT 'user',
    MODIFY COLUMN status ENUM('pending','active','disabled') NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS last_login_at DATETIME(3) NULL AFTER status;

SET @username_index_exists = (
    SELECT COUNT(1)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'users' AND index_name = 'uk_username'
);
SET @username_index_sql = IF(
    @username_index_exists = 0,
    'CREATE UNIQUE INDEX uk_username ON users (username)',
    'SELECT 1'
);
PREPARE username_index_stmt FROM @username_index_sql;
EXECUTE username_index_stmt;
DEALLOCATE PREPARE username_index_stmt;

CREATE TABLE IF NOT EXISTS auth_sessions (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id      BIGINT UNSIGNED NOT NULL,
    token_hash   CHAR(64)        NOT NULL,
    expires_at   DATETIME(3)     NOT NULL,
    last_seen_at DATETIME(3)     NULL,
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_token_hash (token_hash),
    KEY idx_session_user (user_id),
    KEY idx_session_expire (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS installation_state (
    id              TINYINT UNSIGNED NOT NULL,
    installed       TINYINT(1)       NOT NULL DEFAULT 0,
    owner_user_id   BIGINT UNSIGNED  NULL,
    schema_version  INT UNSIGNED     NOT NULL DEFAULT 1,
    installed_at    DATETIME(3)      NULL,
    updated_at      DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id     BIGINT UNSIGNED NULL,
    action      VARCHAR(64)     NOT NULL,
    target_type VARCHAR(64)     NULL,
    target_id   VARCHAR(128)    NULL,
    detail      JSON            NULL,
    ip          VARCHAR(45)     NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_audit_user (user_id),
    KEY idx_audit_action (action),
    KEY idx_audit_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
