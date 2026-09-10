# GravityLink 服务器部署教程

> 适用场景：服务器已有 MySQL 和 Redis 实例，只需部署 GravityLink 应用。

## 前置条件

- 服务器已安装 Docker 20.10+ 和 Docker Compose v2
- 已有可连通的 MySQL 8.0+ 实例
- 已有可连通的 Redis 实例
- 有一个域名指向服务器（用于短链接访问，可选但推荐）

检查 Docker 版本：

```bash
docker --version
docker compose version
```

---

## 第一步：创建部署目录

```bash
mkdir -p /opt/gravitylink/deploy
cd /opt/gravitylink/deploy
```

## 第二步：创建 docker-compose.yml

创建文件 `/opt/gravitylink/deploy/docker-compose.yml`：

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
      TZ: Asia/Shanghai
    ports:
      - "18080:8080"
      - "18081:8081"
    volumes:
      - gravitylink_data:/data

volumes:
  gravitylink_data:
```

**需要修改的地方**：

| 占位符 | 说明 | 示例 |
|--------|------|------|
| `your-mysql-host` | MySQL 地址（初始化向导的默认值） | `127.0.0.1` 或 `mysql.example.com` |
| `your-redis-host` | Redis 地址（初始化向导的默认值） | `127.0.0.1` 或 `redis.example.com` |

> 这两个只是初始化向导的**预填默认值**，实际连接信息在向导里填写，不影响运行。

**端口说明**：

| 端口 | 用途 | 是否需要对外 |
|------|------|-------------|
| 18080 | 公网短链接服务 | 是（用户访问短链） |
| 18081 | 管理后台 | 建议仅内网或加防火墙 |

## 第三步：配置防火墙

```bash
# 开放公网短链接端口
sudo ufw allow 18080/tcp

# 管理后台建议仅限内网访问，或通过 Nginx 反代 + HTTPS
# sudo ufw allow from your-ip to any port 18081
```

## 第四步：启动服务

```bash
cd /opt/gravitylink/deploy
docker compose up -d
```

首次运行会自动拉取镜像（约 53MB）。查看启动状态：

```bash
docker compose ps
docker compose logs -f gravitylink
```

看到以下日志表示启动成功：

```
gravitylink server starting addr=:8080
gravitylink admin frontend starting addr=:8081
```

## 第五步：初始化系统

打开浏览器访问 `http://服务器IP:18081`，进入初始化向导。

### 步骤 5.1：连接数据服务

填写你的 MySQL 和 Redis 连接信息：

**MySQL**（你的已有实例）：

| 字段 | 填写内容 |
|------|---------|
| Host | MySQL 地址（如 `127.0.0.1` 或远程地址） |
| Port | MySQL 端口（默认 `3306`） |
| Database | 数据库名（如 `gravitylink`，不存在会自动创建） |
| User | 数据库用户名 |
| Password | 数据库密码 |
| 连接参数 | `charset=utf8mb4&parseTime=True&loc=Local` |

**Redis**（你的已有实例）：

| 字段 | 填写内容 |
|------|---------|
| Host | Redis 地址 |
| Port | Redis 端口（默认 `6379`） |
| DB | `0`（或你指定的库） |
| Password | Redis 密码（无密码留空） |

> **重要**：必须先点击「测试连接」确认可达，才能进入下一步。测试通过后配置会自动保存。

### 步骤 5.2：建立管理员身份

两种方式二选一：

**方式 A：本地账号（简单）**
- 输入用户名（至少 3 位）和密码（至少 10 位）
- 点击「创建管理员身份」

**方式 B：Logto OIDC（企业级）**
- 填写 Logto 的 Issuer、App ID、Audience
- 点击「检查配置并使用 Logto 验证」
- 跳转到 Logto 完成登录后自动返回

### 步骤 5.3：确认并启用

- 确认 MySQL、Redis、管理员信息无误
- 填写「公开访问地址」：如 `http://你的域名:8080` 或 `http://服务器IP:18080`
- 点击「完成初始化」

初始化完成后系统自动进入管理后台。

## 第六步：配置域名（推荐）

初始化完成后，到「域名」页面添加入口域名：

| 字段 | 示例 |
|------|------|
| 域名 | `s.yourdomain.com` |
| 类型 | 入口（entry） |
| 协议 | https |

然后在你的 DNS 服务商添加 A 记录指向服务器 IP。

如果你用 Nginx 反代，参考配置：

```nginx
server {
    listen 443 ssl;
    server_name s.yourdomain.com;

    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

管理后台反代：

```nginx
server {
    listen 443 ssl;
    server_name admin.yourdomain.com;

    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:18081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## IP 地理库（默认已启用）

镜像构建时已内置 ip2region IPv4 库（`/app/ip2region.xdb`），`GEO_DB_PATH` 默认指向该路径。  
部署后访客地域直接可用，无需手动复制文件。

可选：使用自定义库时，将 xdb 挂载进数据卷并覆盖环境变量：

```yaml
environment:
  GEO_DB_PATH: /data/ip2region.xdb
volumes:
  - ./ip2region.xdb:/data/ip2region.xdb:ro
```

然后 `docker compose up -d` 重建容器即可。

---

## 常用运维命令

```bash
cd /opt/gravitylink/deploy

# 查看状态
docker compose ps

# 查看日志
docker compose logs -f gravitylink

# 重启
docker compose restart gravitylink

# 更新到最新版本
docker compose pull gravitylink
docker compose up -d

# 停止
docker compose down

# 停止并删除数据（慎用！会清空所有配置和业务数据）
docker compose down -v
```

## 数据备份

```bash
# 备份应用配置和上传文件
docker run --rm -v gravitylink_gravitylink_data:/data -v /opt/backup:/backup alpine \
  tar czf /backup/gravitylink-data-$(date +%Y%m%d).tar.gz -C /data .

# 备份 MySQL（你已有实例，用你自己的备份方式）
mysqldump -h your-mysql-host -u gravitylink -p gravitylink > gravitylink-$(date +%Y%m%d).sql
```

## 故障排查

| 问题 | 排查方法 |
|------|---------|
| 容器启动失败 | `docker compose logs gravitylink` |
| 初始化时 MySQL 连不上 | 确认 MySQL 允许来自容器 IP 的连接（`bind-address` 和用户授权） |
| 短链访问 404 | 检查域名是否已在「域名」页面添加且状态为 active |
| 管理后台白屏 | 硬刷新（Ctrl+Shift+R），检查 18081 端口是否可达 |
| 端口被占用 | 修改 compose 中的端口映射，如 `"18080:8080"` |

### MySQL 远程连接授权

如果 MySQL 和 GravityLink 不在同一台机器，需要授权远程访问：

```sql
-- 在 MySQL 中执行
CREATE USER 'gravitylink'@'%' IDENTIFIED BY '你的密码';
GRANT ALL PRIVILEGES ON gravitylink.* TO 'gravitylink'@'%';
FLUSH PRIVILEGES;
```

---

## 完整目录结构

```
/opt/gravitylink/
└── deploy/
    └── docker-compose.yml
```

数据持久化在 Docker 卷 `gravitylink_gravitylink_data` 中，包含：
- 系统配置文件（`gravitylink.json`）
- 上传的图片素材（`/uploads/`）

IP 地理库（`ip2region.xdb`）已打进应用镜像（`/app/ip2region.xdb`），不在数据卷中。
