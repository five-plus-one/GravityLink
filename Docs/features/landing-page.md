# 功能规格：落地页模板系统

## 功能描述

GravityLink 在落地域名上托管落地页，页面内容由预设模板 + 用户填写的内容数据共同决定。无可视化编辑器，管理端提供表单填写界面。

## 模板类型

### 1. liveqr（群活码展示页）

用途：展示群二维码，引导用户扫码加群。

内容字段（`content` JSON）：

```json
{
  "headline": "扫码加入交流群",
  "subtext": "群满自动切换，永久有效",
  "footer_text": "长按识别二维码",
  "theme_color": "#1677ff",
  "show_logo": true,
  "logo_url": "https://example.com/logo.png"
}
```

渲染结果：顶部标题 → 二维码图片（由轮换策略动态注入）→ 说明文字 → 页脚

二维码图片来源：由 Page Handler 在渲染时注入，非落地页静态配置。

### 2. redirect_notice（跳转提示页）

用途：在跳转前展示提示信息，适合需要告知用户"即将离开"的场景，或中转等待页。

内容字段：

```json
{
  "title": "即将跳转",
  "message": "您即将前往外部网站，请注意安全",
  "button_text": "继续访问",
  "countdown": 3,
  "theme_color": "#1677ff",
  "show_target_url": false
}
```

渲染结果：提示文字 → 倒计时（可选）→ 手动确认按钮

`countdown` 为 0 时显示按钮，不自动跳转；大于 0 时倒计时结束后自动跳转。

### 3. custom（自定义 HTML）

用途：完全自定义页面内容，适合高级用户。

内容字段：

```json
{
  "html": "<div>自定义 HTML 内容</div>",
  "inject_meta": true
}
```

`inject_meta = true` 时系统自动注入 viewport、charset 等基础 meta 标签。

**安全约束**：自定义 HTML 经过 sanitize 过滤，禁止 `<script>`、`on*` 事件属性、`javascript:` 协议。

### 4. kf（客服码展示页）

用途：客服活码展示页，在线状态徽章 + 客服二维码 + 微信号一键复制。

内容字段：

```json
{
  "headline": "添加专属客服",
  "subtext": "长按识别二维码，添加客服微信",
  "footer_text": "工作时间内回复更快",
  "theme_color": "#16a34a",
  "safety_tip": "谨防冒充客服的诈骗行为"
}
```

渲染时注入：
- **在线状态徽章**：根据链接 `online_schedule` JSON（每周 7 天 × 时段）判断当前是否在线；未配置时段视为全天在线
- **二维码图片**：由活码轮换策略选中目标后注入
- **微信号**：来自选中目标的 `wx_remark` 字段，展示页提供一键复制按钮

### 5. kami（卡密提取页）

用途：访客打开后点击按钮领取卡密，支持提取口令。

内容字段：

```json
{
  "announcement": "点击下方按钮领取卡密",
  "button_text": "立即领取",
  "theme_color": "#0f766e",
  "project_id": 1
}
```

`project_id` 绑定卡密项目。访客点击按钮后前端 POST `/api/v1/kami/{project_id}/issue`，服务端原子发码（FIFO + FOR UPDATE SKIP LOCKED）。口令、频率限制、重复提取策略由项目配置控制。

**注意**：kami 模板的活码链接无需配置二维码目标（创建时跳过 strategy 校验）。

## 模板渲染方式

Go 侧使用 `html/template` 渲染，模板文件存于 `backend/internal/web/templates/`：

```
templates/
├── liveqr.html
├── kf.html
├── redirect_notice.html
├── kami.html
├── custom.html
├── transit.html
├── unavailable.html
└── public.html
```

落地页渲染时：
1. 从 MySQL 读取 `landing_pages.content` JSON
2. 解析 JSON 到对应模板数据结构
3. `html/template` Execute → 输出 HTML 响应
4. 响应头设置 `Cache-Control: no-store`（避免浏览器缓存群活码页导致展示旧二维码）

## 管理端功能

- 落地页列表（按模板类型过滤）
- 创建/编辑：选模板类型 → 表单填写各字段 → 预览（内嵌 iframe）
- 落地页与链接关联：在链接创建/编辑时选择已有落地页，或快速新建
- 删除前校验：若有链接正在使用该落地页，拒绝删除并提示关联链接

## 扩展模板

后续可通过新增 HTML 模板文件 + 对应枚举值的方式扩展新模板，无需修改核心业务逻辑。

### 2026-09-06 访问状态校验

直接访问落地域名必须与入口短链保持一致：已停用返回 404，已过期返回 410；校验在模板加载和目标计数之前执行，拒绝访问不增加扫码计数。
