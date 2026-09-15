# 落地页

跳转前在**落地域名**上展示的一页内容。没有拖拽编辑器，用「预设模板 + 表单字段」生成，保证轻、快、可缓存策略可控。

## 模板一览

| 模板 | 用途 | 典型挂载 |
|------|------|----------|
| `liveqr` | 展示群/活码二维码 | 群活码 |
| `redirect_notice` | 「即将跳转」提示 | 普通短链 / 中转等待 |
| `kf` | 客服码 + 在线状态 + 微信号 | 客服活码 |
| `kami` | 卡密领取按钮 | 卡密分发 |
| `custom` | 自定义 HTML（消毒） | 高级用户 |

### liveqr

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

渲染顺序：标题 → **动态注入的二维码** → 说明 → 页脚。二维码不是写死在 content 里，而是路由引擎每次请求选出目标后再注入。

### redirect_notice

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

- `countdown > 0`：倒计时结束自动跳转  
- `countdown = 0`：只显示按钮，由用户点击继续  
- `show_target_url`：是否展示目标地址（便于用户判断）

### kf（客服码）

```json
{
  "headline": "添加专属客服",
  "subtext": "长按识别二维码，添加客服微信",
  "footer_text": "工作时间内回复更快",
  "theme_color": "#16a34a",
  "safety_tip": "谨防冒充客服的诈骗行为"
}
```

额外注入：

- **在线徽章**：按链接的 `online_schedule`（每周 7 天 × 时段）判断；未配置视为全天在线  
- **二维码**：活码轮换选中的目标  
- **微信号一键复制**：来自目标的 `wx_remark`

### kami（卡密）

```json
{
  "announcement": "点击下方按钮领取卡密",
  "button_text": "立即领取",
  "theme_color": "#0f766e",
  "project_id": 1
}
```

- `project_id` 绑定卡密项目；访客点按钮后服务端原子发码  
- 口令、频率、重复提取规则由**项目设置**控制  
- kami 模板的活码链接**不需要**配二维码目标  
- 领取成功后页面保留本次卡密；失败可重试  

### custom

```json
{
  "html": "<div>你的 HTML</div>",
  "inject_meta": true
}
```

系统会 sanitize：禁止 `<script>`、`on*` 事件、`javascript:` 协议。`inject_meta=true` 时自动补 charset/viewport。

## 管理端操作

1. **落地页**列表 → 新建  
2. 选模板类型，按表单填字段，可内嵌预览  
3. 创建/编辑链接时选择该落地页（或在链接表单里快速新建）  
4. 删除前会校验是否仍被链接引用；有引用则拒绝并提示  

![落地页管理](https://img.assets.five-plus-one.com/img/2026/09/d149c05bff6adaa8205995014c6bcbb8.png)

创建落地页时先选模板再填内容：

![创建落地页](https://img.assets.five-plus-one.com/img/2026/09/95e1da3b9b2ff88e70cacb3d0b179355.png)

## 渲染与缓存

- Go `html/template` 服务端渲染，首屏快，利于 SEO  
- 群活码类页面 `Cache-Control: no-store`，避免缓存旧二维码  
- 落地域名必须在后台添加且 active；未登记 Host 直接 404  

## 访问状态

直接打开落地域也会校验链接状态：

- 停用 → 404  
- 过期 → 410  
- 校验在加载模板、累计扫码**之前**，拒绝访问不计数  

工程规格原文：仓库 `Docs/features/landing-page.md`。
