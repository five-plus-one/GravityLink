# 备份与升级

## 日常命令

```bash
cd /opt/gravitylink/deploy

docker compose ps
docker compose logs -f gravitylink
docker compose restart gravitylink

# 拉新镜像滚动升级
docker compose pull gravitylink
docker compose up -d

# 停止（保留数据卷）
docker compose down
```

::: danger 会丢数据的命令
`docker compose down -v` 删除数据卷，**配置和上传文件一起没**。业务表在 MySQL，仍要单独备份；卷删了不可恢复。
:::

## 备份范围

| 数据 | 位置 | 方式 |
|------|------|------|
| 配置、上传文件 | 卷 `gravitylink_gravitylink_data` | 打包 `/data` |
| 链接、域名、统计、明细 | 你的 MySQL | `mysqldump` 或厂商备份 |
| IP 库 | 镜像内 `/app/ip2region.xdb` | 一般不用备，可再下载 |

### 备份数据卷

```bash
docker run --rm \
  -v gravitylink_gravitylink_data:/data \
  -v /opt/backup:/backup \
  alpine tar czf /backup/gravitylink-data-$(date +%Y%m%d).tar.gz -C /data .
```

### 备份 MySQL

```bash
mysqldump -h your-mysql-host -u gravitylink -p \
  --single-transaction --routines --triggers \
  gravitylink > gravitylink-$(date +%Y%m%d).sql
```

访问明细表可能很大，可单独策略：全量每周 + binlog/增量。

## 恢复

1. 停容器：`docker compose down`  
2. 恢复卷内容到 `gravitylink_data`  
3. 恢复 MySQL dump  
4. `docker compose up -d`  
5. 登录后台，建一条测试短链验证  

恢复步骤因环境而异，**先在测试机演练**。

## 升级流程

1. 备份卷 + MySQL  
2. `docker compose pull gravitylink`  
3. `docker compose up -d`  
4. 看日志无报错  
5. 手机点一条旧短链，确认统计仍在  

跨大版本时阅读仓库 `Docs/migration.md`（若有迁移 CLI）。

## 卡密与并发注意

卡密项目有口令、去重、并发领取规则。升级不要在业务高峰做；导入大包卡密前先备份对应项目。

## 故障排查

| 现象 | 排查 |
|------|------|
| 容器起不来 | `docker compose logs gravitylink` |
| 初始化连不上 MySQL | 授权、网络、bind-address |
| 短链 404 | 域名 active？DNS/Nginx/端口？ |
| 后台白屏 | 强刷；18081 可达；反代是否只处理了 `/` |
| 端口被占 | 改映射 |
| 统计不涨 | 是否还在异步延迟内；Redis 是否正常；Worker 日志 |
| 地域空 | GEO_DB_PATH 文件是否存在 |

## 监控建议

- 探测后台健康 URL 与短链一条固定测试码  
- 日志保留 ≥ 7 天  
- 每月校验备份能解开、能导入  
- Redis 持久化策略按你的 RPO 选择  

## 安全清单

- 18081 不对公网裸奔  
- MySQL/Redis 仅内网  
- 定期改数据库密码（改完到设置里更新并测试连接）  
- 镜像来源可信，固定 tag 比 `latest` 更利于回滚  
