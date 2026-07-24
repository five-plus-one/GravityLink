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

基础版会启动 MySQL、Redis 和一个 GravityLink 应用容器。应用容器内部监听两个端口：

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
$env:MYSQL_HOST_PORT="13306"
$env:REDIS_HOST_PORT="16379"
$env:APP_HOST_PORT="18080"
$env:ADMIN_HOST_PORT="18081"
docker compose -f deploy/docker-compose.yml up --build
```

本地开发也可以单独跑 Vite 管理端：

```bash
cd frontend/admin
npm install
npm run dev
```

如果后端 API 不是 `8080`，启动前端时指定代理目标：

```powershell
$env:VITE_API_PROXY_TARGET="http://127.0.0.1:18080"
npm run dev -- --host 127.0.0.1 --port 5173
```

打开：

- 管理端：`http://127.0.0.1:5173`
- 健康检查：`http://127.0.0.1:18080/api/v1/health`

## 数据库与缓存配置

推荐使用拆分环境变量，应用会自动拼接 MySQL DSN：

```env
MYSQL_HOST=mysql
MYSQL_PORT=3306
MYSQL_DATABASE=gravitylink
MYSQL_USER=gravitylink
MYSQL_PASSWORD=change-me
MYSQL_PARAMS=charset=utf8mb4&parseTime=True&loc=Local
```

如果你使用云数据库、特殊参数或密码里包含特殊字符，可以直接提供完整 DSN 覆盖：

```env
MYSQL_DSN=gravitylink:change-me@tcp(mysql:3306)/gravitylink?charset=utf8mb4&parseTime=True&loc=Local
```

Redis 推荐拆分配置：

```env
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

也可以用 `REDIS_ADDR=redis:6379` 直接覆盖地址。`MYSQL_HOST_PORT` / `REDIS_HOST_PORT` 只控制 Docker 映射到宿主机的端口，不影响容器内应用连接数据库。

### 访问短链接

管理端只是后台，不应对公网用户开放。公网用户访问的是后端承接的入口/落地域名：

- 生产环境：将 `go.example.com`、`page.example.com` 等域名解析到 Nginx，再在管理端把它们分别配置为 `entry` / `landing`。用户访问 `https://go.example.com/{code}`，后端会直接 `302` 到目标 URL，或跳到 `https://page.example.com/{code}` 渲染落地页。
- 本地开发：可以用 `curl -H "Host: go.demo.localhost" http://127.0.0.1:18080/demo` 验证 Host 路由；开发模式也允许直接访问 `http://127.0.0.1:18080/{code}` 作为入口域名兜底。

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

如果不需要 Nginx，只想用一个应用容器映射两个端口，使用基础 compose 并在 `deploy/.env` 中配置端口：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build -d
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
