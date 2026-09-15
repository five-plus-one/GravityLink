# 服务器部署

面向「服务器上已有 MySQL 与 Redis，只部署 GravityLink 应用」的完整说明。

## 前置条件

| 项目 | 要求 | 检查 |
|------|------|------|
| Docker | 20.10+ | `docker --version` |
| Compose | v2 | `docker compose version` |
| MySQL | 8.0+，网络可达 | 能 `mysql -h ...` 或 telnet 3306 |
| Redis | 网络可达 | 能 `redis-cli -h ... ping` |
| 磁盘 | 镜像约数十 MB + 卷数据 | |
| 域名 | 强烈建议 | 用于短链与 HTTPS |

## 目录与 Compose

```bash
mkdir -p /opt/gravitylink/deploy
cd /opt/gravitylink/deploy
```

```yaml
name: gravitylink

services:
  gravitylink:
    image: 5plus1/gravitylink:latest
    restart: unless-stopped
    environment:
      APP_ENV: production
      HTTP_ADDR: :8080
      ADMIN_HTTP_ADDR: :8081
      CONFIG_FILE: /data/gravitylink.json
      SETUP_DEFAULT_MYSQL_HOST: your-mysql-host
      SETUP_DEFAULT_REDIS_HOST: your-redis-host
      GEO_DB_PATH: /app/ip2region.xdb
      TZ: Asia/Shanghai
    ports:
      - "18080:8080"
      - "18081:8081"
    volumes:
      - gravitylink_data:/data

volumes:
  gravitylink_data:
```

### 环境变量含义

| 变量 | 作用 |
|------|------|
| `HTTP_ADDR` | 短链/落地页监听（容器内） |
| `ADMIN_HTTP_ADDR` | 管理后台监听（容器内） |
| `CONFIG_FILE` | 初始化后配置文件路径，必须在卷内否则重启丢失 |
| `SETUP_DEFAULT_*` | 只影响向导预填，**不**强制成为真实连接 |
| `GEO_DB_PATH` | IP 库；镜像已内置，通常不用改 |
| `TZ` | 统计日切、过期时间依赖时区，建议 `Asia/Shanghai` |

更多变量见[环境变量](/reference/env)。

### 端口映射

| 宿主机 | 容器 | 用途 | 对外 |
|--------|------|------|------|
| 18080 | 8080 | 公网短链 | 是 |
| 18081 | 8081 | 管理后台 | 建议否，或 HTTPS+白名单 |

可改成其它宿主机端口，例如 `"28080:8080"`，反代时对应修改。

## 防火墙

```bash
# 短链
sudo ufw allow 18080/tcp

# 后台：限制来源
# sudo ufw allow from 办公IP to any port 18081
```

生产更推荐：只暴露 443，后台用 Nginx + IP 白名单。

## 启动与健康

```bash
docker compose up -d
docker compose ps
docker compose logs -f gravitylink
```

成功日志：

```text
gravitylink server starting addr=:8080
gravitylink admin frontend starting addr=:8081
```

可再访问：

- `http://IP:18080/`（未知 Host 会 404，属正常）  
- `http://IP:18081/`（应出现后台或初始化向导）

## 初始化系统

访问 `http://IP:18081`，完整步骤见[初始化向导](./setup)。

MySQL 连接参数建议：

```text
charset=utf8mb4&parseTime=True&loc=Local
```

若 MySQL 与 GravityLink 不在同一台机：

```sql
CREATE USER 'gravitylink'@'%' IDENTIFIED BY '强密码';
GRANT ALL PRIVILEGES ON gravitylink.* TO 'gravitylink'@'%';
FLUSH PRIVILEGES;
```

同时检查 `bind-address`、安全组、防火墙。

## IP 地理库

镜像已内置 ip2region IPv4（`/app/ip2region.xdb`），地域统计开箱可用。

自定义库：

```yaml
environment:
  GEO_DB_PATH: /data/ip2region.xdb
volumes:
  - ./ip2region.xdb:/data/ip2region.xdb:ro
```

然后 `docker compose up -d` 重建。

## 数据落在哪

**Docker 卷 `gravitylink_gravitylink_data`**

- 系统配置 `gravitylink.json`  
- 上传素材 `/uploads/`

**你的 MySQL**

- 链接、域名、落地页、统计、访客日志等业务表  

备份两边都要做，见[备份与升级](./ops)。

## 多实例注意

Compose 项目名固定 `name: gravitylink`。从仓库根目录或 `deploy/` 启动都必须复用同一项目名，否则可能开出第二套容器抢端口。

## 完整运维命令

```bash
cd /opt/gravitylink/deploy

docker compose ps
docker compose logs -f gravitylink
docker compose restart gravitylink

# 升级
docker compose pull gravitylink
docker compose up -d

# 停止（保留卷）
docker compose down
```

## 故障排查

| 问题 | 怎么查 |
|------|--------|
| 容器反复重启 | `docker compose logs gravitylink` |
| 初始化连不上 MySQL | 授权、bind-address、容器能否访问宿主机/远程 3306 |
| 短链 404 | 域名是否 active；Nginx 是否指到 18080；短码是否存在 |
| 后台白屏 | 强刷缓存；18081 是否通；是否只反代了静态而漏了 SPA fallback |
| 端口占用 | 改 ports 映射 |
| 地域全是空 | 确认 `GEO_DB_PATH` 文件存在且可读 |

下一步：[域名与 HTTPS](./nginx)、[域名管理](./domains)。
