# 系统架构

## 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                          Internet                           │
└──────────┬──────────────────────┬──────────────────────────┘
           │                      │
    入口/中转/落地域名          管理/API 域名
           │                      │
┌──────────▼──────────────────────▼──────────────────────────┐
│                     Nginx（TLS 终止）                        │
│  管理域名：仅允许内网 IP 访问（allow 10.0.0.0/8; deny all）    │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                   GravityLink（单 Go 进程）                   │
│                                                             │
│  ┌─────────────────── Gin Router ─────────────────────┐    │
│  │                                                     │    │
│  │  Host 中间件                                        │    │
│  │  ├── 入口域名  → Redirect Handler                   │    │
│  │  ├── 中转域名  → Transit Handler                    │    │
│  │  ├── 落地域名  → Page Handler                       │    │
│  │  ├── /assets/landing/* → 公网落地页静态资源         │    │
│  │  ├── /api/admin/* → Admin API（需 Admin 角色）      │    │
│  │  └── /api/v1/*   → 管理 API（需登录）               │    │
│  │                                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │  Domain      │  │  Link        │  │  Stat Worker    │   │
│  │  Service     │  │  Service     │  │  (goroutine)    │   │
│  └──────────────┘  └──────────────┘  └────────┬────────┘   │
│                                               │             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  公网落地页资源 + 管理端 Vue build（go:embed）        │   │
│  └──────────────────────────────────────────────────────┘   │
└───────────────┬──────────────────────┬──────────────────────┘
                │                      │
        ┌───────▼──────┐      ┌────────▼───────┐
        │   MySQL 8    │      │    Redis 7      │
        │  主业务数据   │      │  缓存 + 统计队列 │
        └──────────────┘      └────────────────┘
                                       │
                              ┌────────▼───────┐
                              │  Logto（OIDC）  │
                              │  用户已有部署    │
                              └────────────────┘
```

## 模块划分

### 后端目录结构

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # 服务入口
├── internal/
│   ├── config/                  # 配置加载（.env / yaml）
│   ├── database/                # MySQL + Redis 连接初始化
│   ├── middleware/
│   │   ├── host_router.go       # 按 Host 头分发到不同处理器
│   │   ├── auth.go              # Logto JWT 验证 + RBAC
│   │   └── rate_limit.go        # 重定向接口限流
│   ├── router/
│   │   ├── redirect/            # 入口域名：短链跳转逻辑
│   │   ├── transit/             # 中转域名：中转跳转逻辑
│   │   ├── page/                # 落地域名：落地页渲染
│   │   ├── admin/               # 管理 API（/api/admin/*）
│   │   └── open/                # 用户 API（/api/v1/*）
│   ├── web/                     # 公网落地页资源 + 管理端构建产物（go:embed）
│   │   └── landing/
│   │   └── admin/
│   ├── service/
│   │   ├── link.go              # 短链接 / 渠道码 / 群活码 业务逻辑
│   │   ├── routing.go           # 轮换策略引擎
│   │   ├── domain.go            # 域名管理
│   │   ├── landing.go           # 落地页模板渲染
│   │   └── stat.go              # 统计查询
│   ├── model/                   # GORM 数据模型
│   ├── cache/                   # Redis 操作封装
│   └── worker/
│       ├── stat_flush.go        # 定时将 Redis 统计落盘 MySQL
│       └── log_consumer.go      # 消费 Redis List 中的访问日志
├── pkg/
│   ├── base62/                  # Base62 编解码
│   ├── ipdb/                    # ip2region 封装
│   └── ua/                      # User-Agent 解析封装
```

### 前端目录结构

```
frontend/
└── admin/                       # 管理端 Vue 3 项目（唯一前端工程）
    ├── src/
    │   ├── layouts/
    │   │   └── AdminLayout.vue  # 侧栏 + 顶栏 + 面包屑布局
    │   ├── views/               # 全部页面（平铺，无子目录）
    │   │   ├── LoginView.vue
    │   │   ├── SetupView.vue    # 初始化向导
    │   │   ├── AuthCallbackView.vue
    │   │   ├── NotFoundView.vue
    │   │   ├── DashboardView.vue
    │   │   ├── LinksView.vue
    │   │   ├── DomainsView.vue
    │   │   ├── LandingPagesView.vue
    │   │   ├── StatsView.vue
    │   │   ├── UsersView.vue
    │   │   └── SettingsView.vue
    │   ├── stores/              # Pinia 状态（auth / domain / setup）
    │   ├── styles/              # 设计令牌 + 基础样式
    │   │   ├── tokens.css       # CSS 变量（详见 Docs/ui-design-system.md）
    │   │   └── base.css
    │   ├── api.ts               # API 客户端
    │   ├── auth.ts              # OIDC / 本地认证
    │   ├── setup.ts             # 初始化向导 API
    │   ├── echarts.ts           # ECharts 按需引入
    │   ├── router.ts            # 路由 + 全局守卫
    │   └── main.ts
    └── package.json             # Vue 3 + Naive UI + Pinia + ECharts
```

落地页与公开页**没有独立前端工程**：模板由 Go `html/template` 渲染（`backend/internal/web/templates/`），共享样式由 `backend/internal/web/landing/gravitylink-landing.css` 通过 `go:embed` 提供。

### 前端边界

- `frontend/admin` 是管理端，只面向运营/管理员，不应作为公网用户入口开放；Docker 镜像构建时会将其打包进 Go 二进制。
- 公网用户访问短链接时直接进入 Go 后端公开路由：入口域名负责 302 跳转，落地域名负责渲染模板页。
- 应用容器默认监听两个端口：`8080` 为后端 API/公网短链路由，`8081` 为管理端前端；管理端端口内的 `/api/*` 会转给同一个后端 handler。
- 管理端通过 Logto OIDC Authorization Code + PKCE 登录，不提供生产可用的手工 token 输入；后端按 JWT 与角色声明执行 RBAC。
- 短链接直跳不需要单独公网 SPA；落地页与公开页由 Go `html/template` 渲染（`backend/internal/web/templates/`），共享样式通过 `go:embed` 托管在 `/assets/landing/*`。

## 请求生命周期

### 短链接跳转（热路径）

```
1. 用户访问 go.example.com/abc
2. Nginx TLS 终止，转发到 GravityLink
3. host_router 中间件识别为入口域名，交给 redirect handler
4. Redis GET link:cache:abc
   - HIT  → 取出 {target_url, type, landing_page_id, strategy_id}
   - MISS → 查 MySQL links 表 → 回填 Redis（TTL 24h）→ 继续
5. 根据 type 处理：
   - short + 无落地页 → HTTP 302 到 target_url
   - short + 有落地页 → HTTP 302 到 page.example.com/abc
   - channel         → HTTP 302 到 target_url（附加 utm 参数）
   - liveqr          → 执行轮换策略 → HTTP 302 到 page.example.com/abc
6. 同步：HTTP 响应已发出
7. 异步 goroutine：
   - Redis INCR stat:pv:{link_id}:{date}
   - Redis INCR stat:uv:{link_id}:{date}:{ip_hash}（HyperLogLog）
   - Redis LPUSH access:stream {link_id, ip, ua, referer, timestamp}
```

### 统计异步落盘

```
log_consumer（goroutine，每 5 秒批处理）：
  Redis LRANGE access:stream 0 99 → 取最多 100 条
  → 并发解析 IP 归属地 + UA
  → 批量 INSERT access_logs
  → Redis LTRIM access:stream 100 -1

stat_flush（goroutine，每小时整点）：
  遍历 Redis stat:pv:* 键
  → 批量 GETDEL
  → UPSERT stat_daily / stat_hourly / stat_geo / stat_device
```

## 安全设计

| 层级 | 措施 |
|-----|------|
| 网络层 | Nginx 对管理域名限制内网 IP |
| 认证层 | Logto OIDC，JWT 验证签名 + 过期时间 |
| 授权层 | 路由中间件检查 JWT Claims 中的 role |
| 接口层 | 重定向接口限流（防刷统计）|
| 数据层 | 所有写操作通过 service 层，禁止直接 SQL 拼接 |

## 部署拓扑

### 首次初始化启动器

单进程启动时先创建不依赖 MySQL/Redis 的运行时调度器。配置完整时调度器立即装载正常 Gin 路由与后台 Worker；配置缺失或依赖连接失败时，管理端端口提供一次性初始化 API 和同一套 Vue 管理界面，公网端口只返回未初始化提示。初始化成功后通过原子 handler 切换启用业务服务，不要求重启容器。

初始化配置默认持久化到 `CONFIG_FILE` 指向的 JSON 文件，容器使用 `/data/gravitylink.json` 和独立 volume。环境变量优先于文件配置，适合由编排平台统一托管密钥的生产部署。

```
┌─────────────────────────────────────┐
│          docker-compose             │
│                                     │
│  gravitylink   (Go 二进制 + 前端)   │
│  mysql         (MySQL 8)            │
│  redis         (Redis 7)            │
│  nginx         (TLS 终止 + 反代)    │
│                                     │
│  Logto         (用户自行维护)        │
└─────────────────────────────────────┘
```
