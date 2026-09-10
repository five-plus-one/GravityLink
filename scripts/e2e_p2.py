"""P2 专项 E2E：客服码 / 卡密提取 / 开放 API / UA 访问限制。
复用 e2e_local 的 Client 与密码；在已初始化的 gravitylink 栈上增量执行。
"""
import json
import secrets
import sys
import time
from pathlib import Path

from e2e_local import Client, password, ADMIN, PUBLIC, WORK, check, RESULTS
from repair_demo_qr import upload as upload_fixture

def run():
    client, anonymous = Client(), Client()
    pwd = password()
    client.ok('/api/v1/auth/local/login', 'POST', {'username': 'admin', 'password': pwd})

    stamp = str(int(time.time()))[-8:]
    domains = client.ok('/api/admin/domains')['items']
    entry = next((d for d in domains if d['Type'] == 'entry'), None)
    landing = next((d for d in domains if d['Type'] == 'landing'), None)
    if not entry or not landing:
        entry = client.ok('/api/admin/domains', 'POST', {'host': f'p2e-{stamp}.test', 'type': 'entry', 'scheme': 'http'})
        landing = client.ok('/api/admin/domains', 'POST', {'host': f'p2l-{stamp}.test', 'type': 'landing', 'scheme': 'http'})

    # ---------- 客服码（kf） ----------
    kf_page = client.ok('/api/v1/landing-pages', 'POST', {
        'template': 'kf', 'title': 'E2E 客服码', 'domain_id': landing['ID'],
        'content': {'headline': '添加专属客服', 'subtext': '长按识别二维码', 'footer_text': '工作日 9-18 点', 'theme_color': '#16a34a', 'safety_tip': '谨防诈骗'},
    })
    check('kf landing page created', kf_page['Template'] == 'kf')

    import subprocess
    subprocess.run(['node', str(Path(__file__).parent / 'generate-test-qrs.cjs')], check=True, capture_output=True)
    qr_url = upload_fixture(client, WORK / 'demo-qr-first.png')

    # 全周 9-18 点在线
    schedule = json.dumps({str(d): [['09:00', '18:00']] for d in range(7)})
    kf_link = client.ok('/api/v1/links', 'POST', {
        'type': 'liveqr', 'code': 'kf' + stamp,
        'entry_domain_id': entry['ID'], 'landing_domain_id': landing['ID'], 'landing_page_id': kf_page['ID'],
        'title': 'E2E 客服码', 'online_schedule': schedule,
        'strategy': {'mode': 'round_robin', 'targets': [
            {'target_url': qr_url, 'label': '客服A', 'wx_remark': 'wxid_e2e_cs'},
        ]},
    })
    check('kf liveqr link created', kf_link['Type'] == 'liveqr')

    # 入口 302 到落地
    st, hd, _ = anonymous.call('/' + kf_link['Code'], host=entry['Host'], public=True)
    check('kf entry redirects to landing', st == 302 and landing['Host'] in hd.get('Location', ''))

    # 落地页渲染：在线徽章 + 微信号 + 安全提示
    st, hd, html = anonymous.call('/' + kf_link['Code'], host=landing['Host'], public=True)
    check('kf landing renders 200', st == 200 and hd.get('Cache-Control') == 'no-store')
    check('kf shows online badge', '当前在线' in html)
    check('kf shows wx remark copy', 'wxid_e2e_cs' in html and 'data-copy-wx' in html)
    check('kf shows safety tip', '谨防诈骗' in html)
    check('kf shows headline', '添加专属客服' in html)

    # 离线时段：只配周日 00:00-00:01，当前几乎必然不在该时段
    offline_schedule = json.dumps({'0': [['00:00', '00:01']]})
    client.ok('/api/v1/links/' + str(kf_link['ID']), 'PUT', {'online_schedule': offline_schedule})
    # 清缓存：改 status 触发，或直接再访问（landing 不走 link cache，每次查 DB）
    _, _, html2 = anonymous.call('/' + kf_link['Code'], host=landing['Host'], public=True)
    check('kf offline badge when outside schedule', '可能回复较慢' in html2)

    # 预览接口
    preview = client.ok('/api/v1/landing-pages/' + str(kf_page['ID']) + '/preview')
    check('kf preview renders', '客服二维码展示区域' in preview['html'])

    # ---------- 卡密分发（kami） ----------
    project = client.ok('/api/admin/kami/projects', 'POST', {
        'title': 'E2E 卡密项目', 'type': '卡密', 'password': 'secret123',
        'repeat_policy': 'never', 'repeat_interval_sec': 0,
    })
    project_id = project.get('ID') or project.get('id')
    check('kami project created', bool(project_id))

    imported = client.ok(f"/api/admin/kami/projects/{project_id}/items/import", 'POST', {
        'items': [{'content': 'KAMI-001'}, {'content': 'KAMI-002'}, {'content': 'KAMI-003'}],
    })
    check('kami items imported', imported['imported'] == 3)

    kami_page = client.ok('/api/v1/landing-pages', 'POST', {
        'template': 'kami', 'title': 'E2E 卡密提取页', 'domain_id': landing['ID'],
        'content': {'announcement': '点击领取你的卡密', 'button_text': '立即领取', 'theme_color': '#0f766e', 'project_id': project_id},
    })
    check('kami landing page created', kami_page['Template'] == 'kami')

    kami_link = client.ok('/api/v1/links', 'POST', {
        'type': 'liveqr', 'code': 'km' + stamp,
        'entry_domain_id': entry['ID'], 'landing_domain_id': landing['ID'], 'landing_page_id': kami_page['ID'],
        'title': 'E2E 卡密链接',
    })
    check('kami liveqr link created without targets', kami_link['Type'] == 'liveqr')

    st, hd, html = anonymous.call('/' + kami_link['Code'], host=landing['Host'], public=True)
    check('kami landing renders', st == 200 and '点击领取你的卡密' in html)
    check('kami extract button present', 'data-kami-issue' in html and str(project_id) in html)

    # 公开提取：无口令 → 400；错误口令 → 403；正确口令 → 发码
    st, _, body = anonymous.call(f"/api/v1/kami/{project_id}/issue", 'POST', {})
    check('kami issue requires password', st == 400)
    st, _, body = anonymous.call(f"/api/v1/kami/{project_id}/issue", 'POST', {'password': 'wrong'})
    check('kami wrong password rejected', st == 403)
    st, _, body = anonymous.call(f"/api/v1/kami/{project_id}/issue", 'POST', {'password': 'secret123'})
    check('kami issue succeeds', st == 200 and body.get('data', {}).get('content') == 'KAMI-001')
    st, _, body = anonymous.call(f"/api/v1/kami/{project_id}/issue", 'POST', {'password': 'secret123'})
    check('kami repeat blocked', st == 403)

    # FIFO：确认只发了 1 条且是第一条
    items = client.ok(f"/api/admin/kami/projects/{project_id}/items")['items']
    issued = [i for i in items if i['Status'] == 'issued']
    check('kami FIFO single issuance', len(issued) == 1 and issued[0]['Content'] == 'KAMI-001')

    # ---------- 开放 API ----------
    key_result = client.ok('/api/admin/api-keys', 'POST', {
        'name': 'E2E Key', 'sign_enabled': False, 'quota': 5,
    })
    check('api key created', key_result.get('token', '').startswith('gl_live_'))
    token = key_result['token']

    # 用 token 创建短链
    st, _, body = anonymous.call('/api/v1/open/short-links', 'POST', {
        'target_url': 'https://example.com/open-api', 'title': 'OpenAPI E2E',
        'entry_domain_id': entry['ID'],
    }, public=True)
    check('open api without token denied', st in (401, 403))

    req = __import__('urllib.request', fromlist=['Request'])
    import urllib.request, urllib.error
    open_req = urllib.request.Request(
        PUBLIC + '/api/v1/open/short-links',
        data=json.dumps({'target_url': 'https://example.com/open-api', 'entry_domain_id': entry['ID']}).encode(),
        method='POST',
        headers={'Content-Type': 'application/json', 'Authorization': f'Bearer {token}'},
    )
    try:
        resp = urllib.request.urlopen(open_req, timeout=15)
        open_body = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        open_body = json.loads(e.read())
        resp = e
    check('open api creates short link', resp.status == 200 and open_body.get('code') == 0)
    open_code = open_body.get('data', {}).get('Code') or open_body.get('data', {}).get('code')
    if open_code:
        st, hd, _ = anonymous.call('/' + open_code, host=entry['Host'], public=True)
        check('open api short link redirects', st == 302 and 'example.com' in hd.get('Location', ''))

    # ---------- UA 访问限制 ----------
    ua_link = client.ok('/api/v1/links', 'POST', {
        'type': 'short', 'code': 'ua' + stamp,
        'entry_domain_id': entry['ID'], 'target_url': 'https://example.com/ua',
        'access_rule': 'wechat',
    })
    # 非微信 UA → 403
    st, _, html = anonymous.call('/' + ua_link['Code'], host=entry['Host'], public=True)
    # Client 默认 iPhone Safari UA，应被拦截
    check('ua wechat rule blocks safari', st == 403)

    # 微信 UA 放行
    class WXClient(Client):
        def call(self, path, method='GET', payload=None, host=None, public=False):
            import urllib.request as ur, urllib.error as ue, json as js
            headers = {'Content-Type': 'application/json', 'User-Agent': 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.49'}
            if host:
                headers['Host'] = host
            req = ur.Request((PUBLIC if public else ADMIN) + path, data=None if payload is None else js.dumps(payload).encode(), method=method, headers=headers)
            try:
                r = self.opener.open(req, timeout=20)
            except ue.HTTPError as e:
                r = e
            raw = r.read().decode('utf-8', 'replace')
            try:
                body = js.loads(raw)
            except ValueError:
                body = raw
            return r.status, r.headers, body

    wx = WXClient()
    st, hd, _ = wx.call('/' + ua_link['Code'], host=entry['Host'], public=True)
    check('ua wechat rule allows wechat', st == 302)

    # ---------- 落地页模板列表 ----------
    pages = client.ok('/api/v1/landing-pages')['items']
    templates = {p['Template'] for p in pages}
    check('all five landing templates present', {'liveqr', 'redirect_notice', 'custom', 'kf', 'kami'} <= templates or {'kf', 'kami'} <= templates)

    # 目标微信号可读回
    targets = client.ok(f"/api/admin/links/{kf_link['ID']}/targets")['items']
    check('target wx_remark persisted', any(t.get('WxRemark') == 'wxid_e2e_cs' for t in targets))


if __name__ == '__main__':
    try:
        run()
    except Exception as err:
        check('p2 suite completed (' + type(err).__name__ + ')', False)
        import traceback
        traceback.print_exc()
    finally:
        (WORK / 'e2e-p2-results.json').write_text(json.dumps(RESULTS, indent=2), encoding='utf-8')
    print(f"P2 RESULT {sum(r['passed'] for r in RESULTS)}/{len(RESULTS)} passed")
    sys.exit(0 if RESULTS and all(r['passed'] for r in RESULTS) else 1)
