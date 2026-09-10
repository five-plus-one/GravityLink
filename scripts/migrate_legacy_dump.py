"""旧版引流宝 SQL dump → GravityLink 新库迁移。

用法：
  python scripts/migrate_legacy_dump.py --dump workspace/legacy_dump.sql \
    --mysql-host 127.0.0.1 --mysql-port 23306 --mysql-user gravitylink \
    --mysql-password <pwd> --mysql-db gravitylink

支持 --dry-run 只解析不写入。
"""
import argparse
import json
import re
import sys
from pathlib import Path

try:
    import pymysql
except ImportError:
    print("需要 pymysql：pip install pymysql")
    sys.exit(1)

# 旧版域名类型 → 新版
DOMAIN_TYPE_MAP = {
    '1': 'entry',     # 入口
    '2': 'landing',   # 落地
    '3': 'transit',   # 短链/中转
    # 4=备用, 5=对象存储, 6=? → 跳过
}

# 旧版链接状态 → 新版
STATUS_MAP = {'1': 'active', '2': 'disabled'}


def parse_insert_rows(content: str, table: str) -> list[list]:
    """解析 INSERT INTO `table` VALUES (...),(...); 返回行列表。"""
    bt = chr(96)
    # 找所有 INSERT 语句
    pat = rf'INSERT INTO {bt}{re.escape(table)}{bt}\s+VALUES\s+(.*?);'
    rows = []
    for m in re.finditer(pat, content, re.IGNORECASE | re.DOTALL):
        values_str = m.group(1)
        rows.extend(parse_values_tuple(values_str))
    return rows


def parse_values_tuple(s: str) -> list[list]:
    """解析 VALUES 后面的 (...),(...) 元组列表。"""
    rows = []
    depth = 0
    current = []
    in_string = False
    escape = False
    i = 0
    while i < len(s):
        c = s[i]
        if escape:
            current.append(c)
            escape = False
        elif c == '\\':
            current.append(c)
            escape = True
        elif c == "'" and not escape:
            in_string = not in_string
            current.append(c)
        elif not in_string:
            if c == '(':
                depth += 1
                if depth == 1:
                    current = []
            elif c == ')':
                depth -= 1
                if depth == 0:
                    rows.append(parse_row_fields(''.join(current)))
            elif depth >= 1:
                current.append(c)
        else:
            current.append(c)
        i += 1
    return rows


def parse_row_fields(s: str) -> list:
    """解析一行内的字段值，处理引号和 NULL。"""
    fields = []
    current = []
    in_string = False
    escape = False
    for c in s:
        if escape:
            current.append(c)
            escape = False
        elif c == '\\':
            escape = True
            current.append(c)
        elif c == "'" :
            in_string = not in_string
        elif c == ',' and not in_string:
            fields.append(clean_field(''.join(current)))
            current = []
        else:
            current.append(c)
    fields.append(clean_field(''.join(current)))
    return fields


def clean_field(v: str) -> str | None:
    v = v.strip()
    if v.upper() == 'NULL':
        return None
    # 去掉外层引号（如果有）
    if len(v) >= 2 and v[0] == "'" and v[-1] == "'":
        v = v[1:-1]
    return v


def extract_host(url: str | None) -> str | None:
    """从 URL 提取 host（不含 scheme 和 path）。"""
    if not url:
        return None
    url = url.strip()
    url = re.sub(r'^https?://', '', url)
    url = url.split('/')[0]
    url = url.split('?')[0]
    return url if url else None


def extract_scheme(url: str | None) -> str:
    if url and url.strip().lower().startswith('https'):
        return 'https'
    return 'http'


def migrate(dump_path: str, mysql_cfg: dict, dry_run: bool = False):
    content = Path(dump_path).read_text(encoding='utf-8', errors='replace')
    print(f"已读取 dump: {len(content)} 字符")

    conn = None
    if not dry_run:
        conn = pymysql.connect(
            host=mysql_cfg['host'], port=mysql_cfg['port'],
            user=mysql_cfg['user'], password=mysql_cfg['password'],
            database=mysql_cfg['database'], charset='utf8mb4',
            autocommit=False,
        )
        print(f"已连接 MySQL: {mysql_cfg['host']}:{mysql_cfg['port']}/{mysql_cfg['database']}")

    cur = conn.cursor() if conn else None

    # ---------- 1. 域名 ----------
    domain_rows = parse_insert_rows(content, 'huoma_domain')
    print(f"\n域名: {len(domain_rows)} 条")
    host_to_id: dict[str, int] = {}  # host → new domain ID
    host_type: dict[str, str] = {}   # host → 实际使用的类型
    fake_id = 1000
    # 优先插入 entry 类型，再 landing，再 transit
    for prefer_type in ['entry', 'landing', 'transit']:
        for row in domain_rows:
            old_id, _, dtype, domain_url, _ug = row[0], row[1], row[2], row[3], row[4]
            new_type = DOMAIN_TYPE_MAP.get(str(dtype))
            if new_type != prefer_type:
                continue
            host = extract_host(domain_url)
            scheme = extract_scheme(domain_url)
            if not host or host in host_to_id:
                continue
            print(f"  [{new_type:7s}] {scheme}://{host}")
            if dry_run:
                fake_id += 1
                host_to_id[host] = fake_id
                host_type[host] = new_type
                continue
            if cur:
                cur.execute("SELECT id, type FROM domains WHERE host=%s AND deleted_at IS NULL", (host,))
                existing = cur.fetchone()
                if existing:
                    host_to_id[host] = existing[0]
                    host_type[host] = existing[1]
                    print(f"    已存在 ID={existing[0]} type={existing[1]}")
                    continue
                cur.execute(
                    "INSERT INTO domains (host, type, scheme, remark, status, created_by, created_at, updated_at) VALUES (%s,%s,%s,%s,'active',0,NOW(),NOW())",
                    (host, new_type, scheme, f"迁移自旧版 ID={old_id}"),
                )
                host_to_id[host] = cur.lastrowid
                host_type[host] = new_type
                print(f"    新建 ID={cur.lastrowid}")

    # 构建 host:type → id 映射（兼容后续查询）
    host_type_to_id: dict[str, int] = {}
    for host, did in host_to_id.items():
        host_type_to_id[f"{host}:{host_type[host]}"] = did
        # 也映射为所有类型，方便 fallback
        for t in ['entry', 'landing', 'transit']:
            host_type_to_id.setdefault(f"{host}:{t}", did)

    # ---------- 2. 落地页（群活码页模板） ----------
    landing_pages: dict[int, int] = {}  # domain_id → page_id
    landing_hosts = [h for h, t in host_type.items() if t == 'landing']
    fake_page = 5000
    for host in landing_hosts:
        dom_id = host_to_id[host]
        if dry_run:
            fake_page += 1
            landing_pages[dom_id] = fake_page
            continue
        if cur:
            cur.execute("SELECT id FROM landing_pages WHERE domain_id=%s AND template='liveqr' AND deleted_at IS NULL LIMIT 1", (dom_id,))
            existing = cur.fetchone()
            if existing:
                landing_pages[dom_id] = existing[0]
                continue
            content_json = json.dumps({"headline": "扫码加入群聊", "subtext": "", "footer_text": "长按识别二维码", "theme_color": "#0f766e"}, ensure_ascii=False)
            cur.execute(
                "INSERT INTO landing_pages (template, title, content, domain_id, created_by, created_at, updated_at) VALUES ('liveqr', %s, %s, %s, 0, NOW(), NOW())",
                ("迁移默认活码页", content_json, dom_id),
            )
            landing_pages[dom_id] = cur.lastrowid
            print(f"  落地页 {host} → page={cur.lastrowid}")

    # ---------- 3. 短链接 (huoma_dwz) ----------
    dwz_rows = parse_insert_rows(content, 'huoma_dwz')
    print(f"\n短链接: {len(dwz_rows)} 条")
    migrated_links = 0
    for row in dwz_rows:
        # (id, dwz_id, dwz_title, dwz_key, dwz_creat_time, dwz_pv, dwz_today_pv, dwz_status,
        #  dwz_url, dwz_android_url, dwz_ios_url, dwz_windows_url, dwz_type,
        #  dwz_rkym, dwz_zzym, dwz_dlym, ?, create_user)
        code = row[3]
        title = row[2] or ''
        status = STATUS_MAP.get(str(row[7]), 'active')
        target_url = row[8] or ''
        entry_url = row[13] if len(row) > 13 else None
        if not code or not target_url:
            continue
        entry_host = extract_host(entry_url)
        entry_id = host_to_id.get(entry_host) if entry_host else None
        if not entry_id:
            # 找任意 entry 类型域名
            for h, t in host_type.items():
                if t == 'entry':
                    entry_id = host_to_id[h]
                    break
        if not entry_id:
            print(f"  跳过 {code}: 无入口域名")
            continue
        access_rule = 'none'
        print(f"  [{status:8s}] {code} → {target_url[:60]}")
        if dry_run:
            migrated_links += 1
            continue
        if cur:
            cur.execute("SELECT id FROM links WHERE code=%s AND deleted_at IS NULL", (code,))
            if cur.fetchone():
                print(f"    已存在，跳过")
                continue
            legacy_id = row[1] if row[1] else None  # dwz_id
            cur.execute(
                "INSERT INTO links (code, type, entry_domain_id, target_url, title, access_rule, legacy_id, status, created_by, created_at, updated_at) VALUES (%s,'short',%s,%s,%s,%s,%s,%s,0,%s,NOW())",
                (code, entry_id, target_url, title, access_rule, legacy_id, status, row[4]),
            )
            migrated_links += 1
    print(f"  新建短链: {migrated_links}")

    # ---------- 4. 群活码 (huoma_qun + huoma_qun_zima) ----------
    qun_rows = parse_insert_rows(content, 'huoma_qun')
    zima_rows = parse_insert_rows(content, 'huoma_qun_zima')
    print(f"\n群活码: {len(qun_rows)} 组, 子码: {len(zima_rows)} 条")

    # 按 qun_id 分组子码
    zima_by_qun: dict[str, list] = {}
    for z in zima_rows:
        qun_id = str(z[1])
        zima_by_qun.setdefault(qun_id, []).append(z)

    migrated_qun = 0
    for row in qun_rows:
        # (id, qun_id, qun_title, qun_status, qun_creat_time, qun_pv, qun_today_pv,
        #  qun_qc, qun_notify, qun_rkym, qun_ldym, qun_dlym, qun_kf, qun_kf_status,
        #  qun_safety, qun_beizhu, qun_key, create_user)
        qun_id = str(row[1])
        title = row[2] or f'群码{qun_id}'
        status = STATUS_MAP.get(str(row[3]), 'active')
        code = row[16] if len(row) > 16 else None
        if not code:
            print(f"  跳过 qun_id={qun_id}: 无短码")
            continue

        targets = zima_by_qun.get(qun_id, [])
        if not targets:
            print(f"  跳过 {code}: 无子码")
            continue

        # 找入口和落地域名
        entry_host = extract_host(row[9])  # qun_rkym
        landing_host = extract_host(row[10])  # qun_ldym
        entry_id = host_to_id.get(entry_host) if entry_host else None
        landing_id = host_to_id.get(landing_host) if landing_host else None
        if not entry_id:
            for h, t in host_type.items():
                if t == 'entry':
                    entry_id = host_to_id[h]
                    break
        if not landing_id:
            for h, t in host_type.items():
                if t == 'landing':
                    landing_id = host_to_id[h]
                    break
        if not entry_id or not landing_id:
            print(f"  跳过 {code}: 缺少域名")
            continue

        page_id = landing_pages.get(landing_id)
        if not page_id:
            print(f"  跳过 {code}: 无落地页")
            continue

        print(f"  [{status:8s}] {code} ({title}) → {len(targets)} 个子码")
        if dry_run:
            migrated_qun += 1
            continue

        cur.execute("SELECT id FROM links WHERE code=%s AND deleted_at IS NULL", (code,))
        if cur.fetchone():
            print(f"    已存在，跳过")
            continue

        legacy_id = row[1] if row[1] else None  # qun_id
        cur.execute(
            "INSERT INTO links (code, type, entry_domain_id, landing_domain_id, landing_page_id, title, access_rule, legacy_id, status, created_by, created_at, updated_at) VALUES (%s,'liveqr',%s,%s,%s,%s,'none',%s,%s,0,%s,NOW())",
            (code, entry_id, landing_id, page_id, title, legacy_id, status, row[4]),
        )
        link_id = cur.lastrowid

        # 创建策略
        cur.execute(
            "INSERT INTO routing_strategies (link_id, mode, created_at, updated_at) VALUES (%s,'round_robin',NOW(),NOW())",
            (link_id,),
        )
        strategy_id = cur.lastrowid

        # 插入子码目标
        for z in targets:
            # (id, qun_id, zm_id, zm_yz, zm_pv, zm_qrcode, zm_leader, zm_update_time, zm_status)
            zm_qrcode = z[5]
            zm_leader = z[6] or ''
            zm_limit = int(z[3]) if z[3] else 200
            zm_status = 'active' if str(z[8]) == '1' else 'disabled'
            zm_pv = int(z[4]) if z[4] else 0
            if not zm_qrcode:
                continue
            cur.execute(
                "INSERT INTO routing_targets (strategy_id, label, target_url, weight, scan_limit, scan_count, owner, status, created_at, updated_at) VALUES (%s,%s,%s,1,%s,%s,%s,%s,NOW(),NOW())",
                (strategy_id, zm_leader or '迁移子码', zm_qrcode, zm_limit, zm_pv, zm_leader, zm_status),
            )
        migrated_qun += 1
    print(f"  新建群活码: {migrated_qun}")

    # ---------- 5. 渠道码 (huoma_channel) ----------
    ch_rows = parse_insert_rows(content, 'huoma_channel')
    print(f"\n渠道码: {len(ch_rows)} 条")
    migrated_ch = 0
    for row in ch_rows:
        # (id, channel_id, channel_title, channel_status, channel_creat_time, channel_pv,
        #  Android_Total, iOS_Total, Windows_Total, Linux_Total, MacOS_Total,
        #  channel_DataTotal, channel_today_pv, channel_rkym, channel_ldym, channel_dlym,
        #  channel_key, channel_url, channel_creat_user)
        code = row[16] if len(row) > 16 else None
        title = row[2] or ''
        status = STATUS_MAP.get(str(row[3]), 'active')
        target_url = row[17] if len(row) > 17 else None
        entry_url = row[13] if len(row) > 13 else None
        if not code or not target_url:
            continue
        entry_host = extract_host(entry_url)
        entry_id = host_to_id.get(entry_host) if entry_host else None
        if not entry_id:
            for h, t in host_type.items():
                if t == 'entry':
                    entry_id = host_to_id[h]
                    break
        if not entry_id:
            print(f"  跳过 {code}: 无入口域名")
            continue
        print(f"  [{status:8s}] {code} → {target_url[:60]}")
        if dry_run:
            migrated_ch += 1
            continue
        if cur:
            cur.execute("SELECT id FROM links WHERE code=%s AND deleted_at IS NULL", (code,))
            if cur.fetchone():
                print(f"    已存在，跳过")
                continue
            legacy_id = row[1] if row[1] else None  # channel_id
            cur.execute(
                "INSERT INTO links (code, type, entry_domain_id, target_url, title, access_rule, legacy_id, status, created_by, created_at, updated_at) VALUES (%s,'channel',%s,%s,%s,'none',%s,%s,0,%s,NOW())",
                (code, entry_id, target_url, title, legacy_id, status, row[4]),
            )
            link_id = cur.lastrowid
            cur.execute(
                "INSERT INTO channel_configs (link_id, created_at, updated_at) VALUES (%s,NOW(),NOW())",
                (link_id,),
            )
            migrated_ch += 1
    print(f"  新建渠道码: {migrated_ch}")

    # ---------- 6. 提交 ----------
    if not dry_run and conn:
        conn.commit()
        print(f"\n迁移完成，已提交。")
        cur.execute("SELECT COUNT(*) FROM domains WHERE deleted_at IS NULL")
        print(f"  域名总数: {cur.fetchone()[0]}")
        cur.execute("SELECT COUNT(*) FROM links WHERE deleted_at IS NULL")
        print(f"  链接总数: {cur.fetchone()[0]}")
        cur.execute("SELECT COUNT(*) FROM routing_targets")
        print(f"  目标总数: {cur.fetchone()[0]}")
        conn.close()
    else:
        print(f"\n[dry-run] 解析完成，未写入数据库。")


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='迁移旧版引流宝 SQL dump')
    parser.add_argument('--dump', required=True, help='SQL dump 文件路径')
    parser.add_argument('--mysql-host', default='127.0.0.1')
    parser.add_argument('--mysql-port', type=int, default=23306)
    parser.add_argument('--mysql-user', default='gravitylink')
    parser.add_argument('--mysql-password', required=True)
    parser.add_argument('--mysql-db', default='gravitylink')
    parser.add_argument('--dry-run', action='store_true')
    args = parser.parse_args()

    migrate(args.dump, {
        'host': args.mysql_host, 'port': args.mysql_port,
        'user': args.mysql_user, 'password': args.mysql_password,
        'database': args.mysql_db,
    }, dry_run=args.dry_run)
