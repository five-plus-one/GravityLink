# 渠道码

短链接的投放变体：跳转时自动拼 UTM 参数，统计里可按来源对比。适合「同一落地页、多个投放口」。

## 和普通短链的区别

| 项目 | 短链接 | 渠道码 |
|------|--------|--------|
| 类型 | `short` | `channel` |
| 跳转参数 | 不自动拼参数 | 自动拼 UTM |
| 统计维度 | 通用 PV/UV | 额外按渠道聚合对比 |
| 落地页 / 中转 | 支持 | 同样支持 |
| 短码空间 | 共用全局唯一空间 | 同左 |

## UTM 字段

| 字段 | 含义 | 示例 |
|------|------|------|
| `utm_source` | 流量来源 | `wechat`、`weibo`、`douyin` |
| `utm_medium` | 媒介 | `social`、`cpc`、`email` |
| `utm_campaign` | 活动名 | `spring2026` |
| `utm_term` | 关键词 | `短链接工具` |
| `utm_content` | 创意区分 | `banner_a`、`video_01` |

全部可选；留空的字段不会拼进 URL。

## 跳转时发生什么

```text
访问 /{code}（渠道码）
  → 读出该码配置的 UTM
  → 拼接到目标 URL 查询串
  → 302

示例目标：https://example.com/page
结果：https://example.com/page?utm_source=wechat&utm_medium=social&utm_campaign=spring2026
```

**规则**：目标 URL 里**已有**的同名 UTM 参数不会被覆盖。你可以手动写死一个默认值，再用渠道码补其它字段。

## 使用步骤

1. 先想清楚落地页目标 URL（可带自己的查询参数）  
2. 为每个投放口建一条渠道码，例如：  
   - `wechat-moments` → source=wechat, medium=social, campaign=launch  
   - `weibo-banner` → source=weibo, medium=cpc, campaign=launch  
3. 把对应短链贴到各平台  
4. 统计页用「渠道来源对比」看各码 PV/UV  

创建时在类型里选「渠道链接」，并填写 UTM：

![创建渠道链接](https://img.assets.five-plus-one.com/img/2026/09/9d9ab670c80b4edc3db4b7342f1cf341.png)

列表也可按 `utm_source`、`utm_campaign` 过滤，方便活动收尾后批量下线。

## 什么时候用短链、什么时候用渠道码

| 需求 | 用哪个 |
|------|--------|
| 只是缩短，不关心来源 | 短链接 |
| 同一目标要拆来源 | 渠道码 |
| 要二维码轮换加群 | 群活码 |
| 跳转前必须展示页面 | 任意类型 + 落地页 |

工程规格原文：仓库 `Docs/features/channel-code.md`。
