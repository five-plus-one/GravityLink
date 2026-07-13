# GravityLink — 项目规范

## 项目简介

GravityLink 是一个高性能短链接与活码管理系统，基于 Go + Vue 3 构建，替代旧 PHP 版本。支持短链接、渠道码、群活码、落地页，以及多维度访问统计。

## 目录约定

```
GravityLink/
├── AGENTS.md              # 本文件，项目规范
├── Docs/                  # 所有设计文档（进 git）
│   ├── README.md          # 文档索引
│   ├── architecture.md    # 系统架构
│   ├── data-model.md      # 数据库 Schema
│   ├── tech-decisions.md  # 技术选型决策
│   ├── domain-routing.md  # 域名路由设计
│   ├── api-overview.md    # API 设计规范
│   └── features/          # 各功能模块详细规格
├── workspace/             # 临时工作区（不进 git）
│   └── status.md          # 当前项目状态（必须及时更新）
├── backend/               # Go 后端（Phase 1 后建立）
├── frontend/              # Vue 3 前端（Phase 1 后建立）
└── deploy/                # 部署配置
```

## 工作流规范

1. **文档先行**：任何功能改动前，先更新 `Docs/` 对应文档，再动代码
2. **状态同步**：每次开始和结束工作，更新 `workspace/status.md`
3. **workspace 保洁**：临时文件用完即删，不留垃圾文件
4. **不允许边写边设计**：不清楚的需求先在 workspace 草稿，确认后才写入 Docs

## 开发纪律

- 改完主动运行验证（命令见各阶段说明），不只改不验
- 密钥、token 不进代码、不进 commit、不进日志（使用 `.env`）
- 大改动前进 Plan Mode 出方案，确认后再动手
- 红线操作（删文件、改 .env、数据库 schema 变更、git push）必须先问用户

## 命名约定

- Go 包名：小写单词，无下划线（`internal/router`，`internal/domain`）
- Go 文件名：小写下划线（`access_log.go`，`routing_strategy.go`）
- 数据库表名：小写下划线，无前缀（`links`，`routing_targets`，`stat_daily`）
- API 路由：小写中划线（`/api/v1/short-links`，`/api/admin/domains`）
- Vue 组件：PascalCase（`LinkTable.vue`，`StatChart.vue`）

## 分支策略

- `main`：稳定可发布版本
- `dev`：日常开发主分支
- `feat/<name>`：功能分支，合并到 dev

## 验证命令（Phase 1 建立后补充）

```bash
# 后端
cd backend && go build ./... && go test ./...

# 前端
cd frontend && npm run type-check && npm run build
```
