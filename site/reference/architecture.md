# 架构摘要

给部署、排障和二次开发用的最小地图。完整设计见仓库 `Docs/architecture.md`。

## 进程与端口

```text
                    Internet
                       │
         ┌─────────────┴─────────────┐
         │                           │
   入口/中转/落地 Host           管理 Host
         │                           │
         └─────────────┬─────────────┘
                       │
                 Nginx TLS
                       │
              GravityLink 单进程
         ┌─────────────┴─────────────┐
         │                           │
   :8080 短链/落地              :8081 管理后台
   （公网）                     （建议内网）
         │                           │
         └─────────────┬─────────────┘
                       │
              ┌────────┴────────┐
              │                 │
            MySQL             Redis
         业务+统计          缓存/队列/计数
```

单二进制 + 嵌入管理端静态资源；不拆微服务。

## 请求如何被分发

Host 中间件读 `Request.Host`：

| Host 类型 | 进入 |
|-----------|------|
| entry | Redirect Handler：查短码、校验状态、302 |
| transit | Transit Handler：中转统计后再跳 |
| landing | Page Handler：渲染落地页 |
| 未登记 | 404 |

管理 API 在 ADMIN 监听器上，与公网 Host 路由隔离。

## 跳转与统计

1. 缓存查短码（Redis），未命中回源 MySQL  
2. 按类型执行：直达 / 落地 / 渠道拼 UTM / 活码轮换  
3. 异步：解析 UA/IP → Redis 计数 + 日志队列  
4. Worker 批量写 `access_logs`  
5. 定时把非今日计数落到 `stat_*` 聚合表  

详见[统计分析](/features/statistics)。

## 关键依赖挂了会怎样

| 依赖 | 影响 |
|------|------|
| MySQL | 无法持久化；缓存未命中时跳转失败 |
| Redis | 缓存/计数/队列不可用，性能与统计严重受损 |
| 数据卷 `/data` | 配置与上传丢失，需重新初始化 |
| IP 库文件 | 仅地域解析降级，其它正常 |

## 目录约定（仓库）

```text
backend/          Go：路由、服务、模型、Worker
frontend/admin/   Vue 3 管理端
deploy/           Compose 与部署说明
Docs/             工程规格（面向开发）
site/             本公开文档站
```

## 与旧版关系

Go + Vue 3 重写，提供旧 PHP 数据迁移 CLI（`Docs/migration.md`）。功能边界以当前管理端为准；内部验收报告在 `Docs/acceptance-*.md`。
