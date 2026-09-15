# 功能规格：品牌展示配置

## 目标

管理端与登录页的品牌名称、账号平台文案、Logo 可配置，无需改代码。配置存 `system_configs`，重启后自动保留（无需新表）。

## 配置键

| Key | 说明 | 默认 |
|-----|------|------|
| `brand.name` | 平台名称（登录页左侧、页脚、浏览器标题） | GravityLink |
| `brand.logo_url` | 平台 Logo（建议 `/uploads/...`） | 空 = 使用内置 SVG |
| `brand.auth_label` | 账号平台名称（登录页文案「使用 xx 登录」） | Logto |
| `brand.auth_logo_url` | 账号平台 Logo（登录按钮图标，可选） | 空 |

## 接口

- `GET /api/v1/auth/config`（登录页可访问）响应增加 `brand_name` / `brand_logo` / `auth_label` / `auth_logo`
- `PUT /api/admin/configs` 写入上述键；Logo 通过 `POST /api/admin/materials` 上传后填路径

## 管理端

系统设置 →「品牌」Tab：平台名称、账号平台名称、两个 Logo 上传/粘贴地址，保存后立即刷新登录页文案。

## 数据库迁移

- 沿用现有 `system_configs` 表（AutoMigrate 已覆盖），旧库升级无需 ALTER
- 未配置的键读取时使用默认值，行为与升级前一致
