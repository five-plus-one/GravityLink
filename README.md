# GravityLink

GravityLink 是一个基于 Go + Vue 3 的短链接与活码管理系统，支持短链接、渠道码、群活码、落地页、域名路由和访问统计。

English overview is included below.

## 功能

- 短链接：自定义短码或自动 Base62 短码，支持禁用和过期。
- 渠道码：跳转时自动追加 UTM 参数，且不覆盖目标 URL 已有参数。
- 群活码：支持轮询和加权策略，按扫码上限切换目标。
- 落地页：支持 liveqr、redirect_notice、custom 模板。
- 域名路由：入口、中转、落地三类域名按 Host 自动分发。
- 统计：Redis 实时计数，MySQL 落盘，管理端 ECharts 看板。
- 认证：Logto OIDC JWT 验证，Admin/User RBAC。

## 技术栈

- 后端：Go、Gin、GORM
- 前端：Vue 3、TypeScript、Vite、ECharts
- 存储：MySQL 8、Redis 7
- 部署：Docker Compose、Nginx

## 快速启动

统一使用容器化部署。基础版会启动 MySQL、Redis 和一个 GravityLink 应用容器。应用容器内部监听两个端口：

- `8080`：后端 API / 公网短链入口 / 落地页
- `8081`：管理端前端，且 `/api/*` 会转给同一个后端 handler

默认启动：

```bash
docker compose -f deploy/docker-compose.yml up --build
```

打开：

- 后端健康检查：`http://127.0.0.1:8080/api/v1/health`
- 管理端前端：`http://127.0.0.1:8081`

如果本机 `3306`、`6379`、`8080` 或 `8081` 已被占用，可以只换宿主机映射端口：

```powershell
$env:MYSQL_PORT="13306"
$env:REDIS_PORT="16379"
$env:APP_PORT="18080"
$env:ADMIN_PORT="18081"
docker compose -f deploy/docker-compose.yml up --build
```

## 数据库与缓存配置

MySQL / Redis 连接信息与 Logto 认证配置**不需要提前准备**：首次访问管理端端口（`http://127.0.0.1:8081`）会自动进入初始化向导，在向导中填写连接信息和认证方式即可。compose 已通过 `SETUP_DEFAULT_MYSQL_HOST=mysql`、`SETUP_DEFAULT_REDIS_HOST=redis` 预填了默认主机。

使用官方 compose 时 MySQL 表结构由 `deploy/mysql/init/001_schema.sql` 自动导入；连接外部 MySQL 时，请先手动导入该文件。

## Logto 登录配置

管理端支持两种认证方式，均在首次初始化向导中选择：

- **Logto OIDC**：Authorization Code + PKCE 登录。
- **本地账号**：创建仅保存 bcrypt 哈希的超级管理员。

Logto SPA 应用需要把回调地址加入允许列表：

```text
https://admin.example.com/auth/callback
https://admin.example.com/setup/auth/callback
http://127.0.0.1:8081/auth/callback
http://127.0.0.1:8081/setup/auth/callback
```

### 首次初始化

管理端会先检查安装状态。尚未完成管理员认领、数据库或 Redis 配置缺失、依赖连接失败时，访问管理端端口会自动进入首次初始化向导；无需先编辑容器内文件，也不会显示无效的登录按钮。

向导先选择 Logto 或本地账号。Logto 模式会检查 OIDC Discovery 并要求完成一次真实登录；本地模式会创建仅保存 bcrypt 哈希的超级管理员。身份验证完成后再测试 MySQL/Redis，并在空库中自动创建表结构。最终配置写入 `CONFIG_FILE`（容器默认 `/data/gravitylink.json`），官方 compose 已为 `/data` 挂载独立的 `gravitylink_data` volume。业务路由会在当前进程内启用，不需要重启容器。初始化完成后匿名 setup 写接口自动锁定。

### 访问短链接

管理端只是后台，不应对公网用户开放。公网用户访问的是后端承接的入口/落地域名：

- 生产环境：将 `go.example.com`、`page.example.com` 等域名解析到 Nginx，再在管理端把它们分别配置为 `entry` / `landing`。用户访问 `https://go.example.com/{code}`，后端会直接 `302` 到目标 URL，或跳到 `https://page.example.com/{code}` 渲染落地页。
- 本地验证 Host 路由：`curl -H "Host: go.demo.localhost" http://127.0.0.1:18080/demo`（需先在管理端注册该入口域名）。

落地页的公开样式和脚本由 Go 二进制内嵌并托管在 `/assets/landing/*`，不需要把管理端 Vue 应用暴露给用户。

## 开发验证

后端：

```bash
cd backend
go test ./...
go build ./...
```

前端：

```bash
cd frontend/admin
npm run build
```

部署配置：

```bash
docker compose -f deploy/docker-compose.yml config
```

## 生产部署

唯一部署方式即上面的 compose 文件，生产环境只需两步调整：

1. 复制并修改环境变量模板（至少改掉 MySQL 密码）：

```bash
cp deploy/.env.example deploy/.env
```

2. 以守护模式启动：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build -d
```

TLS 终止与域名路由由你自己的反向代理（Nginx / Caddy 等）负责：将入口/落地域名的 HTTPS 流量反代到 `APP_PORT`，管理域名的流量反代到 `ADMIN_PORT`。

## 旧版迁移

先将旧系统数据导出为标准 CSV，然后执行：

```bash
cd backend
go build ./cmd/migrate-legacy
./migrate-legacy -domains domains.csv -links links.csv -dry-run
./migrate-legacy -domains domains.csv -links links.csv
```

详见 [Docs/migration.md](Docs/migration.md)。

## 文档

- [文档索引](Docs/README.md)
- [系统架构](Docs/architecture.md)
- [数据库模型](Docs/data-model.md)
- [API 设计](Docs/api-overview.md)
- [域名路由](Docs/domain-routing.md)

## English

GravityLink is a Go + Vue 3 link management system for short links, campaign links, live QR routing, landing pages, domain-based routing, and visit analytics.

### Features

- Short links with custom codes or generated Base62 codes.
- Campaign links with UTM parameter injection.
- Live QR links with round-robin and weighted routing targets.
- Landing pages for QR display, redirect notices, and sanitized custom HTML.
- Entry, transit, and landing domain routing by HTTP Host.
- Redis real-time counters and MySQL analytics storage.
- Logto OIDC JWT authentication and RBAC.

### Development

```bash
cd backend
go test ./...
go build ./...

cd ../frontend/admin
npm install
npm run build
```

### Deployment

```bash
cp deploy/.env.example deploy/.env
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build -d
```

TLS termination and domain routing are handled by your own reverse proxy (Nginx, Caddy, etc.): forward entry/landing domain HTTPS traffic to `APP_PORT` and admin traffic to `ADMIN_PORT`.

### Migration

```bash
cd backend
go build ./cmd/migrate-legacy
./migrate-legacy -domains domains.csv -links links.csv -dry-run
./migrate-legacy -domains domains.csv -links links.csv
```
