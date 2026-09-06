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

> 读接口策略：非今日数据读 `stat_*` 聚合表；今日数据实时叠加 Redis 计数。
> StatFlusher 只落盘「非今日」的 key，今日 key 保留在 Redis 提供实时值，两者无缝衔接。

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
2. 只处理「日期 < 今天」的 key（今日 key 保留给读接口叠加实时值）
3. 幂等覆盖写入（`ON DUPLICATE KEY UPDATE` 设为读取值），MySQL 写成功后才 `DEL` Redis key，失败下轮重试
4. `stat:pv/uv` → `stat_daily`（uv 同时写入 ip_count）；`stat:hourly` → `stat_hourly`；`stat:dev` → `stat_device`；`stat:geo` → `stat_geo`

### log_archiver（每天凌晨 2 点）

将 90 天前的 `access_logs` 记录移到 `access_logs_archive` 表（结构相同），然后从主表删除。

## 地域解析（ip2region，可选启用）

基于 [ip2region](https://github.com/lionsoul2014/ip2region) v3 离线库实现，零网络依赖：

- 解析实现：`internal/service/geo.go`（xdb 格式 `国家|省份|城市|ISP|iso-alpha2-code`，缺失字段以 `0` 占位归一为空）
- 启用条件：环境变量 `GEO_DB_PATH` 指向 xdb 文件；未配置或文件加载失败时自动降级为不解析（记日志，不影响其他功能）
- 容器部署：`docker-compose.yml` 已预设 `GEO_DB_PATH=/data/ip2region.xdb`，下载
  [ip2region_v4.xdb](https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb)
  放入 `/data` 卷并重启容器即生效
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

```
GET /api/v1/stats/{link_id}/geo?start=&end=
响应：[{province, pv, percentage}, ...]，按 PV 降序，TOP 20
```

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
- 地图/表格：省份 TOP 10
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
- 全站聚合仍属后续能力，不通过遍历所有链接制造大量统计请求。

### 2026-09-06 历史聚合回归修复

GORM 聚合模型必须显式映射现有 stat_daily、stat_hourly、stat_device、stat_geo 表，禁止依赖默认复数命名。本修复不改数据库 schema；验收必须让真实定时 Worker 将昨日测试计数从 Redis 写入既有 MySQL 表，再通过统计 API 校验。

