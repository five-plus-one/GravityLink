# 数据模型（Database Schema）

## 设计原则

- 表名：小写下划线，无前缀
- 所有表必须有 `id`（BIGINT AUTO_INCREMENT）、`created_at`、`updated_at`
- 软删除：业务核心表使用 `deleted_at`（GORM 软删除），日志表不需要
- 统计表与业务表严格分离，不在业务表中存统计字段

## 完整 DDL

### 用户与权限

```sql
CREATE TABLE users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    auth_source   ENUM('logto','local') NOT NULL,
    sso_id        VARCHAR(128)    NULL COMMENT 'Logto 用户 ID',
    username      VARCHAR(64)     NOT NULL,
    email         VARCHAR(128)    NULL,
    password_hash VARCHAR(255)    NULL COMMENT '本地账号 Argon2id/bcrypt 哈希',
    role          ENUM('super_admin','admin','user') NOT NULL DEFAULT 'user',
    status        ENUM('pending','active','disabled') NOT NULL DEFAULT 'pending',
    last_login_at DATETIME(3)     NULL,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at    DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_sso_id (sso_id),
    UNIQUE KEY uk_username (username),
    UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE auth_sessions (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id     BIGINT UNSIGNED NOT NULL,
    token_hash  CHAR(64)        NOT NULL,
    expires_at  DATETIME(3)     NOT NULL,
    last_seen_at DATETIME(3)    NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_token_hash (token_hash),
    KEY idx_session_user (user_id),
    KEY idx_session_expire (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE installation_state (
    id              TINYINT UNSIGNED NOT NULL,
    installed       TINYINT(1)       NOT NULL DEFAULT 0,
    owner_user_id   BIGINT UNSIGNED  NULL,
    schema_version  INT UNSIGNED     NOT NULL DEFAULT 1,
    installed_at    DATETIME(3)      NULL,
    updated_at      DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE audit_logs (
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
```

权限判断以 `users` 表为最终来源。Logto JWT 只证明身份，不直接授予后台角色。首个通过初始化认证的账号在数据库事务中认领 `super_admin`；后续 Logto 新用户默认进入 `pending`。

### 域名管理

```sql
-- 域名配置（三种类型：入口、中转、落地）
CREATE TABLE domains (
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
```

### 链接核心表

```sql
-- 链接主表（短链接、渠道码、群活码共用）
CREATE TABLE links (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code            VARCHAR(32)     NOT NULL COMMENT '短码，全局唯一',
    type            ENUM('short','channel','liveqr') NOT NULL,
    entry_domain_id BIGINT UNSIGNED NOT NULL COMMENT '入口域名',
    transit_domain_id BIGINT UNSIGNED NULL COMMENT '中转域名，NULL 表示不经中转',
    landing_domain_id BIGINT UNSIGNED NULL COMMENT '落地域名，NULL 表示直跳',
    target_url      TEXT            NULL COMMENT '直跳目标 URL（type=short 且无落地页时使用）',
    landing_page_id BIGINT UNSIGNED NULL COMMENT '落地页 ID，NULL 表示直跳',
    title           VARCHAR(255)    NULL COMMENT '链接名称/备注',
    expire_at       DATETIME        NULL COMMENT 'NULL 表示永不过期',
    status          ENUM('active','disabled','expired') NOT NULL DEFAULT 'active',
    created_by      BIGINT UNSIGNED NOT NULL,
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at      DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_code (code),
    KEY idx_type (type),
    KEY idx_created_by (created_by),
    KEY idx_entry_domain (entry_domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 渠道码

```sql
-- 渠道码配置（附加在 links 上的渠道参数）
CREATE TABLE channel_configs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    utm_source  VARCHAR(128)    NULL,
    utm_medium  VARCHAR(128)    NULL,
    utm_campaign VARCHAR(128)   NULL,
    utm_term    VARCHAR(128)    NULL,
    utm_content VARCHAR(128)    NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_id (link_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 群活码路由策略

```sql
-- 轮换策略主表（一个群活码对应一个策略）
CREATE TABLE routing_strategies (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    mode        ENUM('round_robin','weighted','time_based','geo_device') NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_id (link_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 策略目标（每个策略可有多个目标，如多个群二维码）
CREATE TABLE routing_targets (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    strategy_id     BIGINT UNSIGNED NOT NULL,
    label           VARCHAR(128)    NULL COMMENT '目标标识，如"群1"',
    target_url      TEXT            NOT NULL COMMENT '目标 URL 或落地页',
    weight          INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '权重，用于 weighted 模式',
    scan_limit      INT UNSIGNED    NULL COMMENT '扫码次数上限，NULL 表示无限制',
    scan_count      INT UNSIGNED    NOT NULL DEFAULT 0 COMMENT '当前已扫次数',
    time_start      TIME            NULL COMMENT '生效开始时间（每日循环），NULL 表示不限',
    time_end        TIME            NULL COMMENT '生效结束时间（每日循环）',
    weekday_mask    TINYINT UNSIGNED NULL COMMENT '生效星期位掩码（bit0=周一...bit6=周日），NULL 表示不限',
    geo_filter      JSON            NULL COMMENT '地域过滤，如 {"provinces": ["广东","北京"]}',
    device_filter   JSON            NULL COMMENT '设备过滤，如 {"devices": ["mobile"]}',
    priority        INT             NOT NULL DEFAULT 0 COMMENT '优先级，高优先级先匹配',
    status          ENUM('active','disabled','exhausted') NOT NULL DEFAULT 'active',
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_strategy_id (strategy_id),
    KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 落地页

```sql
-- 落地页模板类型（系统内置，可扩展）
-- 枚举值：liveqr（群活码展示）| redirect_notice（跳转提示）| custom（自定义 HTML）

-- 落地页实例
CREATE TABLE landing_pages (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    template    ENUM('liveqr','redirect_notice','custom') NOT NULL,
    title       VARCHAR(255)    NOT NULL COMMENT '页面标题（显示在浏览器标签）',
    content     JSON            NOT NULL COMMENT '模板内容，结构见 features/landing-page.md',
    domain_id   BIGINT UNSIGNED NOT NULL COMMENT '使用哪个落地域名渲染',
    created_by  BIGINT UNSIGNED NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at  DATETIME(3)     NULL,
    PRIMARY KEY (id),
    KEY idx_domain_id (domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 统计表

```sql
-- 原始访问日志（异步写入，保留 90 天后归档）
CREATE TABLE access_logs (
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

-- 按天聚合（PV / UV / IP 数）
CREATE TABLE stat_daily (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    stat_date   DATE            NOT NULL,
    pv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    uv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    ip_count    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_date (link_id, stat_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 按小时聚合
CREATE TABLE stat_hourly (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    stat_date   DATE            NOT NULL,
    stat_hour   TINYINT UNSIGNED NOT NULL COMMENT '0-23',
    pv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_date_hour (link_id, stat_date, stat_hour)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 地域聚合（按天）
CREATE TABLE stat_geo (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    link_id     BIGINT UNSIGNED NOT NULL,
    stat_date   DATE            NOT NULL,
    country     VARCHAR(64)     NOT NULL DEFAULT '',
    province    VARCHAR(64)     NOT NULL DEFAULT '',
    pv          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_link_date_geo (link_id, stat_date, country, province)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 设备聚合（按天）
CREATE TABLE stat_device (
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
```

### 系统配置

```sql
-- 系统全局配置（KV 形式）
CREATE TABLE system_configs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    key_name    VARCHAR(128)    NOT NULL,
    value       TEXT            NOT NULL,
    description VARCHAR(255)    NULL,
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_key_name (key_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## Redis Key 设计

| Key 模式 | 类型 | TTL | 说明 |
|---------|------|-----|------|
| `link:cache:{code}` | Hash | 24h | 短码映射缓存（自动续期） |
| `link:code:set` | Set | 永久 | 已使用短码集合（用于生成时去重） |
| `stat:pv:{link_id}:{yyyymmdd}` | String | 48h | 当日 PV 计数 |
| `stat:uv:{link_id}:{yyyymmdd}` | HyperLogLog | 48h | 当日 UV 估算 |
| `stat:hourly:{link_id}:{yyyymmdd}:{hh}` | String | 48h | 小时 PV |
| `access:stream` | List | - | 访问日志队列（LPUSH/RPOP） |
| `routing:scan:{target_id}` | String | 永久 | 活码目标实时扫码计数 |

## 表关系图

```
users ──────────────────── links (created_by)
                              │
domains (entry) ─────────── links (entry_domain_id)
domains (transit) ────────── links (transit_domain_id)
domains (landing) ────────── links (landing_domain_id)
                              │
landing_pages ───────────── links (landing_page_id)
                              │
                    ┌─────────┴──────────┐
              channel_configs     routing_strategies
                                        │
                                 routing_targets
                              │
access_logs ─────────────── links (link_id)
stat_daily  ─────────────── links (link_id)
stat_hourly ─────────────── links (link_id)
stat_geo    ─────────────── links (link_id)
stat_device ─────────────── links (link_id)
```
