# 旧版数据迁移

GravityLink 提供 `migrate-legacy` CLI，用于把旧系统导出的标准 CSV 导入新库。

## 构建

```bash
cd backend
go build ./cmd/migrate-legacy
```

## 命令

```bash
MYSQL_DSN='gravitylink:gravitylink@tcp(127.0.0.1:3306)/gravitylink?charset=utf8mb4&parseTime=True&loc=Local' \
  ./migrate-legacy -domains domains.csv -links links.csv
```

先验证不写库：

```bash
./migrate-legacy -domains domains.csv -links links.csv -dry-run
```

## domains.csv

必填列：`host`

可选列：`type`、`scheme`、`remark`、`status`、`created_by`

```csv
host,type,scheme,remark,status,created_by
go.example.com,entry,https,主入口,active,1
page.example.com,landing,https,落地域名,active,1
```

## links.csv

必填列：`code`

可选列：`type`、`entry_domain_id`、`transit_domain_id`、`landing_domain_id`、`target_url`、`landing_page_id`、`title`、`expire_at`、`status`、`created_by`

```csv
code,type,entry_domain_id,target_url,title,status,created_by
abc,short,1,https://example.com,示例短链,active,1
```

## 注意事项

- 迁移不会覆盖已存在的短码或域名，遇到唯一索引冲突会失败。
- 建议先迁移域名，再迁移链接。
- 群活码策略、渠道码 UTM 和落地页内容建议在新系统内补齐，或单独编写项目定制转换脚本。
