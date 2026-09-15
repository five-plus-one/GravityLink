# 域名管理

GravityLink 用**一个 Go 进程**读 HTTP `Host`，决定这条请求是跳转、中转还是渲染落地页。域名必须先在后台登记。

## 三种域名

| 类型 | 枚举 | 职责 | 示例 |
|------|------|------|------|
| 入口 | `entry` | 用户点击的短链 | `go.example.com` |
| 中转 | `transit` | 跳转中间层，隔离入口与落地 | `t.example.com` |
| 落地 | `landing` | 渲染落地页 | `page.example.com` |

管理后台单独用另一个 Host（如 `admin.example.com`），不要和入口混用。

同一类型可以配多个：不同业务线用不同入口域名很常见。

```text
入口：go.company.com / s.company.com
中转：t.company.com
落地：page.company.com
后台：admin.company.com（内网）
```

## 在后台添加域名

字段：

| 字段 | 说明 |
|------|------|
| host | 不含协议，如 `go.example.com`；可带显式端口 |
| type | entry / transit / landing |
| scheme | http 或 https（影响分享出去的地址前缀） |
| remark | 备注 |

添加后系统**不会**帮你改 DNS 或证书，需要你手动：

1. DNS A/CNAME 指到服务器  
2. Nginx 反代到短链端口 `18080`，并正确传 `Host`  
3. 等后台里域名可访问后再创建链接  

![域名管理](https://img.assets.five-plus-one.com/img/2026/09/6a97bf99019e1a8f7a9aa5144a3e533f.png)

添加域名弹窗：

![添加域名](https://img.assets.five-plus-one.com/img/2026/09/ecc3f4ddd5c8a18115dd17b4f4feabce.png)

## 跳转链路（和域名的关系）

### 最短（无中转、无落地）

```text
用户 → go.example.com/abc → 302 → https://target.com/...
```

### 有落地页

```text
用户 → go.example.com/abc → 302 → page.example.com/abc → 渲染 HTML
```

### 完整（中转 + 落地）

```text
用户 → go.example.com/abc
    → 302 t.example.com/abc
    → 302 page.example.com/abc → 渲染 HTML
```

中转用于：多记一层统计、入口与落地隔离、降低连坐风险。

## 缓存与生效

- 启动时全量加载域名到内存  
- 后台增删改会触发刷新  
- 另有周期性兜底刷新  

未登记的 Host 一律 404。若改了域名但旧缓存还在，可后台再保存一次或重启容器。

## 删除保护

若仍有链接通过该域名的入口/中转/落地 ID 引用，删除会被拒绝，并提示关联数量。先改链接或删链接。

## 与 Nginx 的约定

GravityLink **不**处理 TLS。Nginx 必须：

1. 所有入口/中转/落地域名的 443 → `127.0.0.1:18080`  
2. `proxy_set_header Host $host;`（丢了 Host 就全 404）  
3. 管理域名额外 `allow/deny`  

示例见[域名与 HTTPS](./nginx)。

## 为什么短链 404

按顺序查：

1. Host 是否已在后台添加，类型是否 entry，状态 active  
2. DNS 是否解析到这台机器  
3. Nginx `server_name` 与 `proxy_pass` 是否正确，是否传了 Host  
4. 是否反代错到了 18081（那是后台）  
5. 短码本身是否存在、未禁用、未过期  

## 复制地址的规则

- 链接绑定的入口域决定复制出来的完整 URL  
- 不会偷偷用列表里第一个域名  
- 入口域不可用时，复制/打开按钮会禁用并提示  

工程规格：仓库 `Docs/domain-management.md`、`Docs/domain-routing.md`。
