# API 设计规范

## 基本约定

- 基础路径：`/api/v1`（用户 API）、`/api/admin`（管理 API）
- 数据格式：JSON，Content-Type: `application/json`
- 认证方式：`Authorization: Bearer <access_token>`（Logto 签发的 JWT）
- 时间格式：ISO 8601，`2026-07-02T15:04:05Z`
- 分页：`?page=1&page_size=20`，响应体包含 `total`

## 统一响应格式

```json
// 成功
{
  "code": 0,
  "message": "ok",
  "data": { ... }
}

// 失败
{
  "code": 4001,
  "message": "短码已被占用",
  "data": null
}
```

## 错误码规范

| 范围 | 含义 |
|-----|------|
| 0 | 成功 |
| 4001–4099 | 业务错误（链接相关） |
| 4101–4199 | 业务错误（域名相关） |
| 4201–4299 | 业务错误（统计相关） |
| 4401 | 未认证 |
| 4403 | 无权限 |
| 5000 | 服务器内部错误 |

## 路由总览

### 公开路由（无需认证）

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/{code}` | 短链接跳转（入口域名处理） |
| GET | `/page/{code}` | 落地页渲染（落地域名处理） |
| GET | `/t/{code}` | 中转跳转（中转域名处理） |
| GET | `/api/v1/health` | 健康检查 |

### 用户 API（需登录，`/api/v1/`）

#### 链接管理

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/api/v1/links` | 链接列表（支持按 type 过滤） |
| POST | `/api/v1/links` | 创建链接（short / channel / liveqr） |
| GET | `/api/v1/links/:id` | 链接详情 |
| PUT | `/api/v1/links/:id` | 更新链接 |
| DELETE | `/api/v1/links/:id` | 删除链接（软删除） |
| POST | `/api/v1/links/:id/disable` | 禁用链接 |
| POST | `/api/v1/links/:id/enable` | 启用链接 |

#### 群活码路由策略

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/api/v1/links/:id/strategy` | 获取轮换策略 |
| PUT | `/api/v1/links/:id/strategy` | 更新轮换策略（含目标列表） |
| POST | `/api/v1/links/:id/strategy/targets` | 新增目标 |
| PUT | `/api/v1/links/:id/strategy/targets/:tid` | 更新目标 |
| DELETE | `/api/v1/links/:id/strategy/targets/:tid` | 删除目标 |

#### 落地页

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/api/v1/landing-pages` | 落地页列表 |
| POST | `/api/v1/landing-pages` | 创建落地页 |
| GET | `/api/v1/landing-pages/:id` | 落地页详情 |
| PUT | `/api/v1/landing-pages/:id` | 更新落地页 |
| DELETE | `/api/v1/landing-pages/:id` | 删除落地页 |

#### 统计

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/api/v1/stats/:link_id/summary` | 汇总数据（总 PV/UV） |
| GET | `/api/v1/stats/:link_id/daily` | 按天趋势（`?start=&end=`） |
| GET | `/api/v1/stats/:link_id/hourly` | 按小时分布（指定某天） |
| GET | `/api/v1/stats/:link_id/geo` | 地域分布 |
| GET | `/api/v1/stats/:link_id/device` | 设备分布 |

### 管理 API（需 Admin 角色，`/api/admin/`）

#### 域名管理

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/api/admin/domains` | 域名列表 |
| POST | `/api/admin/domains` | 添加域名 |
| PUT | `/api/admin/domains/:id` | 更新域名 |
| DELETE | `/api/admin/domains/:id` | 删除域名 |

#### 用户管理

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/api/admin/users` | 用户列表 |
| PUT | `/api/admin/users/:id/role` | 修改用户角色 |
| POST | `/api/admin/users/:id/disable` | 禁用用户 |

#### 系统配置

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/api/admin/configs` | 获取系统配置 |
| PUT | `/api/admin/configs` | 更新系统配置 |

#### 全局统计（管理端看板）

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | `/api/admin/stats/overview` | 系统总览（总链接数、今日 PV 等） |
| GET | `/api/admin/stats/top-links` | 访问量 TOP 链接 |

## 关键请求体示例

### 创建短链接

```json
POST /api/v1/links
{
  "type": "short",
  "code": "mylink",           // 可选，不填则自动生成
  "entry_domain_id": 1,
  "transit_domain_id": null,  // 不经中转
  "landing_domain_id": null,  // 直跳
  "target_url": "https://example.com/page",
  "title": "活动落地页",
  "expire_at": null
}
```

### 创建渠道码

```json
POST /api/v1/links
{
  "type": "channel",
  "code": "wechat-a",
  "entry_domain_id": 1,
  "target_url": "https://example.com/page",
  "title": "微信渠道 A",
  "channel": {
    "utm_source": "wechat",
    "utm_medium": "social",
    "utm_campaign": "spring2026",
    "utm_content": "poster_a"
  }
}
```

### 创建落地页

```json
POST /api/v1/landing-pages
{
  "template": "liveqr",
  "title": "扫码加入交流群",
  "domain_id": 3,
  "content": {
    "headline": "扫码加入交流群",
    "subtext": "群满自动切换，永久有效",
    "footer_text": "长按识别二维码",
    "theme_color": "#1677ff"
  }
}
```

### 创建群活码

```json
POST /api/v1/links
{
  "type": "liveqr",
  "entry_domain_id": 1,
  "transit_domain_id": 2,
  "landing_domain_id": 3,
  "landing_page_id": 5,
  "title": "春季活动群",
  "strategy": {
    "mode": "round_robin",
    "targets": [
      {
        "label": "群1",
        "target_url": "https://weixin.qq.com/g/xxx1",
        "scan_limit": 200,
        "weight": 1
      },
      {
        "label": "群2",
        "target_url": "https://weixin.qq.com/g/xxx2",
        "scan_limit": 200,
        "weight": 1
      }
    ]
  }
}
```
