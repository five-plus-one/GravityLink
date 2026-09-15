# 环境变量

业务连接在初始化向导里配置后写入 `CONFIG_FILE`。环境变量主要用于进程行为与向导预填。

## 应用运行

| 变量 | 默认/示例 | 说明 |
|------|-----------|------|
| `APP_ENV` | `production` | 运行环境 |
| `HTTP_ADDR` | `:8080` | 短链/落地页监听 |
| `ADMIN_HTTP_ADDR` | `:8081` | 管理后台与初始化监听 |
| `CONFIG_FILE` | `/data/gravitylink.json` | 持久化配置；必须在数据卷内 |
| `TZ` | `Asia/Shanghai` | 时区；影响日切、过期、小时统计 |
| `GEO_DB_PATH` | `/app/ip2region.xdb` | ip2region xdb；缺失则地域为空 |

## 初始化预填

| 变量 | 说明 |
|------|------|
| `SETUP_DEFAULT_MYSQL_HOST` | 向导 MySQL Host 默认值 |
| `SETUP_DEFAULT_REDIS_HOST` | 向导 Redis Host 默认值 |

只影响首次展示，不写死为真实连接。

## 配置合并顺序

1. 非空环境变量  
2. `CONFIG_FILE`  
3. 开发默认值  

Compose 里写空字符串**不会**覆盖已保存配置。

## 配置文件里有什么

初始化成功后大致包含：

- MySQL DSN、Redis 地址  
- 安装状态、管理员认证模式  
- Logto 配置（若启用）  
- 公开访问地址等运行参数  

**包含密码**：文件权限仅运行用户可读写；不要打进镜像或提交 git。

## 安全相关

| 项 | 要求 |
|----|------|
| 密码/DSN | 不进日志、不回传 API |
| 初始化接口 | 仅 ADMIN 监听器；完成后锁定 |
| production | 无无认证管理模式 |
| 管理端 | 建议 Nginx 白名单 |

## 常用组合示例

```yaml
environment:
  APP_ENV: production
  HTTP_ADDR: :8080
  ADMIN_HTTP_ADDR: :8081
  CONFIG_FILE: /data/gravitylink.json
  TZ: Asia/Shanghai
  GEO_DB_PATH: /app/ip2region.xdb
  SETUP_DEFAULT_MYSQL_HOST: mysql.internal
  SETUP_DEFAULT_REDIS_HOST: redis.internal
```

## 不在环境变量里配的

以下在后台/向导配置，不靠环境变量硬塞：

- 具体数据库密码、库名  
- 域名列表  
- 链接与落地页业务数据  
- 对象存储 AK/SK（若启用素材云存储，在设置里填）
