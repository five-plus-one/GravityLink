"""旧版引流宝 dump + 新版 GravityLink dump → 合并后的新版 dump。

保留新版的表结构和现有配置（users、system_configs、installation_states 等），
将旧版业务数据（域名、短链、活码、渠道码、卡密）转换为新版 schema 后追加。

用法：
  python merge_dumps.py \
    --new gravitylink_20260910121627goykm.sql.gz \
    --old r_5plus1_top_ylb_20260909193002o0zpp.sql.gz \
    --output gravitylink-merged.sql
"""
import argparse
import gzip
import json
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


def extract_scheme(url) -> str:
    return 'https' if url and str(url).strip().lower().startswith('https') else 'http'


DOMAIN_TYPE = {'1': 'entry', '2': 'landing', '3': 'transit'}
STATUS_MAP = {'1': 'active', '2': 'disabled'}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--new', required=True)
    parser.add_argument('--old', required=True)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()

    new_sql = read_gz(args.new)
    old_sql = read_gz(args.old)

    # ---------- 域名 ----------
    # 先从新版 dump 读取已有域名，建立 host → id 映射
    host_to_id: dict[str, int] = {}
    host_type: dict[str, str] = {}
    next_id = 50000
    domain_inserts = []

    # 解析新版 dump 中已有的 domains
    for m in re.finditer(r'INSERT INTO `domains`\s+VALUES\s+(.*?);', new_sql, re.DOTALL | re.IGNORECASE):
        for row in parse_values_tuple(m.group(1)):
            # (id, host, type, scheme, remark, status, created_by, created_at, updated_at, deleted_at)
            did, host, dtype = row[0], row[1], row[2]
            if host and dtype:
                host_to_id[host.lower()] = int(did)
                host_type[host.lower()] = dtype
    print(f'新版已有域名: {len(host_to_id)} 个')

    # 旧版域名：只插入新版没有的
    old_domains = parse_insert_rows(old_sql, 'huoma_domain')
    for prefer in ['entry', 'landing', 'transit']:
        for row in old_domains:
            dtype = DOMAIN_TYPE.get(str(row[2]))
            if dtype != prefer: continue
            host = extract_host(row[3])
            if not host: continue
            host_lower = host.lower()
            if host_lower in host_to_id: continue  # 已存在，跳过
            scheme = extract_scheme(row[3])
            next_id += 1
            host_to_id[host_lower] = next_id
            host_type[host_lower] = dtype
            domain_inserts.append(
                f"INSERT INTO `domains` (`id`,`host`,`type`,`scheme`,`remark`,`status`,`created_by`,`created_at`,`updated_at`) "
                f"VALUES ({next_id},{sq(host)},{sq(dtype)},{sq(scheme)},'迁移自旧版','active',0,NOW(),NOW());"
            )
    print(f'新增域名: {len(domain_inserts)} 条（旧版有、新版没有的）')

    def find_entry():
        for h, t in host_type.items():
            if t == 'entry': return host_to_id[h]
        return None

    def find_landing():
        for h, t in host_type.items():
            if t == 'landing': return host_to_id[h]
        return None

    # ---------- 落地页 ----------
    landing_page_inserts = []
    landing_id_map: dict[int, int] = {}
    page_id = 60000
    for host, did in host_to_id.items():
        if host_type[host] != 'landing': continue
        page_id += 1
        landing_id_map[did] = page_id
        content = json.dumps({"headline": "扫码加入群聊", "subtext": "", "footer_text": "长按识别二维码", "theme_color": "#0f766e"}, ensure_ascii=False)
        landing_page_inserts.append(
            f"INSERT INTO `landing_pages` (`id`,`template`,`title`,`content`,`domain_id`,`created_by`,`created_at`,`updated_at`) "
            f"VALUES ({page_id},'liveqr','迁移默认活码页',{sq(content)},'{did}',0,NOW(),NOW());"
        )
    print(f'落地页: {len(landing_page_inserts)} 条')

    # ---------- 短链 ----------
    link_id = 70000
    link_inserts = []
    channel_config_inserts = []
    strategy_inserts = []
    target_inserts = []
    alias_inserts = []

    for row in parse_insert_rows(old_sql, 'huoma_dwz'):
        code, legacy_id = row[3], row[1]
        title = row[2] or ''
        status = STATUS_MAP.get(str(row[7]), 'active')
        target_url = row[8] or ''
        if not code or not target_url: continue
        entry_id = host_to_id.get(extract_host(row[13]) or '') or find_entry()
        if not entry_id: continue
        link_id += 1
        link_inserts.append(
            f"INSERT INTO `links` (`id`,`code`,`type`,`entry_domain_id`,`target_url`,`title`,`access_rule`,`legacy_id`,`status`,`created_by`,`created_at`,`updated_at`) "
            f"VALUES ({link_id},{sq(code)},'short',{sqi(entry_id)},{sq(target_url)},{sq(title)},'none',{sqi(legacy_id)},{sq(status)},0,{sq(row[4])},NOW());"
        )
    print(f'短链: {link_id - 70000} 条')

    # ---------- 渠道码 ----------
    ch_count = 0
    for row in parse_insert_rows(old_sql, 'huoma_channel'):
        code = row[16] if len(row) > 16 else None
        title = row[2] or ''
        status = STATUS_MAP.get(str(row[3]), 'active')
        target_url = row[17] if len(row) > 17 else None
        if not code or not target_url: continue
        entry_id = host_to_id.get(extract_host(row[13]) or '') or find_entry()
        if not entry_id: continue
        link_id += 1
        link_inserts.append(
            f"INSERT INTO `links` (`id`,`code`,`type`,`entry_domain_id`,`target_url`,`title`,`access_rule`,`legacy_id`,`status`,`created_by`,`created_at`,`updated_at`) "
            f"VALUES ({link_id},{sq(code)},'channel',{sqi(entry_id)},{sq(target_url)},{sq(title)},'none',{sqi(row[1])},{sq(status)},0,{sq(row[4])},NOW());"
        )
        channel_config_inserts.append(
            f"INSERT INTO `channel_configs` (`link_id`,`created_at`,`updated_at`) VALUES ({link_id},NOW(),NOW());"
        )
        ch_count += 1
    print(f'渠道码: {ch_count} 条')

    # ---------- 群活码 ----------
    zima_by_qun: dict[str, list] = {}
    for z in parse_insert_rows(old_sql, 'huoma_qun_zima'):
        zima_by_qun.setdefault(str(z[1]), []).append(z)

    qun_count = 0
    for row in parse_insert_rows(old_sql, 'huoma_qun'):
        qun_id = str(row[1])
        title = row[2] or f'群码{qun_id}'
        status = STATUS_MAP.get(str(row[3]), 'active')
        code = row[16] if len(row) > 16 else None
        if not code: continue
        targets = zima_by_qun.get(qun_id, [])
        if not targets: continue
        entry_id = host_to_id.get(extract_host(row[9]) or '') or find_entry()
        landing_id = host_to_id.get(extract_host(row[10]) or '') or find_landing()
        if not entry_id or not landing_id: continue
        page = landing_id_map.get(landing_id)
        if not page: continue
        link_id += 1
        link_inserts.append(
            f"INSERT INTO `links` (`id`,`code`,`type`,`entry_domain_id`,`landing_domain_id`,`landing_page_id`,`title`,`access_rule`,`legacy_id`,`status`,`created_by`,`created_at`,`updated_at`) "
            f"VALUES ({link_id},{sq(code)},'liveqr',{sqi(entry_id)},{sqi(landing_id)},{sqi(page)},{sq(title)},'none',{sqi(row[1])},{sq(status)},0,{sq(row[4])},NOW());"
        )
        strategy_id = link_id  # 用 link_id 做 strategy_id 简化
        strategy_inserts.append(
            f"INSERT INTO `routing_strategies` (`id`,`link_id`,`mode`,`created_at`,`updated_at`) "
            f"VALUES ({strategy_id},{link_id},'round_robin',NOW(),NOW());"
        )
        for z in targets:
            zm_url = z[5]
            zm_leader = z[6] or ''
            zm_limit = sqi(z[3]) if z[3] else '200'
            zm_status = 'active' if str(z[8]) == '1' else 'disabled'
            zm_pv = sqi(z[4]) if z[4] else '0'
            if not zm_url: continue
            target_inserts.append(
                f"INSERT INTO `routing_targets` (`strategy_id`,`label`,`target_url`,`weight`,`scan_limit`,`scan_count`,`owner`,`status`,`created_at`,`updated_at`) "
                f"VALUES ({strategy_id},{sq(zm_leader or '迁移子码')},{sq(zm_url)},1,{zm_limit},{zm_pv},{sq(zm_leader)},{sq(zm_status)},NOW(),NOW());"
            )
        qun_count += 1
    print(f'群活码: {qun_count} 组')

    # ---------- 卡密 ----------
    kami_proj_inserts = []
    kami_item_inserts = []
    kami_id = 90000
    for row in parse_insert_rows(old_sql, 'ylb_kami'):
        kami_id += 1
        kami_proj_inserts.append(
            f"INSERT INTO `kami_projects` (`id`,`title`,`type`,`status`,`created_by`,`created_at`,`updated_at`) "
            f"VALUES ({kami_id},{sq(row[2] or '卡密项目')},{sq(row[3] or '卡密')},'active',0,NOW(),NOW());"
        )
    km_item_id = 900000
    for row in parse_insert_rows(old_sql, 'ylb_kmlist'):
        old_kami_id = row[1]
        # 找到对应的新 project id（按顺序）
        proj_idx = None
        old_kamis = parse_insert_rows(old_sql, 'ylb_kami')
        for i, k in enumerate(old_kamis):
            if str(k[1]) == str(old_kami_id):
                proj_idx = i
                break
        if proj_idx is None: continue
        new_proj_id = 90001 + proj_idx
        km_item_id += 1
        km_status = 'issued' if str(row[7]) == '2' else 'unissued'
        kami_item_inserts.append(
            f"INSERT INTO `kami_items` (`id`,`project_id`,`content`,`note`,`expires_text`,`status`,`created_at`) "
            f"VALUES ({km_item_id},{new_proj_id},{sq(row[3])},{sq(row[8])},{sq(row[4])},{sq(km_status)},NOW());"
        )
    print(f'卡密项目: {len(kami_proj_inserts)}, 卡密条目: {len(kami_item_inserts)}')

    # ---------- 分享卡片 ----------
    share_card_inserts = []
    sc_id = 80000
    for row in parse_insert_rows(old_sql, 'huoma_shareCard'):
        # (id, shareCard_id, shareCard_title, shareCard_desc, shareCard_img,
        #  shareCard_ldym, shareCard_url, shareCard_pv, shareCard_create_time,
        #  shareCard_status, shareCard_model, shareCard_create_user)
        sc_id += 1
        legacy_id = row[1]  # shareCard_id
        title = row[2] or '分享卡片'
        desc = row[3] or ''
        img = row[4] or ''
        target_url = row[6] or ''
        pv = sqi(row[7]) if row[7] else '0'
        status = 'active' if str(row[9]) == '1' else 'disabled'
        created = row[8] or 'NOW()'
        # 找落地域名对应的 domain_id
        ldym_host = extract_host(row[5])
        domain_id = host_to_id.get(ldym_host) if ldym_host else None
        share_card_inserts.append(
            f"INSERT INTO `share_cards` (`id`,`domain_id`,`title`,`description`,`image_url`,`target_url`,`status`,`visits`,`legacy_id`,`created_at`,`updated_at`) "
            f"VALUES ({sc_id},{sqi(domain_id)},{sq(title)},{sq(desc)},{sq(img)},{sq(target_url)},{sq(status)},{pv},{sqi(legacy_id)},{sq(created)},NOW());"
        )
    print(f'分享卡片: {len(share_card_inserts)} 条')

    # ---------- 组装输出 ----------
    all_inserts = (
        domain_inserts + landing_page_inserts + link_inserts +
        channel_config_inserts + strategy_inserts + target_inserts +
        kami_proj_inserts + kami_item_inserts + share_card_inserts
    )
    print(f'\n合计新增 INSERT: {len(all_inserts)} 条')

    # 去掉新版 dump 中的 LOCK TABLES / UNLOCK TABLES（避免和追加的 INSERT 冲突）
    cleaned_lines = []
    for line in new_sql.split('\n'):
        stripped = line.strip().upper()
        if stripped.startswith('LOCK TABLES') or stripped == 'UNLOCK TABLES;':
            continue
        cleaned_lines.append(line)
    new_sql = '\n'.join(cleaned_lines)

    fk_off = "SET FOREIGN_KEY_CHECKS=0;\n"
    fk_on = "SET FOREIGN_KEY_CHECKS=1;\n"

    # 在新版 dump 末尾（最后一个分号后）追加旧版数据
    merged = new_sql.rstrip()
    if not merged.endswith(';'):
        merged += ';'
    merged += '\n\n-- ===== 旧版数据迁移 =====\n'
    merged += fk_off
    merged += '\n'.join(all_inserts) + '\n'
    merged += fk_on
    merged += '\n'

    out = Path(args.output)
    out.write_text(merged, encoding='utf-8')
    print(f'\n已写入: {out} ({out.stat().st_size} 字节)')


if __name__ == '__main__':
    main()
