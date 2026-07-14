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

开发环境默认开启 `AUTH_DISABLED=true`，方便本地调试。

```bash
docker compose -f deploy/docker-compose.yml up --build
```

服务默认监听：

- API: `http://127.0.0.1:8080`
- MySQL: `127.0.0.1:3306`
- Redis: `127.0.0.1:6379`

前端管理端：

```bash
cd frontend/admin
npm install
npm run dev
```

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
docker compose -f deploy/docker-compose.full.yml --env-file deploy/.env config
```

## 生产部署

复制环境变量模板：

```bash
cp deploy/.env.example deploy/.env
```

修改 `deploy/.env` 中的域名、数据库密码和 Logto 配置，并把 TLS 证书放到：

```text
deploy/nginx/certs/fullchain.pem
deploy/nginx/certs/privkey.pem
```

启动完整栈：

```bash
docker compose -f deploy/docker-compose.full.yml --env-file deploy/.env up --build -d
```

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
docker compose -f deploy/docker-compose.full.yml --env-file deploy/.env up --build -d
```

Put TLS certificates at:

```text
deploy/nginx/certs/fullchain.pem
deploy/nginx/certs/privkey.pem
```

### Migration

```bash
cd backend
go build ./cmd/migrate-legacy
./migrate-legacy -domains domains.csv -links links.csv -dry-run
./migrate-legacy -domains domains.csv -links links.csv
```
