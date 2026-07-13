# GravityLink 文档索引

## 核心设计文档

| 文档 | 说明 | 状态 |
|-----|------|------|
| [architecture.md](architecture.md) | 系统整体架构、模块划分、部署拓扑 | ✅ 完成 |
| [tech-decisions.md](tech-decisions.md) | 技术选型及决策理由 | ✅ 完成 |
| [data-model.md](data-model.md) | 数据库 Schema 设计（全量 DDL） | ✅ 完成 |
| [domain-routing.md](domain-routing.md) | 域名类型与请求路由分发设计 | ✅ 完成 |
| [api-overview.md](api-overview.md) | API 设计规范与路由总览 | ✅ 完成 |
| [phase-1-plan.md](phase-1-plan.md) | Phase 1 实施拆分、验收标准与风险 | ✅ 完成 |

## 功能模块规格

| 文档 | 说明 | 状态 |
|-----|------|------|
| [features/short-link.md](features/short-link.md) | 短链接功能规格 | ✅ 完成 |
| [features/channel-code.md](features/channel-code.md) | 渠道码功能规格 | ✅ 完成 |
| [features/live-qr.md](features/live-qr.md) | 群活码功能规格 | ✅ 完成 |
| [features/landing-page.md](features/landing-page.md) | 落地页模板系统规格 | ✅ 完成 |
| [features/statistics.md](features/statistics.md) | 统计系统规格 | ✅ 完成 |
| [features/domain-management.md](features/domain-management.md) | 域名管理规格 | ✅ 完成 |

## 开发阶段

- **Phase 1A**：后端最小骨架（Go 初始化、健康检查、配置、MySQL/Redis、docker-compose）
- **Phase 1B**：Schema 与域名路由（建表、GORM 模型、Host 路由、域名缓存）
- **Phase 1C**：短链接闭环（短码、CRUD、Redis 缓存、302 重定向）
- **Phase 1D**：认证与管理端脚手架（Logto、RBAC、Vue 3 管理端）
- **Phase 2**：核心功能（渠道码、群活码、落地页、域名路由）
- **Phase 3**：统计看板（多维聚合、ECharts 图表）
- **Phase 4**：开源收尾（迁移工具、docker-compose、README）

当前阶段：**Phase 1 已完成，Phase 2 待启动**，见 [workspace/status.md](../workspace/status.md)
