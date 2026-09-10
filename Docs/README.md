# GravityLink 文档索引

功能完成度请以 [2026-09-07 旧版对照审计](legacy-comparison-2026-09-07.md) 为准。下表“完成”表示规格文档编写状态，不代表每项业务已通过验收。

## 核心设计文档

| 文档 | 说明 | 状态 |
|-----|------|------|
| [architecture.md](architecture.md) | 系统整体架构、模块划分、部署拓扑 | ✅ 完成 |
| [tech-decisions.md](tech-decisions.md) | 技术选型及决策理由 | ✅ 完成 |
| [data-model.md](data-model.md) | 数据库 Schema 设计（全量 DDL） | ✅ 完成 |
| [domain-routing.md](domain-routing.md) | 域名类型与请求路由分发设计 | ✅ 完成 |
| [api-overview.md](api-overview.md) | API 设计规范与路由总览 | ✅ 完成 |
| [phase-1-plan.md](phase-1-plan.md) | Phase 1 实施拆分、验收标准与风险 | ✅ 完成 |
| [migration.md](migration.md) | 旧版数据迁移 CLI 使用说明 | ✅ 完成 |
| [setup.md](setup.md) | 初始化向导设计（数据服务/管理员身份/确认启用） | ✅ 完成 |
| [ui-design-system.md](ui-design-system.md) | 设计令牌、组件约定、文案规范 | ✅ 完成 |

## 功能模块规格

| 文档 | 说明 | 状态 |
|-----|------|------|
| [features/short-link.md](features/short-link.md) | 短链接功能规格 | ✅ 完成 |
| [features/share-card.md](features/share-card.md) | 微信分享卡片、公众号配置和素材 | 已实现，真实微信待接入验收 |
| [features/channel-code.md](features/channel-code.md) | 渠道码功能规格 | ✅ 完成 |
| [features/live-qr.md](features/live-qr.md) | 群活码功能规格 | ✅ 完成 |
| [features/landing-page.md](features/landing-page.md) | 落地页模板系统规格 | ✅ 完成 |
| [features/statistics.md](features/statistics.md) | 统计系统规格 | ✅ 完成 |
| [features/domain-management.md](features/domain-management.md) | 域名管理规格 | ✅ 完成 |

## 运维与部署

| 文档 | 说明 |
|-----|------|
| [../deploy/DEPLOY.md](../deploy/DEPLOY.md) | 服务器部署教程（外部 MySQL/Redis、Nginx 反代、备份） |

预构建镜像：`5plus1/gravitylink:latest`。本地全栈构建与外部库部署 compose 文件见 `deploy/`。

## 开发阶段

- **Phase 1A**：后端最小骨架（Go 初始化、健康检查、配置、MySQL/Redis、docker-compose）
- **Phase 1B**：Schema 与域名路由（建表、GORM 模型、Host 路由、域名缓存）
- **Phase 1C**：短链接闭环（短码、CRUD、Redis 缓存、302 重定向）
- **Phase 1D**：认证与管理端脚手架（Logto、RBAC、Vue 3 管理端）
- **Phase 2**：核心功能（渠道码、群活码、落地页、域名路由）
- **Phase 3**：统计看板（多维聚合、ECharts 图表）
- **Phase 4**：开源收尾（迁移工具、docker-compose、README）
- **Phase 5**：UI/UX 全量重构（Naive UI + Pinia、路由化、设计令牌、五模板落地页）
- **Phase 6**：统计闭环与工程加固（StatFlusher、UA 解析、路由懒加载、关键路径测试）
- **Phase 7**：管理端产品化（概览看板、个人中心、交互深化、运行态恢复）
- **Phase 8**：客服码 / 卡密分发 / 开放 API / 分享卡片 / 访客记录 / 旧版兼容与部署

当前阶段：**Phase 8 收尾 + 旧版迁移兼容 + 服务器部署**，见 [workspace/status.md](../workspace/status.md)。

## 补充资料

| 文档 | 说明 |
|-----|------|
| [acceptance-2026-09-07.md](acceptance-2026-09-07.md) | 2026-09-07 验收记录 |
| [legacy-comparison-2026-09-07.md](legacy-comparison-2026-09-07.md) | 旧版功能对照与差距清单 |
| [operations-completion.md](operations-completion.md) | 批量操作、群码维护、启动恢复等补齐说明 |
