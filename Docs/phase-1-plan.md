# Phase 1 实施计划

## 目标

Phase 1 的目标是把 GravityLink 从文档阶段推进到“可运行、可验证”的最小系统：

- 后端 Go 服务可以启动
- MySQL / Redis 依赖可以通过 docker-compose 拉起
- 数据库 Schema 可以初始化
- 域名 Host 路由可以识别入口/中转/落地域名
- 短链接可以创建、查询、缓存并完成 302 跳转
- 管理端具备最小 Vue 3 脚手架和登录态接入基础

Phase 1 不追求完整产品体验，重点是打通热路径和工程骨架。

## 阶段拆分

### Phase 1A：后端最小骨架

交付内容：

- 初始化 `backend/` Go module
- 建立目录结构：`cmd/server`、`internal/config`、`internal/database`、`internal/middleware`、`internal/router`、`internal/service`、`internal/model`
- Gin 服务启动与 `/api/v1/health`
- 配置加载（环境变量优先，后续可扩展 `.env`）
- MySQL / Redis 连接初始化
- 基础日志与优雅退出
- `deploy/docker-compose.yml` 基础版：MySQL、Redis、后端

验收命令：

```bash
cd backend && go build ./... && go test ./...
```

验收标准：

- 服务能启动在 `:8080`
- `GET /api/v1/health` 返回统一响应格式
- MySQL / Redis 连接失败时有明确错误日志

### Phase 1B：Schema 与域名路由

交付内容：

- 将 `Docs/data-model.md` 中的 DDL 落为初始化 SQL 或 migration
- 建立 GORM 模型：`User`、`Domain`、`Link`
- 实现域名缓存：启动加载、管理 API 更新后刷新、后台兜底刷新
- 实现 Host 路由中间件
- 建立 entry / transit / landing 三类占位 handler

验收标准：

- 未登记 Host 返回 404
- entry Host 进入短链处理器
- transit Host 进入中转处理器
- landing Host 进入落地页处理器

### Phase 1C：短链接闭环

交付内容：

- Base62 包
- 短码校验与唯一性约束
- `links` CRUD API 的最小实现
- Redis `link:cache:{code}` 读写
- 短链接 302 重定向
- 禁用、过期、未找到的访问响应
- 最小访问统计写入 Redis List：`access:stream`

验收标准：

- 可通过 API 创建短链接
- 访问入口域名 `/{code}` 能 302 到目标 URL
- Redis 命中时不查 MySQL
- 过期链接返回 410
- 禁用或不存在链接返回 404

### Phase 1D：认证与管理端脚手架

交付内容：

- Logto JWT 验证中间件
- Admin / User 两级 RBAC
- Vue 3 + TypeScript + Vite 管理端脚手架
- 登录态基础封装
- 链接列表最小页面
- 前端构建产物预留 Go embed 路径

验收标准：

- 未登录访问受保护 API 返回 4401
- 普通用户访问 Admin API 返回 4403
- 管理端能完成登录态判断并请求链接列表

## 技术决策补充

### 统计队列

Phase 1 采用 Redis List 作为访问日志队列：

- 写入：`LPUSH access:stream {json_payload}`
- 消费：`LRANGE access:stream 0 99`
- 裁剪：`LTRIM access:stream 100 -1`

理由：

- 实现简单，足够支撑早期闭环
- 已与 `Docs/features/statistics.md` 和 `Docs/data-model.md` 的 Redis Key 设计一致
- 后续如需要消费者组、ACK、重试语义，可在 Phase 3 或更高流量阶段升级为 Redis Stream

### Git 状态

当前 `.git` 目录为空，不是有效 Git 仓库。Phase 1 正式开工前需要二选一：

- 恢复原有 `.git` 元数据
- 删除空 `.git` 后重新 `git init`

该动作会影响版本历史，应由用户确认后执行。

## 非目标

Phase 1 暂不实现：

- 渠道码完整管理
- 群活码完整轮换策略
- 落地页模板完整渲染
- 统计看板
- 日志归档
- 旧版 PHP 数据迁移
- Nginx 完整生产配置

## 风险与注意事项

- 短码设计要求“软删除不释放短码”，实现时不能只依赖普通唯一索引与软删除查询逻辑。
- Host 路由必须去除端口后再匹配域名。
- 统计写入必须异步，不能阻塞重定向响应。
- 自定义 HTML 落地页涉及 XSS 过滤，应放到落地页阶段再严肃实现。
- Logto Claims 的角色字段名需要与用户现有 Logto 配置核对。
