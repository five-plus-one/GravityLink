# 首次初始化

## 目标

GravityLink 的管理端必须能够在业务数据库或 Logto 尚未配置时启动，并提供一次性初始化向导。初始化页面只由管理端端口提供，公网短链接端口不暴露初始化写接口。

## 状态机

1. 程序启动后读取环境变量与 `CONFIG_FILE` 指向的持久化配置文件。
2. 配置完整且 MySQL、Redis 可连接时，直接启动正常业务路由。
3. 配置缺失或依赖连接失败时，管理端进入 `setup_required` 状态；公网端口返回服务尚未初始化的提示。
4. 管理端通过初始化 API 测试 MySQL、Redis，并校验 Logto 必填项。
5. 用户确认后，后端以原子方式写入配置文件，并在当前进程内启动业务服务。
6. 业务服务启动成功后，初始化写接口立即锁定；后续配置由已认证管理员维护。

## 配置优先级

运行配置按以下优先级合并：

1. 显式环境变量
2. `CONFIG_FILE` 持久化配置
3. 开发默认值

容器默认使用 `/data/gravitylink.json`，并通过独立 Docker volume 持久化。配置文件可能包含数据库和 Redis 密码，创建时使用仅当前运行用户可读写的权限，API 永不回传密码。

## 初始化表单

### 数据库

- MySQL Host、Port、Database、User、Password、连接参数
- Redis Host、Port、Password、DB
- 独立的连接测试操作

当前版本要求目标 MySQL 已创建 GravityLink 表结构；官方 compose 会通过 `deploy/mysql/init/001_schema.sql` 自动完成。

### Logto

- 是否关闭鉴权（仅非 production 环境允许）
- Issuer
- SPA App ID
- API Audience
- JWKS URL（可选）
- Scopes
- 管理端公开 URL
- 允许管理的角色

Logto 回调地址为 `{ADMIN_BASE_URL}/auth/callback`。

## 安全边界

- `/api/setup/*` 只在管理端监听器中处理。
- 初始化完成后，测试与保存接口返回 `setup_locked`。
- production 环境禁止保存 `AUTH_DISABLED=true`。
- API 状态响应只包含非敏感配置和缺失项，不返回数据库、Redis 密码或完整 DSN。
- 初始化接口应由反向代理限制在可信网络内；完成初始化后仍建议限制管理端域名的公网访问。
