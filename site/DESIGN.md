# GravityLink 文档站 — 设计说明

内部设计稿，指导 `site/` 的实现；不是给终端用户看的页面。

## 模式

- **首页**：Expressive — 有记忆点的引力场视觉。
- **教程/功能页**：Convention — 侧栏、可读、不抢戏。

## 风格锚点

深空观测台 / 轨道任务控制台。产品名 GravityLink 的「引力」直接做成首页签名元素：短链、渠道码、群活码、落地页像天体一样被中心质量捕获。避免通用 SaaS 渐变 Hero。

## 色板

| Token | 值 | 用途 |
|---|---|---|
| `--void` | `#070B14` | 首页深空底 |
| `--ink` | `#E8EEF7` | 深底正文 |
| `--paper` | `#F6F8FB` | 文档页浅底 |
| `--text` | `#14233A` | 文档正文（对齐产品） |
| `--well` | `#2787F5` | 品牌主色（对齐管理端） |
| `--orbit` | `#62A7FF` | 轨道高亮、链接 |
| `--dust` | `#8B9BB4` | 次级文字 |
| `--event-horizon` | `#0A1628` | 引力井中心 |

## 字体

- Display / Body：`"Segoe UI", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif`（中文文档站优先系统字体，不依赖外网字体 CDN）
- Mono：`"Cascadia Code", Consolas, ui-monospace, monospace` — 短码、端口、命令

## 布局

- 首页：全幅引力场 Hero（标题 + 副题 + 双 CTA）→ 三条轨道入口（部署 / 功能 / 配置）→ 四能力星座卡片 → 极简页脚
- 文档页：标准 VitePress 浅色侧栏；最大内容宽约 760px

## 签名元素

`GravityField` 组件：Canvas 绘制引力井 + 多个节点沿椭圆轨道缓慢运行，鼠标靠近轻微偏移。加载时一次淡入；尊重 `prefers-reduced-motion`。

## 目录

```
site/
├── index.md          # 自定义首页
├── guide/            # 快速开始、部署、反代、配置
├── features/         # 短链 / 渠道码 / 群活码 / 落地页 / 统计
└── reference/        # 环境变量、架构摘要
```

## 内容原则

面向自托管运维者，不搬内部验收报告。每页必须有：问题场景、字段/表、可复制命令或 YAML、状态与错误码、排查表。禁止「一句话模块介绍」式空页。命令可复制，表格写清占位符含义。教程页用「你」直接操作的祈使句。
