# GravityLink UI 设计系统

本文档是 GravityLink 所有用户界面的唯一视觉与交互规范来源。任何前端/模板改动必须先符合本文档，再动代码。

## 1. 设计目标

- **一致性**：所有管理端页面、落地页、公开错误页共享同一套设计令牌。
- **可维护**：颜色、间距、字号全部来自 CSS 变量，禁止硬编码十六进制字面量。
- **可访问**：对比度 ≥ WCAG AA（4.5:1），交互元素最小触控 36px。
- **轻量**：管理端用 Naive UI，落地页用纯 CSS，不引入 Tailwind。

## 2. 技术选型

| 层 | 选型 | 理由 |
|---|---|---|
| 管理端组件库 | **Naive UI** | 全中文文档、TS 原生、CSS 变量主题、按需引入 |
| 状态管理 | **Pinia** | Vue 官方推荐 |
| 图表库 | **ECharts**（+ vue-echarts） | 国内事实标准，覆盖统计全部图表需求 |
| CSS 方案 | 原生 CSS + CSS 变量（design tokens） | 与 Naive UI 主题机制天然契合 |
| 落地页 | Go `html/template` + 共享 landing.css | SEO/首屏快，与 Docs/features/landing-page.md 对齐 |

## 3. 色彩令牌

管理端与落地页共用同一套变量，定义于 `frontend/admin/src/styles/tokens.css`（管理端）和 `backend/internal/web/landing/gravitylink-landing.css`（落地页）。

### 3.1 品牌色

| Token | 值 | 用途 |
|---|---|---|
| `--color-primary` | `#0f766e` | 主操作、链接、激活态 |
| `--color-primary-hover` | `#0d655e` | 主按钮 hover |
| `--color-primary-pressed` | `#0b544e` | 主按钮 active |
| `--color-primary-soft` | `#d3ece8` | 主色浅底（标签/徽章底） |
| `--color-accent` | `#14b8a6` | 品牌装饰、Logo 渐变 |

### 3.2 中性色（灰阶）

| Token | 值 | 用途 |
|---|---|---|
| `--color-bg-page` | `#f5f7f8` | 页面背景 |
| `--color-bg-surface` | `#ffffff` | 卡片/弹窗/表格底 |
| `--color-bg-subtle` | `#fafbfc` | 表头/分隔区 |
| `--color-bg-sidebar` | `#0f1c22` | 深色侧栏 |
| `--color-border` | `#dbe3e6` | 常规边框 |
| `--color-border-strong` | `#c2cdd2` | 强边框（输入框） |
| `--color-text-primary` | `#17222a` | 一级文字 |
| `--color-text-secondary` | `#48565e` | 二级文字 |
| `--color-text-tertiary` | `#7a8990` | 辅助文字 |
| `--color-text-inverse` | `#f7fafb` | 深底上的文字 |

### 3.3 功能色

| Token | 值 | 用途 |
|---|---|---|
| `--color-success` | `#0e7a5f` | 成功 |
| `--color-success-bg` | `#d9f2e8` | 成功浅底 |
| `--color-warning` | `#8a5a00` | 警告 |
| `--color-warning-bg` | `#fdeec8` | 警告浅底 |
| `--color-error` | `#b3261e` | 错误/危险 |
| `--color-error-bg` | `#fbe3e1` | 错误浅底 |
| `--color-info` | `#1d4ed8` | 信息 |
| `--color-info-bg` | `#dbe7fd` | 信息浅底 |

## 4. 字号阶梯

| Token | 值 | 用途 |
|---|---|---|
| `--font-size-xs` | `11px` | 辅助标签 |
| `--font-size-sm` | `12px` | 表头、徽章 |
| `--font-size-md` | `13px` | 正文 |
| `--font-size-base` | `14px` | 默认正文 |
| `--font-size-lg` | `16px` | 卡片标题 |
| `--font-size-xl` | `18px` | 区块标题 |
| `--font-size-2xl` | `22px` | 页面标题 |
| `--font-size-3xl` | `28px` | 数字指标 |

字重：`400` 常规 / `500` 强调 / `600` 标题 / `700` 数字。

字体栈：`Inter, "Segoe UI", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif`。

## 5. 间距栅格（4px 基线）

| Token | 值 |
|---|---|
| `--space-1` | `4px` |
| `--space-2` | `8px` |
| `--space-3` | `12px` |
| `--space-4` | `16px` |
| `--space-5` | `20px` |
| `--space-6` | `24px` |
| `--space-8` | `32px` |
| `--space-10` | `40px` |

禁止出现 13/15/21px 等非 4 倍数间距。

## 6. 圆角与阴影

| Token | 值 | 用途 |
|---|---|---|
| `--radius-sm` | `4px` | 输入框、小按钮 |
| `--radius-md` | `6px` | 按钮、下拉 |
| `--radius-lg` | `8px` | 卡片、弹窗 |
| `--radius-pill` | `999px` | 徽章、胶囊 |

| Token | 值 |
|---|---|
| `--shadow-sm` | `0 1px 2px rgba(15, 28, 34, .05)` |
| `--shadow-md` | `0 4px 12px rgba(15, 28, 34, .08)` |
| `--shadow-lg` | `0 12px 32px rgba(15, 28, 34, .12)` |

## 7. 布局约定

### 7.1 管理端

- 整体：`NLayout` 双栏（侧栏 232px + 主区）。
- 侧栏：深色 `--color-bg-sidebar`，含品牌、导航、公开入口、用户卡片。
- 顶部栏：面包屑 + 页面标题 + 页面级操作。
- 主区容器：最大宽 1440px，内边距 `--space-6`。
- 路由切换使用 `<Transition>` 淡入 150ms。

### 7.2 落地页 / 公开页

- 单列居中卡片，宽 `min(92vw, 420px)`。
- 卡片：`--color-bg-surface` + `--radius-lg` + `--shadow-md`。
- 主色按钮全宽，最小高 44px。

## 8. 组件约定（管理端）

| 场景 | 组件 | 规则 |
|---|---|---|
| 表格 | `NDataTable` | 必须配 `pagination`，空态用 `NEmpty` |
| 表单 | `NForm` + `NFormItem` | 必须配校验规则，错误就地显示 |
| 弹窗 | `NModal` | 提交失败错误显示在弹窗内（用 NForm 的 `validate` 或 `NAlert`） |
| 删除/危险确认 | `NPopconfirm` 或 `useDialog` | 禁止 `window.confirm` |
| 通知 | `useMessage` | 成功/失败统一走 toast，不写页面内 banner |
| 加载 | `NSpin` 或 `NSkeleton` | 表格用 `loading` 属性 |
| 状态徽章 | `NTag` | 通过 `type` 映射状态，禁用 `:class="rawStatus"` |
| 图标 | `@lucide/vue` | 尺寸统一 16/18/20 |
| 图表 | `v-chart`（vue-echarts） | 高度统一 320px |

## 9. 状态映射

| 后端状态 | NTag type | 文案 |
|---|---|---|
| `active` | `success` | 正常 |
| `pending` | `warning` | 待审核 |
| `disabled` | `default` | 已停用 |
| `expired` | `error` | 已过期 |
| `super_admin` | `error` | 超级管理员 |
| `admin` | `info` | 管理员 |
| `user` | `default` | 普通用户 |

## 10. 文案规范

- 所有面向用户的文案使用简体中文。
- 状态、角色、类型不直接展示后端原始值，必须经过映射。
- 按钮文案使用动词短语：「创建链接」「保存更改」「刷新」。
- 错误文案给出可操作的下一步，避免只写「操作失败」。

## 11. 响应式断点

| 断点 | 行为 |
|---|---|
| ≥1200px | 完整双栏布局 |
| 768–1199px | 侧栏收起为图标栏 |
| ≤767px | 侧栏改底部抽屉入口；表格改卡片列表；指标卡单列 |

## 12. 落地页特别约定

- 模板文件存放于 `backend/internal/templates/`，用 `html/template` 渲染。
- 共享样式 `/assets/landing/gravitylink-landing.css`（主色通过 Go 注入 `--color-primary`）。
- 所有落地页必须设置 `Cache-Control: no-store`（避免缓存旧二维码）。
- 落地页渲染必须记录访问日志（与入口域等价）。
- `custom` 模板 sanitize 必须过滤 `<script>`（大小写不敏感）、`on*` 事件属性、`javascript:` 协议。
