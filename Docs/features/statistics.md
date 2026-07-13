# 功能规格：统计系统

## 架构概览

统计系统分三层：

```
访问事件
   ↓
[热层] Redis 实时计数（PV/UV 原子累加，访问日志队列）
   ↓  （Worker 每 5 秒批量消费）
[温层] MySQL access_logs（原始日志，保留 90 天）
   ↓  （Worker 每小时整点聚合）
[冷层] MySQL stat_* 聚合表（长期保留，查询直接用此层）
```

## 数据采集

每次短链接跳转触发异步统计（不阻塞响应）：

### Redis 实时写入

```
Redis INCR  stat:pv:{link_id}:{yyyymmdd}           # 日 PV
Redis PFADD stat:uv:{link_id}:{yyyymmdd} {ip}      # 日 UV（HyperLogLog，误差 <1%）
Redis INCR  stat:hourly:{link_id}:{yyyymmdd}:{hh}  # 小时 PV
Redis LPUSH access:stream {json_payload}            # 访问日志入队
```

### access:stream 日志格式

```json
{
  "link_id": 123,
  "ip": "1.2.3.4",
  "ua": "Mozilla/5.0 (iPhone...)",
  "referer": "https://weixin.qq.com/",
  "visited_at": "2026-07-02T10:30:00Z",
  "via_transit": false
}
```

## Worker 处理

### log_consumer（每 5 秒）

1. `LRANGE access:stream 0 99`（最多取 100 条）
2. 并发解析：IP → 国家/省份/城市/ISP（ip2region）；UA → 设备/OS/浏览器
3. 批量 `INSERT INTO access_logs`
4. `LTRIM access:stream 100 -1`

### stat_flush（每小时整点）

1. 扫描 `stat:pv:*` 键（SCAN 命令，避免 KEYS 阻塞）
2. 批量 `GETDEL`，解析 link_id 和日期
3. `INSERT INTO stat_daily ... ON DUPLICATE KEY UPDATE pv = pv + ?`
4. 同步处理 `stat:uv:*`（`PFCOUNT` 后 `DEL`）、`stat:hourly:*`

### log_archiver（每天凌晨 2 点）

将 90 天前的 `access_logs` 记录移到 `access_logs_archive` 表（结构相同），然后从主表删除。

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
