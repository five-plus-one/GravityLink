# 功能规格：渠道码

## 功能描述

渠道码是短链接的一种变体，主要用于广告投放和流量来源追踪。与短链接的区别在于：跳转时自动拼接 UTM 参数，并在统计中区分渠道来源。

## 数据结构

渠道码复用 `links` 表（`type = 'channel'`），附加配置存于 `channel_configs`：

| 字段 | 说明 | 示例 |
|-----|------|------|
| `utm_source` | 流量来源 | `wechat`、`weibo` |
| `utm_medium` | 营销媒介 | `social`、`cpc` |
| `utm_campaign` | 营销活动 | `spring2026` |
| `utm_term` | 付费关键词 | `短链接工具` |
| `utm_content` | 区分广告内容 | `banner_a` |

## 跳转逻辑

```
访问 /{code}（渠道码）
  → 读取 channel_configs 中的 UTM 配置
  → 拼接参数到 target_url：
    https://example.com/page?utm_source=wechat&utm_medium=social&utm_campaign=spring2026
  → 302 跳转（UTM 参数不覆盖 target_url 中已有的同名参数）
```

## 与短链接的区别

| 项目 | 短链接 | 渠道码 |
|-----|--------|--------|
| 类型标识 | `short` | `channel` |
| 附加参数 | 无 | UTM 参数 |
| 统计维度 | 通用 | 额外按渠道聚合 |
| 落地页支持 | 是 | 是 |
| 中转域名 | 可选 | 可选 |

## 管理端功能

- 创建时额外填写 UTM 参数（非必填，空则不拼接）
- 列表页可按 utm_source、utm_campaign 过滤
- 统计页增加"渠道来源对比"视图（各渠道码 PV/UV 横向对比）
