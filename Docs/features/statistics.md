# 功能规格：统计系统

## 架构概览

统计系统分三层：

```
访问事件
   ↓
[热层] Redis 实时计数（PV/UV 原子累加，访问日志队列）
   ↓  （Worker 每 5 秒批量消费）
[温层] MySQL access_logs（原始日志，保留 90 天）
   ↓  （StatFlusher 每分钟落盘非今日计数）
[冷层] MySQL stat_* 聚合表（长期保留，查询直接用此层）
```

> 读接口策略：非今日数据读 `stat_*` 聚合表；今日数据从 `access_logs` 读取，与访客明细一致。
> StatFlusher 只落盘「非今日」的 key，今日 key 保留供次日汇总，查询不重复叠加今日聚合。

## 数据采集

每次短链接跳转触发异步统计（不阻塞响应）：

### Redis 实时写入

```
Redis INCR   stat:pv:{link_id}:{yyyymmdd}           # 日 PV
Redis PFADD  stat:uv:{link_id}:{yyyymmdd} {ip}      # 日 UV（HyperLogLog，误差 <1%）
Redis INCR   stat:hourly:{link_id}:{yyyymmdd}:{hh}  # 小时 PV
Redis HINCRBY stat:dev:{link_id}:{yyyymmdd}         # 设备维度 Hash，field 为 "device|os|browser"
Redis HINCRBY stat:geo:{link_id}:{yyyymmdd}         # 地域维度 Hash，field 为 "country|province"（启用 IP 库时）
Redis LPUSH  access:stream {json_payload}           # 访问日志入队
```

入队前同步完成两类解析（解析结果随 payload 入队，不重复计算）：
UA → 设备/OS/浏览器；IP → 国家/省份/城市/ISP（未启用 IP 库时为空）。

### access:stream 日志格式

```json
{
  "link_id": 123,
  "ip": "1.2.3.4",
  "ua": "Mozilla/5.0 (iPhone...)",
  "referer": "https://weixin.qq.com/",
  "visited_at": "2026-07-02T10:30:00Z",
  "via_transit": false,
  "country": "中国",
  "province": "江苏省",
  "city": "南京市",
  "isp": "电信"
}
```

## Worker 处理

### log_consumer（每 5 秒）

1. `LRANGE access:stream 0 99`（最多取 100 条）
2. 解析 UA → 设备/OS/浏览器（`internal/service/useragent.go`，纯关键词匹配，零外部依赖；设备取值 mobile/tablet/desktop/bot/unknown，浏览器识别 WeChat/QQ/Edge/Firefox/Chrome/Safari）
3. 批量 `INSERT INTO access_logs`（含 country/province/city/isp 列）
4. `LTRIM access:stream 100 -1`

### stat_flush（每分钟）

1. `SCAN` 扫描 `stat:pv:*`、`stat:uv:*`、`stat:hourly:*`、`stat:dev:*`、`stat:geo:*`（避免 KEYS 阻塞）
2. 只处理「日期 < 今天」的 key（今日 key 保留供次日汇总）
3. 幂等覆盖写入（`ON DUPLICATE KEY UPDATE` 设为读取值），MySQL 写成功后才 `DEL` Redis key，失败下轮重试
4. `stat:pv/uv` → `stat_daily`（uv 同时写入 ip_count）；`stat:hourly` → `stat_hourly`；`stat:dev` → `stat_device`；`stat:geo` → `stat_geo`

### log_archiver（每天凌晨 2 点）

将 90 天前的 `access_logs` 记录移到 `access_logs_archive` 表（结构相同），然后从主表删除。

## 地域解析（ip2region，镜像内置）

基于 [ip2region](https://github.com/lionsoul2014/ip2region) v3 离线库实现，零网络依赖：

- 解析实现：`internal/service/geo.go`（xdb 格式 `国家|省份|城市|ISP|iso-alpha2-code`，缺失字段以 `0` 占位归一为空）
- 启用条件：环境变量 `GEO_DB_PATH` 指向 xdb 文件；未配置或文件加载失败时自动降级为不解析（记日志，不影响其他功能）
- 容器部署：镜像构建时将 ip2region IPv4 库打入 `/app/ip2region.xdb`（在 `/data` 卷之外），
  `docker-compose.yml` 默认 `GEO_DB_PATH=/app/ip2region.xdb`，开箱即用；
  如需自定义库可挂载 xdb 并覆盖 `GEO_DB_PATH`
- 数据流：`RecordAsync` 解析一次 → 写 `stat:geo:{link}:{day}` Hash（field `country|province`）与
  `access_logs` 的 country/province/city/isp 列 → StatFlusher 每分钟落盘 `stat_geo`

## 统计查询 API

所有查询走 MySQL 聚合表（非 access_logs 全扫），保证查询性能。

### 汇总数据

```
GET /api/v1/stats/{link_id}/summary
响应：{total_pv, total_uv, today_pv, today_uv, yesterday_pv}
```

计算方式：SUM(stat_daily.pv) / SUM(stat_daily.uv)

### 趋势数据（按天）

```
GET /api/v1/stats/{link_id}/daily?start=2026-06-01&end=2026-07-01
响应：[{date, pv, uv}, ...]，最多返回 90 天
```

### 小时分布

```
GET /api/v1/stats/{link_id}/hourly?date=2026-07-02
响应：[{hour: 0, pv: 12}, {hour: 1, pv: 3}, ...]，固定 24 条
```

### 地域分布

按国家、中国省份两个维度分别聚合（不再把省份与国家混在一条 label 里）：

```
GET /api/v1/stats/{link_id}/geo?start=&end=
GET /api/v1/stats/window?...   # window 响应内 geo 字段同结构
响应：
{
  "country":  [{"label": "中国", "value": 481}, {"label": "United States", "value": 158}, ...],
  "province": [{"label": "江苏省", "value": 108}, ...]
}
```

- `country`：按 `country` 列聚合，空值归为「未知」，用于世界地图着色与排行
- `province`：仅统计 `country = '中国'` 且 `province` 非空的记录，用于中国地图着色与排行
- 前端将国名/省名映射到 ECharts map GeoJSON 的 `name` 字段；「未知」与内网保留地址不上图

### 设备分布

```
GET /api/v1/stats/{link_id}/device?start=&end=
响应：
{
  "device": [{"label": "mobile", "value": 1234}, ...],
  "os": [{"label": "iOS", "value": 800}, ...],
  "browser": [{"label": "Chrome", "value": 600}, ...]
}
```

## 管理端统计看板

### 链接详情统计页

- 顶部卡片：总 PV、总 UV、今日 PV、今日 UV
- 折线图：近 30 天 PV/UV 趋势（双轴）
- 条形图：24 小时分布
- 地域分布：世界地图 + 中国地图切换，右侧 TOP10 来源排行（2026-09-14 替换原饼图）
- 饼图：设备类型 / OS 分布

### 全局看板（管理员）

- 系统今日总 PV / 活跃链接数 / 新增链接数
- TOP 10 链接（按今日 PV 排序）
- 系统近 7 天 PV 趋势

## 统计精度说明

| 指标 | 精度 | 说明 |
|-----|------|------|
| PV | 精确 | Redis INCR 原子操作 |
| UV | 约 99% | HyperLogLog，误差率 < 1% |
| IP 归属 | 省级 | ip2region 离线库 |
| UA 解析 | 设备/OS/浏览器 | 无法识别时标记 unknown |

### 概览访问看板（2026-09-06）

- 复用单链接 summary/daily/hourly 接口，默认选择列表第一条，支持搜索切换链接。明确展示单链接范围。
- 四指标为今日 PV、今日 UV、累计 PV、昨日 PV；不将逐日 UV 之和标为累计独立访客。
- 趋势支持近 7/30 天；按接口最后日期补齐缺失日为零；小时图标为今日小时分布。
- 请求切换立即清除旧数据，以请求序号丢弃过期结果；失败显示重试，成功零访问显示空态。
- 全站聚合由 `/stats/overview/*` 单接口完成（SQL SUM），不通过遍历所有链接制造大量统计请求。

### 统计页「全部链接」（2026-09-10）

- 统计页链接选择器固定含「全部链接」（`linkId = 0`），默认选中。
- 全部链接走 `/stats/overview/summary|daily|hourly|device|geo`；单链接仍走 `/stats/:id/*`。
- 空态判定不得用 `!linkId`（0 会被误判为未选择）；汇总数据区对 0 与具体链接一并展示。
- 全部链接与单链接均展示设备、操作系统、浏览器与地域分布；分布按所选日期读取历史聚合并叠加今日明细。

### 2026-09-06 历史聚合回归修复

GORM 聚合模型必须显式映射现有 stat_daily、stat_hourly、stat_device、stat_geo 表，禁止依赖默认复数命名。本修复不改数据库 schema；验收必须让真实定时 Worker 将昨日测试计数从 Redis 写入既有 MySQL 表，再通过统计 API 校验。


## 访客查询与时间范围（2026-09-13）

- `/stats/visitors` 联合 `access_logs` 与 `access_logs_archive`，支持链接、关键词、开始/结束、limit/offset。按访问时间和记录编号倒序稳定分页，空结果返回空数组。
- 完整时间采用 RFC3339，服务端转换为本地时区；范围为开始包含、结束不包含。仅日期的旧调用兼容整天查询，结束日期包含整天。
- 默认展示全部保留记录；快捷范围含今天、最近24小时、近7天、近30天。查询应用后翻页保持该筛选快照。
- 统计的日/小时/设备/地域图响应日期范围；跨度超过90天及无效范围返回明确错误。今日小时图在选择范围后改为该范围内按小时汇总。
- 归档使用显式列集合，兼容缺少 source_app 的旧归档表；这类旧归档记录的来源应用为空，已有来源地址仍可查看。
- 今日明细等待异步消费者写入后可见，通常为数秒；历史聚合仍按原有定时任务落盘。旧资料缺失的设备、来源或地域信息不推算补造。

## 精确时段与最近24小时

GET /api/v1/stats/window 接收带时区的 start/end 和可选 link_id，按左闭右开区间查询近期及归档访问明细，返回：

- `pv` / `uv`：区间访问量与去重 IP
- `daily`：按日 PV/UV（区间内补齐空日为 0）
- `device`：设备 / 操作系统 / 浏览器
- `geo`：`country` / `province`（仅中国）/ `city`（仅中国，TOP）
- `source`：`app`（来源 APP，空归为「直接访问」）/ `referer`（域名 TOP）

范围最多 90 天。

### 管理端统计页信息架构（2026-09-14）

顶部保留链接选择与日期范围，下方选项卡分区：

| Tab | 内容 |
|-----|------|
| 趋势 | 区间 KPI、每日 PV/UV、最近 24 小时 |
| 设备 | 设备类型饼图、OS / 浏览器条形图 |
| 地域 | 世界/中国大地图 + 国家/省份/城市排行（含占比） |
| 来源 | 来源应用、Referer 域名 TOP |

地图资源本地化：`frontend/admin/public/maps/{world,china}.json`，国名/省名映射见 `geoNames.ts`。
