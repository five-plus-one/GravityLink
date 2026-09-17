# 功能规格：通知渠道 · 模板 · 活码健康巡检

> 状态：P1 已实施（2026-09-16）。验收以本地容器手工验收为准。

## 1. 目标

运营方在群活码「即将失效 / 已失效 / 快满员 / 已满员 / 整码无可用」时能及时收到通知，而不是等访客撞到「暂无可用群」才发现。

本规格覆盖三块，彼此衔接：

1. **通知渠道扩展**：在现有企业微信机器人、自定义 HTTP 之外，增加 **SMTP 邮件**（默认预填 QQ 邮箱）。
2. **通知模板与事件目录**：事件化、可配置文案，支持占位变量。
3. **活码过期识别与主动引导**：上传群二维码后自动识别/估算过期时间并回填；若未开启任何通知渠道，主动引导配置。

## 2. 现状（2026-09-16）

| 能力 | 现状 | 缺口 |
|------|------|------|
| 渠道 | `notify_webhook_url`、`notify_http_url` 两个 KV，扁平存在 `system_configs` | 无邮件、无渠道启停/列表、无测试发送 |
| 发送器 | `service.Notifier`：读配置 → Redis SetNX 防抖 1h → 同步 POST | 防抖粒度粗、无模板、无发送记录 |
| 事件 | 访客触发 `qr_exhausted`；主动巡检活码健康 | 无「即将到期 / 已到期 / 接近阈值」主动提醒 |
| 目标到期 | `routing_targets.expire_at` 已有，手工填写；到期目标不参与分发 | 上传不自动填、无即将到期提醒 |
| 二维码解码 | 无 | 无法从图片识别微信码类型/过期 |

> 2026-09-17：域名封禁检测服务已下线，相关 Worker / 事件 / 设置项已移除。

相关代码：

- 发送器：`backend/internal/service/notify_channels.go`、`notify_smtp.go`、`notify_templates.go`
- 耗尽告警：`backend/internal/router/public.go`（访客命中 `ErrNoRoutingTarget` 时）
- 活码健康巡检：`backend/internal/worker/liveqr_health.go`
- 目标模型：`backend/internal/model/model.go` → `RoutingTarget`
- 素材上传：`backend/internal/router/content.go` → `POST /api/admin/materials`
- 目标批量添加：`backend/internal/router/targets.go` → `POST /links/:id/targets/batch`

## 3. 通知渠道

### 3.1 渠道模型

从「两个扁平 URL」升级为「渠道列表」。每个渠道：

| 字段 | 说明 |
|------|------|
| `id` | 本地标识（如 `smtp` / `wecom` / `http`，首版每类型至多一条） |
| `type` | `wecom` \| `http` \| `smtp`（预留 `dingtalk` / `feishu` / `telegram`） |
| `enabled` | 是否参与发送 |
| `settings` | 各类型私有配置（JSON） |

系统配置键：

- `notify_channels`：渠道数组 JSON（敏感字段见 3.4）
- `notify_events`：事件订阅与参数 JSON（见 §4.4）
- 旧键 `notify_webhook_url` / `notify_http_url` **只读兼容**：若新键不存在而旧键有值，首次加载时自动迁移为 `wecom` / `http` 渠道（各 enabled=true），写入新键后不再读旧键。

### 3.2 SMTP（新增，QQ 邮箱默认）

选 SMTP 是因为：零外部依赖、QQ/163/Gmail 通用、适合「换群码」这类需要留档的运营提醒。

**QQ 邮箱默认预填**（表单 placeholder + 首次「启用邮件」时的默认值，可改）：

| 项 | 默认值 | 说明 |
|----|--------|------|
| 服务器 | `smtp.qq.com` | |
| 端口 | `465` | SSL/TLS；同时可选 `587`（STARTTLS）、`25`（不推荐） |
| 加密 | `ssl`（465）/ `starttls`（587） | 与端口联动 |
| 发件人 | 留空，要求等于登录账号 | 填完整 QQ 邮箱 |
| 登录账号 | 同发件人 | |
| 密码 | **SMTP 授权码**，不是 QQ 密码 | 设置 → 账户 → 开启 SMTP → 生成授权码 |

实现约定：

- 依赖：标准库 `net/smtp` + 手工 TLS（或 `gomail`/`go-mail` 等轻量库，选型时以零 CGO、维护活跃为准）。
- 连接超时 10s，发送超时 15s。
- 邮件头：`Subject` 使用 MIME 编码中文；`From`/`To` 规范化。
- 正文首版纯文本（`text/plain; charset=utf-8`）；HTML 邮件列为 P2。
- 多收件人：逗号/分号分隔，上限 20。
- **禁止**日志打印授权码、完整连接串。

### 3.3 既有渠道

| type | 行为 |
|------|------|
| `wecom` | 现状：`POST` webhook，`msgtype=text`，`content = title + "\n" + content` |
| `http` | 现状：`POST` JSON `{title, content, time}`；继续兼容 Bark / Server酱 自建转发 |

两者均增加 `enabled`；关闭后跳过。

### 3.4 密钥存储

SMTP 授权码不得明文落库。复用现有 `sealSecret` / `openSecret`（`storage.s3.encrypted` 同款，密钥材料为数据目录 `wechat.key`）：

- `notify_channels` 中 `settings.password` / `settings.auth_code` 存密文。
- 查询接口只返回 `secret_configured: true/false`，不回显明文。
- 解密失败时返回明确中文提示（同 `ErrStorageUndecryptable`），要求重新填写授权码保存。
- 未配置任何加密密钥的全新安装：首次保存 SMTP 时生成/复用 `wechat.key`。

### 3.5 渠道 API（管理端）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/notify/channels` | 列表（脱敏） |
| PUT | `/api/admin/notify/channels` | 全量/增量保存 |
| POST | `/api/admin/notify/channels/{type}/test` | 向该渠道发一封测试通知，返回成败与错误摘要 |

测试通知标题固定：`GravityLink 通知测试`；内容含时间戳与渠道名。测试发送**绕过**事件防抖。

## 4. 事件与模板

### 4.1 设计原则

- **事件**是业务事实（到期、满额、耗尽…）；**模板**只负责把事实渲染成文案；**渠道**只负责投递。
- 一次事件可发往多个已启用且订阅了该事件的渠道；单渠道失败不影响其他渠道（记日志 + 可选发送记录）。
- 文案默认中文，用户语言与现有产品一致。

### 4.2 事件目录（首版）

| event | 触发时机 | 防抖建议 | 默认开启 |
|-------|----------|----------|----------|
| `target_expiring_soon` | 目标启用、未满额，且 `expire_at` 落入提前量窗口 | 每目标 24h 一次 | 是 |
| `target_expired` | 巡检发现：目标曾可分发（启用且未满额）且 `expire_at` 已过 | 每目标一次（自然日级） | 是 |
| `target_scan_near` | 扫码数 ≥ 阈值 × 比例（默认 90%） | 每目标 6h 一次 | 是 |
| `target_scan_exhausted` | 扫码数达到 `scan_limit`（分发层已跳过该目标） | 每目标一次 | 是 |
| `liveqr_no_available` | 某活码当前无任何可分发目标 | 每活码 1h（沿用现有） | 是 |

说明：

- `liveqr_no_available` **增加主动巡检触发**（不只依赖访客撞到耗尽页）。
- 「启用且未达阈值但已过期」对应 `target_expired`：正文明确写出当前扫码数/阈值，提示「未满员但码已过期，请换码」。
- 不在首版做：日报摘要、成员级订阅、微信服务号模板消息、域名封禁检测（服务已下线）。

### 4.3 模板变量

统一占位符（`{{.Name}}`）：

| 变量 | 含义 |
|------|------|
| `LinkCode` | 活码短码 |
| `LinkTitle` | 活码标题（无则空） |
| `LinkURL` | 管理端链接页地址（方便直达） |
| `TargetLabel` | 目标名称 |
| `TargetID` | 目标 ID |
| `ExpireAt` | 到期时间（本地时区 `2006-01-02 15:04`） |
| `ExpireIn` | 剩余可读时长（如 `23小时`、`2天`） |
| `ScanCount` / `ScanLimit` | 当前扫码 / 阈值（无限制显示 `—`） |
| `AvailableCount` / `TotalCount` | 当前可分发 / 总目标数 |
| `Reason` | 不可用原因汇总（满额/到期/停用） |
| `Time` | 事件发生时间 |

模板字段：`title`、`body`。渲染失败时回退到内置默认模板，不阻塞发送。

### 4.4 默认模板（内置，可改）

```text
# target_expiring_soon
title: 群活码即将过期
body: |
  活码「{{.LinkTitle}}{{if not .LinkTitle}}{{.LinkCode}}{{end}}」
  二维码「{{.TargetLabel}}」将于 {{.ExpireAt}} 过期（约 {{.ExpireIn}}）。
  当前扫码 {{.ScanCount}}/{{.ScanLimit}}，请及时更换群二维码。

# target_expired
title: 群活码二维码已过期
body: |
  活码「{{.LinkCode}}」二维码「{{.TargetLabel}}」已于 {{.ExpireAt}} 过期。
  扫码 {{.ScanCount}}/{{.ScanLimit}}（未达阈值但码已失效），请更换后启用。

# target_scan_near
title: 群活码扫码接近阈值
body: |
  活码「{{.LinkCode}}」二维码「{{.TargetLabel}}」
  扫码 {{.ScanCount}}/{{.ScanLimit}}，接近上限，请准备下一码。

# target_scan_exhausted
title: 群活码二维码已达阈值
body: |
  活码「{{.LinkCode}}」二维码「{{.TargetLabel}}」已达阈值 {{.ScanLimit}}，
  将不再参与分发。

# liveqr_no_available
title: 活码暂无可用二维码
body: |
  活码「{{.LinkCode}}」当前无可用二维码（可分发 {{.AvailableCount}}/{{.TotalCount}}）。
  原因：{{.Reason}}。访客会看到「暂无可用群」，请尽快补充。
```

`notify_events` 可保存：每事件的 `enabled`、`title`、`body` 覆盖，以及全局/事件级参数（见 4.5）。

### 4.5 可调参数

| 配置键（JSON 字段） | 默认 | 说明 |
|---------------------|------|------|
| `expiring_lead_hours` | `24` | 提前多少小时开始报「即将过期」 |
| `scan_near_ratio` | `0.9` | 接近阈值比例（0–1） |
| `liveqr_health_interval_minutes` | `10` | 活码健康巡检间隔 |

## 5. 活码健康巡检 Worker

新增 `backend/internal/worker/liveqr_health.go`，进程内与 StatFlusher 等并列启动。

### 5.1 周期与范围

- 默认每 10 分钟一轮（可配）。
- 扫描 `links.type='liveqr' AND status='active'` 及其策略与目标。
- 目标条件关注：`status='active'`（`disabled` / `exhausted` 仅统计，不单发过期事件）。

### 5.2 每轮逻辑

```text
for each active liveqr link:
  targets = active targets of strategy
  available = filter(target)  # 与 routing.filterAvailable 同口径：
                              # 未到期 且 (无阈值 或 scan_count < scan_limit)

  # A. 单目标：即将过期 / 已过期 / 接近阈值 / 已达阈值
  for t in targets:
    if t.ExpireAt 有效:
      if now >= ExpireAt:           → target_expired（若此前未报过）
      else if ExpireAt-now <= lead: → target_expiring_soon
    if t.ScanLimit 有效:
      if ScanCount >= ScanLimit:    → target_scan_exhausted
      else if ratio 达标:           → target_scan_near

  # B. 整码无可用
  if len(targets) > 0 and len(available) == 0:
    → liveqr_no_available（附 AvailableCount/TotalCount/Reason）
```

### 5.3 状态与去重

- 防抖继续走 Redis `notify:debounce:{eventKey}`，**TTL 按事件类型**（见 4.2），不再全局固定 1 小时。
- `eventKey` 约定：
  - `expiring:{targetID}:{yyyyMMdd}`（同一天只提醒一次）
  - `expired:{targetID}`
  - `scan_near:{targetID}`
  - `scan_exhausted:{targetID}`
  - `qr_exhausted:{linkCode}`（沿用，主动巡检与访客触发共用，防抖 1h）
- Redis 不可用时降级：跳过本轮通知（避免惊群），只打日志；**不**在无防抖下盲发。
- 不单独维护「已通知」表（首版）；以 Redis 防抖为准。发送失败日志可查。

### 5.4 与路由层的关系

- 访客触发的 `liveqr_no_available` **保留**（立即感知）；巡检补上「无人访问也会报」。
- 路由/编辑代码 **不**在请求路径上做扫描逻辑；仍保持异步 `SendAsync`。

## 6. 二维码过期时间识别

### 6.1 微信事实（产品约束）

| 码类型 | 过期规则 | 能否从图片精确解出过期时刻 |
|--------|----------|----------------------------|
| 微信群二维码 | 官方约 **7 天** | 多数情况下 **不能** 稳定读出生成时刻；以「识别为群码 → 按默认策略估算」为主 |
| 微信个人名片/渠道码 | 规则不同，可能长期有效或含参数 | 视载荷而定，失败则不填 |
| 非微信二维码（网页等） | 载荷本身 | 有 URL 参数 `expire/exp/expiry` 等则解析，否则不填 |

因此策略是：**能解析则解析，识别为微信群码则按默认天数估算，否则保持空并可手工填**。不在文档或界面上承诺「100% 读出微信服务器侧精确过期」。

### 6.2 识别流水线

在 **素材上传** 与 **目标添加（本地图片）** 两处复用同一服务 `service.QRInspect`：

```text
图片字节 → 解码 QR（gozxing / tuotoo 等纯 Go 库）
  → 得到 text 或 binary
  → classify:
       A. URL 且含 expire/exp/expiry（秒或毫秒时间戳，或 RFC3339）
            → 提取 ExpireAt，来源 = url_param
       B. 微信群入口特征（weixin.qq.com/g/、含群邀请特征的二进制/短链）
            → ExpireAt = 观察时刻 + default_days（默认 7），来源 = wechat_group_default
       C. 可识别的微信载荷内嵌秒级时间戳（启发式，可选实现）
            → ExpireAt = 解析值，来源 = wechat_payload（标记为「自动识别，可改」）
       D. 其他
            → 不填，来源 = none
```

启发式 C 允许失败；失败回落 B/D，不得阻断上传。

### 6.3 数据落点

**素材表增量字段**（`materials`，AutoMigrate/AddColumn）：

| 列 | 类型 | 说明 |
|----|------|------|
| `qr_content` | TEXT NULL | 解码得到的原始载荷（截断至 2KB） |
| `qr_kind` | VARCHAR(32) | `wechat_group` / `wechat_other` / `url` / `unknown` / `not_qr` |
| `suggested_expire_at` | DATETIME(3) NULL | 建议到期时间 |
| `expire_source` | VARCHAR(32) | `url_param` / `wechat_group_default` / `wechat_payload` / `none` |
| `inspected_at` | DATETIME(3) NULL | 上次识别时间 |

**路由目标**：不强制新列；添加目标时若 `expire_at` 为空且素材有 `suggested_expire_at`，则写入目标 `expire_at`。

已有目标不被巡检回写覆盖手工时间。

### 6.4 上传与添加链路

1. `POST /api/admin/materials`：写文件成功后同步调用 `QRInspect`（≤5MB，超时 3s），结果写入素材行。失败只记 `not_qr`，不失败上传。
2. `POST /links/:id/targets`（单条）与 `/targets/batch`：
   - URL 形如 `/uploads/...` 或同源素材 → 查素材识别结果；
   - 若目标未带 `expire_at` 且素材有建议值 → 自动填充；
   - 响应中带回 `expire_at`、`expire_source`，便于前端展示「已自动识别」。
3. 仅绝对外链、且素材库无记录的图片：首版 **不** 远程拉取解码（避免 SSRF 与超时）；前端提示可手工设置到期时间。P2 可加白名单域名拉取。

### 6.5 默认天数配置

`notify_events` 或独立键 `qr_default_expire_days`，默认 `7`。管理端「通知与检测」或活码目标区可改；用于 `wechat_group_default`。

## 7. 上传后引导开启通知

目标：用户刚上传/配置了带过期的群码时，若通知渠道全部未启用，能被看见地引导一次，而不是藏在设置里。

### 7.1 触发条件

同时满足：

1. 当前操作为：活码目标保存成功 / 批量添加成功 / 素材上传成功且 `suggested_expire_at` 非空；
2. 系统内 **没有任何 enabled 的通知渠道**；
3. 会话内该引导未被「不再提示」关闭（存 `localStorage` 键 `gravitylink.notify_guide_dismissed`）。

### 7.2 交互（管理端）

- `TargetManager` 保存/批量添加后：`NAlert type="info"` + 主按钮「去配置通知」→ 跳转 `设置 → 通知与检测`。
- 文案示例：「已为 3 张二维码识别到期时间。开启邮件/企业微信后，到期前会自动提醒你换码。」
- 设置页通知 Tab：若所有渠道关闭，顶部空状态卡片说明收益 + 一键填入 QQ SMTP 默认项 +「发送测试邮件」。
- 不阻断用户继续添加二维码；可关闭。

### 7.3 素材库

素材选择器对已识别的图显示角标（如「约 7 天后过期」）；未识别不显示。批量加入目标时在确认列表展示建议到期日。

## 8. 管理端信息架构

设置 →「通知与检测」重构为：

1. **渠道**：卡片列表（邮件 / 企业微信 / 自定义 HTTP），每张：启用开关、关键字段、密钥状态、「发送测试」。
2. **事件订阅**：表格勾选；展开可编辑该事件 title/body 模板与提前量/比例参数。
3. **域名封禁检测**：保留现有开关。

活码二维码配置（`TargetManager`）：

- 列表「到期」列增加来源提示（自动/手工）。
- 编辑表单在到期时间下说明自动规则。
- 批量添加结果若含自动到期，汇总提示。

## 9. 兼容与迁移

| 项 | 策略 |
|----|------|
| 旧 webhook/http URL | 首次读取时迁移进 `notify_channels`，行为不变 |
| 无渠道时 | 不发送，不报错（与现状一致） |
| 未设置到期的目标 | 不产生 expiring/expired 事件 |
| schema | `materials` 增量列；`system_configs` 不改表结构；无强制数据迁移脚本 |
| 密钥文件 | 复用 `data/wechat.key`；备份说明与 object-storage 文档一致 |

## 10. 非目标（明确不做）

- 钉钉/飞书/Telegram/Bark 原生 SDK（可继续走自定义 HTTP）
- 按管理员个人订阅、@ 指定人
- 通知历史检索 UI（仅日志；表结构可 P2 再加 `notification_logs`）
- 微信服务号模板消息
- 保证解析出微信服务器权威过期时间

## 11. 分期建议

| 阶段 | 内容 |
|------|------|
| **P1** | SMTP 渠道（QQ 预填）+ 渠道模型迁移 + 测试发送 + 事件/默认模板 + `LiveQRHealthWorker` + 上传识别回填到期 + 引导横幅 |
| **P2** | 发送记录表与失败重试；HTML 邮件；外链白名单解码；钉钉/飞书渠道；素材角标增强 |

建议一次 PR 做完 P1，避免「只有邮件渠道、没有事件」的半成品。

## 12. 验收清单（P1）

1. 配置 QQ SMTP（授权码）后「发送测试」成功；错误授权码返回中文失败原因且不泄漏密钥。
2. 旧 webhook 配置在升级后仍能发送；可单独关闭邮件渠道。
3. 上传微信群码图片 → 素材出现建议到期 ≈ +7 天；加入目标后 `expire_at` 自动填充且可改。
4. 将某目标 `expire_at` 调到 1 小时后 → 巡检窗口内收到「即将过期」；改为已过去时间 → 收到「已过期」且文案含扫码数/阈值。
5. 批量设阈值为当前扫码+1，访问满 → 「已达阈值」；全部目标满额/到期后 → 「暂无可用二维码」（巡检与访客触发不重复轰炸，防抖生效）。
6. 未配置任何渠道时，上传/添加带到期目标出现引导横幅；关闭后会话内不再出现。
7. `go build ./...`、`go test ./...`、前端 `type-check` / 单测 / `build` 通过。

## 13. 待实施时再确认的选型点

- SMTP 客户端：优先标准库；若选用第三方邮件库，需核对许可证与维护状态。
- QR 解码库：在 `github.com/makiuchi-d/gozxing`、`github.com/tuotoo/qrcode` 中按解码成功率与依赖体积实测后定。
- 微信群码启发式解析（6.2-C）是否在 P1 做：默认 **不做**，先落 7 天默认 + URL 参数解析；有真实样本后再加。
