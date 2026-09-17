# 功能规格：域名管理

## 功能描述

管理员在后台配置三种类型的域名（入口/中转/落地），系统根据请求的 Host 头自动识别域名类型并路由到对应处理器。

## 域名类型职责

| 类型 | 枚举 | 说明 |
|-----|------|------|
| 入口域名 | `entry` | 对外分发，用户点击的短链接 |
| 中转域名 | `transit` | 流量中间层，隔离入口与落地（可选使用） |
| 落地域名 | `landing` | 系统渲染落地页的域名 |

同一系统可配置多个各类型域名，创建链接时选择使用哪个。

## 按域名配置首页

每个域名可单独设置访问根路径 `/` 时的展示逻辑；未覆盖的域名回退到系统设置里的全局首页。

| 模式 | 枚举 `home_mode` | 行为 |
|-----|------------------|------|
| 跟随全局 | `default`（默认） | 使用系统设置的首页标题/说明；若配置了全局「首页自动跳转地址」则跳转 |
| 自动跳转 | `redirect` | 跳转到该域名的 `home_redirect_url`；为空时回退全局跳转地址，再为空则显示默认首页 |
| 落地页 | `landing` | 渲染 `home_landing_page_id` 指定的落地页（仅支持 `custom` / `redirect_notice` 模板；`redirect_notice` 的目标 URL 取 `home_redirect_url`）。落地页无效时回退默认首页 |

字段：

- `home_mode`：`default` / `redirect` / `landing`
- `home_redirect_url`：http/https 绝对地址（自动去掉首尾空白与包裹引号）
- `home_landing_page_id`：落地页 ID

说明：活码/客服/卡密等模板依赖具体链接上下文，不能直接作为域名首页。

## 管理端操作

### 添加 / 编辑域名

字段：

- `host`：域名（不含协议，如 `go.example.com`）
- `type`：类型选择
- `scheme`：`http` 或 `https`（默认 https）
- `remark`：备注说明
- `home_mode` / `home_redirect_url` / `home_landing_page_id`：按域名首页（可选，默认跟随全局）

添加后需在 DNS 和 Nginx 中手动配置，系统不自动处理 DNS 和证书（开源版本由用户自行管理 TLS）。

### 删除域名

删除前校验：若有链接正在使用该域名（`entry_domain_id`、`transit_domain_id`、`landing_domain_id` 任一关联），拒绝删除，提示关联链接数量。

## 域名缓存

系统维护内存 Map `host → domain record`：

- 启动时从 MySQL 全量加载
- 管理端增删改域名时，API 层触发刷新（发信号给 host_router 重新加载）
- 后台每 5 分钟兜底刷新一次

未在数据库中登记的 Host 请求一律返回 404，不做任何处理。

## 与 Nginx 的协作

GravityLink 本身不处理 TLS，依赖 Nginx 做 TLS 终止。Nginx 需要：

1. 将所有配置的域名（入口/中转/落地）的 443 端口流量转发到 GravityLink 的 8080 端口
2. 正确设置 `proxy_set_header Host $host`，确保 GravityLink 能读到原始域名
3. 管理域名（API 域名）额外限制内网 IP

文档 `Docs/domain-routing.md` 提供 Nginx 配置示例。

## 多域名场景示例

```
入口域名：go.company.com（主域）、s.company.com（备用）
中转域名：t.company.com
落地域名：page.company.com
管理域名：admin.company.com（内网限制）
```

创建链接时选择使用哪个入口域名，同一个系统的不同业务线可以使用不同的入口域名。

首页示例：

- `r-l.ink` → `redirect` 到官网
- `s.example.com` → `default` 显示说明页
- `page.example.com` → `landing` 挂自定义门户页
