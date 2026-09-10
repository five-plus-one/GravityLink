"""在新版 GravityLink dump 基础上补入旧版分享卡片数据。

新版 dump 已包含域名、链接、活码、渠道码、卡密等迁移数据。
本脚本只补入旧版 huoma_shareCard → 新版 share_cards。

用法：
  python merge_sharecards.py \
    --new gravitylink_xxx.sql.gz \
    --old r_5plus1_top_ylb_xxx.sql.gz \
    --output gravitylink-merged.sql
"""
import argparse
import gzip
import re
from pathlib import Path


def read_gz(path: str) -> str:
    with gzip.open(path, 'rt', encoding='utf-8', errors='replace') as f:
        return f.read()


def parse_insert_rows(content: str, table: str) -> list[list]:
    bt = chr(96)
    pat = rf'INSERT INTO {bt}{re.escape(table)}{bt}\s+VALUES\s+(.*?);'
    rows = []
    for m in re.finditer(pat, content, re.IGNORECASE | re.DOTALL):
        rows.extend(parse_values_tuple(m.group(1)))
    return rows


def parse_values_tuple(s: str) -> list[list]:
    rows, depth, current, in_str, esc = [], 0, [], False, False
    for c in s:
        if esc:
            current.append(c); esc = False
        elif c == '\\':
            current.append(c); esc = True
        elif c == "'":
            in_str = not in_str; current.append(c)
        elif not in_str:
            if c == '(':
                depth += 1
                if depth == 1: current = []
            elif c == ')':
                depth -= 1
                if depth == 0: rows.append(parse_fields(''.join(current)))
            elif depth >= 1:
                current.append(c)
        else:
            current.append(c)
    return rows


def parse_fields(s: str) -> list:
    fields, current, in_str, esc = [], [], False, False
    for c in s:
        if esc: current.append(c); esc = False
        elif c == '\\': esc = True; current.append(c)
        elif c == "'": in_str = not in_str
        elif c == ',' and not in_str:
            fields.append(clean(''.join(current))); current = []
        else: current.append(c)
    fields.append(clean(''.join(current)))
    return fields


def clean(v: str):
    v = v.strip()
    if v.upper() == 'NULL': return None
    if len(v) >= 2 and v[0] == "'" and v[-1] == "'":
        v = v[1:-1].replace("\\'", "'").replace('\\"', '"').replace('\\\\', '\\')
    return v


def sq(v) -> str:
    if v is None: return 'NULL'
    s = str(v).replace('\\', '\\\\').replace("'", "\\'")
    return f"'{s}'"


def sqi(v) -> str:
    if v is None: return 'NULL'
    try: return str(int(v))
    except: return 'NULL'


def extract_host(url) -> str | None:
    if not url: return None
    url = re.sub(r'^https?://', '', str(url).strip())
    return url.split('/')[0].split('?')[0] or None


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--new', required=True)
    parser.add_argument('--old', required=True)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()

    new_sql = read_gz(args.new)
    old_sql = read_gz(args.old)

    # 从新版 dump 读取已有域名 host → id
    host_to_id: dict[str, int] = {}
    for m in re.finditer(r'INSERT INTO `domains`\s+VALUES\s+(.*?);', new_sql, re.DOTALL | re.IGNORECASE):
        for row in parse_values_tuple(m.group(1)):
            did, host = row[0], row[1]
            if host:
                host_to_id[host.lower()] = int(did)
    print(f'新版已有域名: {len(host_to_id)} 个')

    # 从新版 dump 读取已有的 share_cards legacy_id，避免重复
    existing_legacy: set[int] = set()
    for m in re.finditer(r'INSERT INTO `share_cards`\s+VALUES\s+(.*?);', new_sql, re.DOTALL | re.IGNORECASE):
        for row in parse_values_tuple(m.group(1)):
            # 找 legacy_id 字段（如果有）
            pass
    # 简单方式：检查新版 dump 中 share_cards 的 INSERT 数量
    sc_count = len(re.findall(r'INSERT INTO `share_cards`', new_sql, re.IGNORECASE))
    print(f'新版已有分享卡片: {sc_count} 条')

    # ---------- 分享卡片 ----------
    share_card_inserts = []
    sc_id = 80000
    for row in parse_insert_rows(old_sql, 'huoma_shareCard'):
        # (id, shareCard_id, shareCard_title, shareCard_desc, shareCard_img,
        #  shareCard_ldym, shareCard_url, shareCard_pv, shareCard_create_time,
        #  shareCard_status, shareCard_model, shareCard_create_user)
        sc_id += 1
        legacy_id = row[1]
        title = row[2] or '分享卡片'
        desc = row[3] or ''
        img = row[4] or ''
        target_url = row[6] or ''
        pv = sqi(row[7]) if row[7] else '0'
        status = 'active' if str(row[9]) == '1' else 'disabled'
        created = row[8] or 'NOW()'
        ldym_host = extract_host(row[5])
        domain_id = host_to_id.get(ldym_host.lower()) if ldym_host else None
        share_card_inserts.append(
            f"INSERT INTO `share_cards` (`id`,`domain_id`,`title`,`description`,`image_url`,`target_url`,`status`,`visits`,`legacy_id`,`created_at`,`updated_at`) "
            f"VALUES ({sc_id},{sqi(domain_id)},{sq(title)},{sq(desc)},{sq(img)},{sq(target_url)},{sq(status)},{pv},{sqi(legacy_id)},{sq(created)},NOW());"
        )
    print(f'分享卡片: {len(share_card_inserts)} 条')

    # ---------- 组装输出 ----------
    # 去掉 LOCK TABLES
    cleaned = []
    for line in new_sql.split('\n'):
        s = line.strip().upper()
        if s.startswith('LOCK TABLES') or s == 'UNLOCK TABLES;':
            continue
        cleaned.append(line)
    merged = '\n'.join(cleaned).rstrip()
    if not merged.endswith(';'):
        merged += ';'

    merged += '\n\n-- ===== 旧版分享卡片迁移 =====\n'
    merged += 'SET FOREIGN_KEY_CHECKS=0;\n'
    # 确保 share_cards 有 legacy_id 列（应用 AutoMigrate 也会加，这里提前加避免导入失败）
    merged += 'SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=\'share_cards\' AND COLUMN_NAME=\'legacy_id\');\n'
    merged += 'SET @sql = IF(@col_exists=0, \'ALTER TABLE share_cards ADD COLUMN legacy_id BIGINT UNSIGNED NULL, ADD INDEX idx_sharecard_legacy (legacy_id)\', \'SELECT 1\');\n'
    merged += 'PREPARE stmt FROM @sql;\n'
    merged += 'EXECUTE stmt;\n'
    merged += 'DEALLOCATE PREPARE stmt;\n'
    merged += '\n'.join(share_card_inserts) + '\n'
    merged += 'SET FOREIGN_KEY_CHECKS=1;\n'

    out = Path(args.output)
    out.write_text(merged, encoding='utf-8')
    print(f'\n已写入: {out} ({out.stat().st_size} 字节)')


if __name__ == '__main__':
    main()
