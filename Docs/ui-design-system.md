# GravityLink UI 设计系统

本文档是 GravityLink 所有用户界面的唯一视觉与交互规范来源。任何前端/模板改动必须先符合本文档，再动代码。

## 1. 设计目标

- **一致性**：所有管理端页面、落地页、公开错误页共享同一套设计令牌。
- **可维护**：颜色、间距、字号全部来自 CSS 变量，禁止硬编码十六进制字面量。
- **可访问**：对比度 ≥ WCAG AA（4.5:1），交互元素最小触控 36px。
- **轻量**：管理端用 Naive UI，落地页用纯 CSS，不引入 Tailwind。
- **产品感**：管理端采用浅色导航、蓝色主操作、柔和渐变背景和高圆角卡片；信息层级应先呈现页面标题与摘要，再呈现操作区和数据内容。

### 1.1 2026 产品化改造基线

- 参考界面只用于视觉方向和交互密度，不照搬其品牌、会员、支付等业务模块。
- 管理端保持 GravityLink 品牌名称和现有功能边界，第一阶段优先改造全局外壳、概览、统计、链接/活码与系统设置。
- 页面结构统一为：全局顶栏 → 页面标题/副标题 → 指标或业务卡片 → 表格/图表。
- 禁止在构建时直接以正在服务的 `dist` 目录作为可验收环境；部署应由镜像内嵌静态资源或原子目录切换完成。

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
| `--color-primary` | `#2787f5` | 主操作、链接、激活态 |
| `--color-primary-hover` | `#1877e5` | 主按钮 hover |
| `--color-primary-pressed` | `#1267c9` | 主按钮 active |
| `--color-primary-soft` | `#e7f2ff` | 主色浅底（标签/徽章底） |
| `--color-accent` | `#62a7ff` | 品牌装饰、Logo 渐变 |

### 3.2 中性色（灰阶）

| Token | 值 | 用途 |
|---|---|---|
| `--color-bg-page` | `#f3f7ff` | 页面背景 |
| `--color-bg-surface` | `#ffffff` | 卡片/弹窗/表格底 |
| `--color-bg-subtle` | `#f7faff` | 表头/分隔区 |
| `--color-bg-sidebar` | `rgba(255,255,255,.92)` | 浅色侧栏 |
| `--color-border` | `#e5ecf5` | 常规边框 |
| `--color-border-strong` | `#ccd8e8` | 强边框（输入框） |
| `--color-text-primary` | `#14233a` | 一级文字 |
| `--color-text-secondary` | `#53647a` | 二级文字 |
| `--color-text-tertiary` | `#8997a9` | 辅助文字 |
| `--color-text-inverse` | `#ffffff` | 深底上的文字 |

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
| `--radius-sm` | `8px` | 输入框、小按钮 |
| `--radius-md` | `10px` | 按钮、下拉 |
| `--radius-lg` | `16px` | 卡片、弹窗 |
| `--radius-pill` | `999px` | 徽章、胶囊 |

| Token | 值 |
|---|---|
| `--shadow-sm` | `0 2px 8px rgba(37, 91, 154, .05)` |
| `--shadow-md` | `0 10px 28px rgba(37, 91, 154, .08)` |
| `--shadow-lg` | `0 20px 56px rgba(37, 91, 154, .14)` |

## 7. 布局约定

### 7.1 管理端

- 整体：`NLayout` 双栏（侧栏 232px + 主区）。
- 侧栏：半透明浅色 `--color-bg-sidebar`，含品牌、导航、公开入口、用户卡片；激活菜单使用浅蓝底和左侧蓝色标记。
- 顶部栏：菜单折叠按钮 + 当前用户；页面标题与副标题放在主内容顶部。
- 主区容器：最大宽 1440px，内边距 `--space-6`。
- 主区背景：至少两层低饱和蓝/紫/青径向渐变，不使用纯灰平铺。
- 路由切换使用 `<Transition>` 淡入 150ms。
- 个人中心只展示当前认证体系可提供的真实字段；未实现会员、积分、订单时不得使用占位数字模拟商业数据。

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

### 2026-09-05 域名管理细化

- 域名页提供总数、正常域名和停用域名摘要；按域名/备注关键词与用途联合筛选，并展示筛选数量。
- 无数据与无筛选结果分别呈现，后者提供清空筛选入口。
- 添加域名弹窗适配窄屏，说明入口、中转、落地三种已支持用途；保留单域名单用途的现有接口契约。

### 2026-09-06 概览访问看板

- 访问看板位于资源摘要前，使用四指标、双趋势卡片布局；窄屏改为单列。
- 图表配色读取现有设计令牌。链接选择、刷新和时间范围控件带可访问名称。
- 概览资源读取按权限请求，各资源失败独立呈现；未知数量显示横线而非零。

- 2026-09-06：活码 round_robin 在界面标为顺序阈值模式，说明不限次数的目标不会自动切到下一项，与当前后端语义保持一致。

# 2026-09-07 路由切换修复

后台 HTML 入口发送 Cache-Control: no-store；缺失的 assets 文件返回 404，不回退为 HTML，防止部署更新后错误脚本响应被误当作页面加载。

RouterView 使用按 route.path 区分的真实 DOM 容器，不使用 out-in 过渡等待页面卸载，避免多根节点或 Teleport 弹窗使路由内容卡在离场状态。验收需从分享卡片连续切换域名、落地页和统计，不能只测试直接打开 URL。

异步 JS/CSS 资源加载失败时必须显示可恢复的错误页，明确提示服务可能尚未启动或浏览器缓存了旧版本，并提供重新加载入口；禁止让预加载错误表现为无内容白屏。

# 2026-09-11 界面密度与品牌降噪

- **品牌标识只出现在页脚**：管理端侧边栏不渲染品牌 Logo/名称，避免喧宾夺主；页脚保留 20px 渐变方标 + "GravityLink" 文字链接。登录页、初始化页不受此约束。
- **滚动条统一自绘**：全局在 `base.css` 定义细滚动条（WebKit `::-webkit-scrollbar` 宽高 10px + Firefox `scrollbar-width: thin`），禁止依赖浏览器默认样式。Naive UI 内部 NScrollBar 已用类选择器隐藏原生滚动条，不受全局规则影响。
- **服务端分页表格必须加 `remote`**：Naive UI `NDataTable` 未设置 `remote` 时会忽略 `pagination.itemCount`，用本地数据长度计算页数，导致远端分页不显示翻页按钮（访客记录页踩坑）。`itemCount` + `remote` 必须成对出现；总条数通过 pagination `prefix` 展示。
- **列表页工具栏与标题分离**：卡片头部只放标题与统计摘要；搜索、筛选、批量操作、主操作按钮放卡片正文顶部的独立工具栏行（flex-wrap + 8px 间距），窄屏时搜索框整行、筛选控件半宽自适应。
- **表格长文本禁止换行撑高**：名称、URL 等长文本列使用 `ellipsis: { tooltip: true }`，保持行高一致；`scroll-x` 不得小于各列 width 之和。
- **移动端卡片内边距**：≤640px 时 `n-card` 左右内边距降为 16px，为表格和筛选控件腾出空间。
