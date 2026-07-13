CREATE TABLE IF NOT EXISTS users (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    sso_id      VARCHAR(128)    NOT NULL COMMENT 'Logto 用户 ID',
    username    VARCHAR(64)     NOT NULL,
    email       VARCHAR(128)    NOT NULL,
    role        ENUM('admin','user') NOT NULL DEFAULT 'user',
    status      ENUM('active','disabled') NOT NULL DEFAULT 'active',
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at  DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_sso_id (sso_id),
    UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS domains (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    host        VARCHAR(253)    NOT NULL COMMENT '域名，不含协议和路径，如 go.example.com',
    type        ENUM('entry','transit','landing') NOT NULL COMMENT '入口/中转/落地',
    scheme      ENUM('http','https') NOT NULL DEFAULT 'https',
    remark      VARCHAR(255)    NULL COMMENT '备注',
    status      ENUM('active','disabled') NOT NULL DEFAULT 'active',
    created_by  BIGINT UNSIGNED NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at  DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_host (host)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS links (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code              VARCHAR(32)     NOT NULL COMMENT '短码，全局唯一',
    type              ENUM('short','channel','liveqr') NOT NULL,
    entry_domain_id   BIGINT UNSIGNED NOT NULL COMMENT '入口域名',
    transit_domain_id BIGINT UNSIGNED NULL COMMENT '中转域名，NULL 表示不经中转',
    landing_domain_id BIGINT UNSIGNED NULL COMMENT '落地域名，NULL 表示直跳',
    target_url        TEXT            NULL COMMENT '直跳目标 URL（type=short 且无落地页时使用）',
    landing_page_id   BIGINT UNSIGNED NULL COMMENT '落地页 ID，NULL 表示直跳',
    title             VARCHAR(255)    NULL COMMENT '链接名称/备注',
    expire_at         DATETIME        NULL COMMENT 'NULL 表示永不过期',
    status            ENUM('active','disabled','expired') NOT NULL DEFAULT 'active',
    created_by        BIGINT UNSIGNED NOT NULL,
    created_at        DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at        DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at        DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_code (code),
    KEY idx_type (type),
    KEY idx_created_by (created_by),
    KEY idx_entry_domain (entry_domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS channel_configs (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id      BIGINT UNSIGNED NOT NULL,
    utm_source   VARCHAR(128)    NULL,
    utm_medium   VARCHAR(128)    NULL,
    utm_campaign VARCHAR(128)    NULL,
    utm_term     VARCHAR(128)    NULL,
    utm_content  VARCHAR(128)    NULL,
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_id (link_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS routing_strategies (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    mode        ENUM('round_robin','weighted','time_based','geo_device') NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_id (link_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS routing_targets (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    strategy_id   BIGINT UNSIGNED NOT NULL,
    label         VARCHAR(128)    NULL COMMENT '目标标识，如"群1"',
    target_url    TEXT            NOT NULL COMMENT '目标 URL 或落地页',
    weight        INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '权重，用于 weighted 模式',
    scan_limit    INT UNSIGNED    NULL COMMENT '扫码次数上限，NULL 表示无限制',
    scan_count    INT UNSIGNED    NOT NULL DEFAULT 0 COMMENT '当前已扫次数',
    time_start    TIME            NULL COMMENT '生效开始时间（每日循环），NULL 表示不限',
    time_end      TIME            NULL COMMENT '生效结束时间（每日循环）',
    weekday_mask  TINYINT UNSIGNED NULL COMMENT '生效星期位掩码（bit0=周一...bit6=周日），NULL 表示不限',
    geo_filter    JSON            NULL COMMENT '地域过滤，如 {"provinces": ["广东","北京"]}',
    device_filter JSON            NULL COMMENT '设备过滤，如 {"devices": ["mobile"]}',
    priority      INT             NOT NULL DEFAULT 0 COMMENT '优先级，高优先级先匹配',
    status        ENUM('active','disabled','exhausted') NOT NULL DEFAULT 'active',
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_strategy_id (strategy_id),
    KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS landing_pages (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    template    ENUM('liveqr','redirect_notice','custom') NOT NULL,
    title       VARCHAR(255)    NOT NULL COMMENT '页面标题（显示在浏览器标签）',
    content     JSON            NOT NULL COMMENT '模板内容',
    domain_id   BIGINT UNSIGNED NOT NULL COMMENT '使用哪个落地域名渲染',
    created_by  BIGINT UNSIGNED NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at  DATETIME(3)     NULL,
    PRIMARY KEY (id),
    KEY idx_domain_id (domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS access_logs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    visited_at  DATETIME(3)     NOT NULL,
    ip          VARCHAR(45)     NOT NULL COMMENT 'IPv4 或 IPv6',
    country     VARCHAR(64)     NULL,
    province    VARCHAR(64)     NULL,
    city        VARCHAR(64)     NULL,
    isp         VARCHAR(64)     NULL,
    device      ENUM('mobile','tablet','desktop','bot','unknown') NOT NULL DEFAULT 'unknown',
    os          VARCHAR(64)     NULL,
    browser     VARCHAR(64)     NULL,
    referer     VARCHAR(2048)   NULL,
    via_transit TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '是否经过中转域名',
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_link_visited (link_id, visited_at),
    KEY idx_visited_at (visited_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS access_logs_archive LIKE access_logs;

CREATE TABLE IF NOT EXISTS stat_daily (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    stat_date   DATE            NOT NULL,
    pv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    uv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    ip_count    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_date (link_id, stat_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS stat_hourly (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    stat_date   DATE            NOT NULL,
    stat_hour   TINYINT UNSIGNED NOT NULL COMMENT '0-23',
    pv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_date_hour (link_id, stat_date, stat_hour)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS stat_geo (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    stat_date   DATE            NOT NULL,
    country     VARCHAR(64)     NOT NULL DEFAULT '',
    province    VARCHAR(64)     NOT NULL DEFAULT '',
    pv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_date_geo (link_id, stat_date, country, province)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS stat_device (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    stat_date   DATE            NOT NULL,
    device      VARCHAR(32)     NOT NULL DEFAULT '',
    os          VARCHAR(64)     NOT NULL DEFAULT '',
    browser     VARCHAR(64)     NOT NULL DEFAULT '',
    pv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_date_device (link_id, stat_date, device, os, browser)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS system_configs (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    key_name     VARCHAR(128)    NOT NULL,
    value        TEXT            NOT NULL,
    description  VARCHAR(255)    NULL,
    updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_key_name (key_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
