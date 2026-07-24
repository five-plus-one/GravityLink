# 域名路由设计

## 域名类型与职责

| 类型 | 枚举值 | 职责 | 示例 |
|-----|--------|------|------|
| 入口域名 | `entry` | 对外分发的短链域名，用户点击的链接 | `go.example.com` |
| 中转域名 | `transit` | 跳转链路中间层，隔离入口与落地，防连坐封号 | `t.example.com` |
| 落地域名 | `landing` | 系统托管落地页的域名，渲染群活码页或跳转提示页 | `page.example.com` |

同一系统可配置多个各类型域名（如多个入口域名对应不同业务线）。

公网用户只访问入口/中转/落地域名，例如 `https://go.example.com/abc`。`frontend/admin` 是管理端，用于创建和维护链接、域名、落地页与统计，不作为公网用户访问入口。

短链接直跳场景不需要公网前端：入口域名命中后由后端直接返回 `302 Location`。只有群活码、跳转提示、自定义落地页等需要页面展示的场景，才由落地域名渲染 HTML，并加载后端嵌入的 `/assets/landing/*` 静态资源。

## 跳转链路

### 完整链路（启用中转）

```
用户点击 → go.example.com/abc
            ↓
        入口域名处理：读取链接配置
        transit_domain_id 不为空
            ↓
        302 → t.example.com/abc?_from=entry
            ↓
        中转域名处理：记录中转统计
        landing_page_id 不为空
            ↓
        302 → page.example.com/abc
            ↓
        落地域名处理：渲染落地页（群活码 / 跳转提示）
```

### 精简链路（无中转，有落地页）

```
用户点击 → go.example.com/abc
            ↓
        302 → page.example.com/abc
            ↓
        渲染落地页
```

### 最短链路（无中转，无落地页）

```
用户点击 → go.example.com/abc
            ↓
        302 → https://target.com/path
```

## Host 路由分发（核心中间件）

GravityLink 单进程通过读取 HTTP `Host` 请求头决定处理逻辑：

```go
// 伪代码，实际在 middleware/host_router.go 实现
func HostRouter(domains *DomainCache) gin.HandlerFunc {
    return func(c *gin.Context) {
        host := c.Request.Host  // 去掉端口
        domain, ok := domains.Get(host)
        if !ok {
            c.AbortWithStatus(404)
            return
        }
        switch domain.Type {
        case "entry":
            c.Set("handler_type", "redirect")
        case "transit":
            c.Set("handler_type", "transit")
        case "landing":
            c.Set("handler_type", "page")
        }
        c.Next()
    }
}
```

域名列表从 MySQL 加载后缓存在内存 Map 中，域名配置变更时失效并重新加载（无需重启进程）。

## 各处理器行为

### Redirect Handler（入口域名）

1. 从 URL Path 提取短码（`/abc` → `abc`）
2. 查 Redis `link:cache:{code}`，Miss 则查 MySQL 并回填
3. 校验链接状态（是否过期、是否禁用）
4. 根据 `type` 执行不同逻辑：
   - `short`：直跳 `target_url` 或跳落地页
   - `channel`：拼接 UTM 参数后跳转
   - `liveqr`：执行轮换策略，选出目标后跳落地页
5. 若配置了中转域名，先跳中转域名（携带原短码）
6. 异步记录访问统计

### Transit Handler（中转域名）

1. 从 URL Path 提取短码
2. 记录中转统计（`via_transit = true`）
3. 查出最终目标（落地页 URL 或直跳 URL）
4. 302 跳转

### Page Handler（落地域名）

1. 从 URL Path 提取短码
2. 查链接关联的 `landing_page_id`
3. 读取 `landing_pages` 的 `template` 和 `content`
4. 服务端渲染 HTML（Go template）或返回前端 SPA 渲染所需的 JSON 数据

## Nginx 配置示意

```nginx
# 管理域名：仅内网访问
server {
    listen 443 ssl;
    server_name admin.example.com;

    allow 10.0.0.0/8;
    allow 192.168.0.0/16;
    allow 172.16.0.0/12;
    deny all;

    location / {
        proxy_pass http://gravitylink:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}

# 入口/中转/落地域名：公网访问
server {
    listen 443 ssl;
    server_name go.example.com t.example.com page.example.com;

    location / {
        proxy_pass http://gravitylink:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## 域名缓存刷新策略

- 启动时从 MySQL 加载所有 `status = active` 的域名到内存 Map
- 管理端更新域名配置时，API 层主动触发缓存刷新
- 后台每 5 分钟全量刷新一次（兜底）
