# 快速开始

目标：在一台已装 Docker、能连 MySQL 和 Redis 的服务器上，把 GravityLink 跑到可创建第一条短链。

## 你将得到

| 组件 | 默认端口 | 谁用 |
|------|----------|------|
| 短链服务 | 宿主机 `18080` → 容器 `8080` | 公网用户访问短链/落地页 |
| 管理后台 | 宿主机 `18081` → 容器 `8081` | 你登录管理 |
| 数据 | Docker 卷 + 你的 MySQL | 配置/上传在卷，业务表在 MySQL |

## 三步跑通

### 1. 检查依赖

```bash
docker --version          # 需要 20.10+
docker compose version    # 需要 Compose v2
```

再确认 MySQL 8.0+、Redis 可网络连通。

### 2. 写 Compose 并启动

```bash
mkdir -p /opt/gravitylink/deploy
cd /opt/gravitylink/deploy
```

创建 `docker-compose.yml`：

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
      TZ: Asia/Shanghai
    ports:
      - "18080:8080"
      - "18081:8081"
    volumes:
      - gravitylink_data:/data

volumes:
  gravitylink_data:
```

启动：

```bash
docker compose up -d
docker compose logs -f gravitylink
```

成功时类似：

```text
gravitylink server starting addr=:8080
gravitylink admin frontend starting addr=:8081
```

### 3. 打开后台完成初始化

浏览器访问 `http://服务器IP:18081`，按向导：

1. 填 MySQL / Redis，点「测试连接」  
2. 建本地管理员（或接 Logto）  
3. 填「公开访问地址」后完成初始化  

细节字段见[初始化向导](./setup)。

登录成功后进入**数据概览**，可看到访问指标与资源概况：

![数据概览](https://img.assets.five-plus-one.com/img/2026/09/f1379f895113666cf8c8638cee04e798.png)

## 初始化之后必做

1. 后台「域名」添加**入口域名**（如 `s.yourdomain.com`）  
2. DNS 解析到服务器，Nginx 反代到 `18080`  
3. 域名状态变 active 后，创建短链并手机点一次  
4. 统计页确认有记录  

域名与 Nginx 见[域名与 HTTPS](./nginx)、[域名管理](./domains)。

## 防火墙最低配置

```bash
sudo ufw allow 22/tcp          # SSH（若你已用 ufw）
sudo ufw allow 18080/tcp       # 或只开 443，由 Nginx 终止 TLS
# 18081 不要对整个公网敞开
```

## 下一步读什么

- 完整端口、备份、故障表 → [服务器部署](./deploy)  
- 业务功能怎么用 → [功能总览](/features/)  
- 素材、卡密、分享卡片 → [运营工具](/features/materials)  
- 环境变量清单 → [环境变量](/reference/env)
