"""Destructive test fixtures only in the dedicated gravitylink Compose project.
Run from the repository root on Windows. Existing deploy volumes are untouched.
The generated admin password is stored with Windows DPAPI, never in output.
"""
import ctypes
import http.cookiejar
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import sys
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
WORK = ROOT / 'workspace'
ENV = {}
for line in (ROOT / 'deploy/.env').read_text(encoding='utf-8-sig').splitlines():
    if '=' in line and not line.lstrip().startswith('#'):
        key, value = line.split('=', 1)
        ENV[key.strip()] = value.strip().strip('"').strip("'")
ADMIN = 'http://127.0.0.1:' + ENV.get('ADMIN_PORT', '8081')
PUBLIC = 'http://127.0.0.1:' + ENV.get('APP_PORT', '8080')
RESULTS = []

class Blob(ctypes.Structure):
    _fields_ = [('size', ctypes.c_ulong), ('data', ctypes.POINTER(ctypes.c_byte))]

def protect(data, decrypt=False):
    buffer = ctypes.create_string_buffer(data)
    source = Blob(len(data), ctypes.cast(buffer, ctypes.POINTER(ctypes.c_byte)))
    target = Blob()
    function = ctypes.windll.crypt32.CryptUnprotectData if decrypt else ctypes.windll.crypt32.CryptProtectData
    if not function(ctypes.byref(source), None, None, None, None, 1, ctypes.byref(target)):
        raise RuntimeError('Windows credential protection failed')
    try:
        return ctypes.string_at(target.data, target.size)
    finally:
        ctypes.windll.kernel32.LocalFree(target.data)

def password():
    file = WORK / 'local-admin.dpapi'
    if file.exists():
        return protect(file.read_bytes(), True).decode()
    value = secrets.token_urlsafe(24)
    file.write_bytes(protect(value.encode()))
    return value

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None

class Client:
    def __init__(self):
        self.jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(self.jar), NoRedirect())

    def call(self, path, method='GET', payload=None, host=None, public=False):
        headers = {'Content-Type': 'application/json', 'User-Agent': 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1'}
        if host:
            headers['Host'] = host
        req = urllib.request.Request((PUBLIC if public else ADMIN) + path,
            data=None if payload is None else json.dumps(payload).encode(), method=method, headers=headers)
        try:
            response = self.opener.open(req, timeout=20)
        except urllib.error.HTTPError as err:
            response = err
        raw = response.read().decode('utf-8', errors='replace')
        try:
            body = json.loads(raw)
        except ValueError:
            body = raw
        return response.status, response.headers, body

    def ok(self, path, method='GET', payload=None):
        status, _, body = self.call(path, method, payload)
        if status != 200 or not isinstance(body, dict) or body.get('code') != 0:
            raise AssertionError(f'{method} {path}: HTTP {status}, expected success')
        return body['data']

def check(name, condition):
    RESULTS.append({'name': name, 'passed': bool(condition)})
    print(('PASS ' if condition else 'FAIL ') + name, flush=True)

def sql(query):
    result = subprocess.run(['docker', 'exec', '-i', 'gravitylink-mysql-1', 'sh', '-c',
        'MYSQL_PWD="$MYSQL_PASSWORD" mysql -ugravitylink -N -B gravitylink'], input=query,
        text=True, capture_output=True, encoding='utf-8')
    if result.returncode:
        raise RuntimeError('Test database fixture command failed')
    return result.stdout.strip()

def run():
    client, anonymous = Client(), Client()
    pwd = password()
    state = client.ok('/api/setup/status')
    if state['setup_required']:
        check('fresh initialization required', True)
        check('public port hides initialization', anonymous.call('/api/setup/status', public=True)[0] != 200)
        check('short password rejected', client.call('/api/setup/auth/local', 'POST', {'auth': {'username': 'admin', 'password': 'short'}})[0] == 400)
        client.ok('/api/setup/auth/local', 'POST', {'auth': {'username': 'admin', 'password': pwd, 'admin_base_url': ADMIN}})
        config = {'database': {'host': 'mysql', 'port': '3306', 'database': 'gravitylink', 'user': 'gravitylink', 'password': ENV['MYSQL_PASSWORD']}, 'redis': {'host': 'redis', 'port': '6379'}}
        connectivity = client.ok('/api/setup/database/test', 'POST', config)
        check('MySQL and Redis connectivity', connectivity['mysql'] and connectivity['redis'])
        client.ok('/api/setup/complete', 'POST', config)
        check('initialize runtime', not client.ok('/api/setup/status')['setup_required'])
    else:
        check('initialized runtime available', True)
    check('setup completion locked', client.call('/api/setup/complete', 'POST', {})[0] == 409)
    check('setup authentication locked', client.call('/api/setup/auth/local', 'POST', {})[0] == 409)
    check('setup connectivity locked', client.call('/api/setup/database/test', 'POST', {})[0] == 409)
    check('anonymous links denied', anonymous.call('/api/v1/links')[0] == 401)
    check('wrong password denied', client.call('/api/v1/auth/local/login', 'POST', {'username': 'admin', 'password': 'incorrect'})[0] == 401)
    account = client.ok('/api/v1/auth/local/login', 'POST', {'username': 'admin', 'password': pwd})
    check('local login creates super administrator', account['role'] == 'super_admin')
    check('profile session', client.ok('/api/v1/auth/me')['username'] == 'admin')
    check('session cookie httponly', any(c.has_nonstandard_attr('HttpOnly') for c in client.jar))
    check('health ready', client.call('/api/v1/health')[0] == 200)
    status_text = json.dumps(client.ok('/api/setup/status'))
    check('setup status excludes secrets', pwd not in status_text and ENV['MYSQL_PASSWORD'] not in status_text)
    for route in ['/', '/setup', '/login', '/links', '/domains', '/landing-pages', '/stats', '/profile', '/settings', '/users']:
        status, _, body = anonymous.call(route)
        check('SPA route ' + route, status == 200 and isinstance(body, str) and 'id="app"' in body)
    html = anonymous.call('/')[2]
    scripts = re.findall(r'(?:src|href)="(/assets/[^" ]+)"', html)
    check('production static assets', bool(scripts) and all(anonymous.call(asset)[0] == 200 for asset in scripts))
    stamp = str(int(time.time()))[-8:]
    def domain(kind):
        return client.ok('/api/admin/domains', 'POST', {'host': f'{kind}-{stamp}.test', 'type': kind, 'scheme': 'http'})
    entry, transit, landing = [domain(kind) for kind in ['entry', 'transit', 'landing']]
    check('three domain types', len(client.ok('/api/admin/domains')['items']) >= 3)
    check('duplicate domain rejected', client.call('/api/admin/domains', 'POST', {'host': entry['Host'], 'type': 'entry'})[0] == 409)
    check('invalid domain rejected', client.call('/api/admin/domains', 'POST', {'host': 'bad/domain', 'type': 'entry'})[0] == 400)
    client.ok('/api/admin/domains/' + str(entry['ID']), 'PUT', {'remark': 'E2E verified'})
    check('domain edit', any(d['Remark'] == 'E2E verified' for d in client.ok('/api/admin/domains')['items']))
    check('public home', anonymous.call('/', host=entry['Host'], public=True)[0] == 200)
    check('unknown domain 404', anonymous.call('/absent', host='unknown.test', public=True)[0] == 404)
    def link(kind='short', **extra):
        payload = {'type': kind, 'code': 'e' + stamp + secrets.token_hex(2), 'entry_domain_id': entry['ID'], 'target_url': 'https://example.com/a', 'title': 'E2E ' + kind}
        payload.update(extra)
        return client.ok('/api/v1/links', 'POST', payload)
    def visit(item, host=None):
        return anonymous.call('/' + item['Code'], host=host or entry['Host'], public=True)
    short = link()
    check('short link redirect', visit(short)[1].get('Location') == 'https://example.com/a')
    check('duplicate code rejected', client.call('/api/v1/links', 'POST', {'code': short['Code'], 'entry_domain_id': entry['ID'], 'target_url': 'https://example.com'})[0] == 409)
    check('unsafe short URL rejected', client.call('/api/v1/links', 'POST', {'entry_domain_id': entry['ID'], 'target_url': 'javascript:alert(1)'})[0] == 400)
    client.ok('/api/v1/links/' + str(short['ID']), 'PUT', {'target_url': 'https://example.com/updated'})
    check('edit invalidates redirect cache', visit(short)[1].get('Location') == 'https://example.com/updated')
    client.ok('/api/v1/links/' + str(short['ID']), 'PUT', {'status': 'disabled'})
    check('disabled short link blocked', visit(short)[0] == 404)
    client.ok('/api/v1/links/' + str(short['ID']), 'PUT', {'status': 'active'})
    check('reenabled short link works', visit(short)[0] == 302)
    expired = link(expire_at='2020-01-01T00:00:00Z')
    check('expired short link 410', visit(expired)[0] == 410)
    channel = link('channel', channel={'utm_source': 'wechat', 'utm_medium': 'qr', 'utm_campaign': 'e2e'})
    check('channel UTM redirect', 'utm_source=wechat' in visit(channel)[1].get('Location', ''))
    response = visit(short, transit['Host'])
    check('transit HTML and no-store', response[0] == 200 and response[1].get('Cache-Control') == 'no-store')
    def page(template, content):
        return client.ok('/api/v1/landing-pages', 'POST', {'template': template, 'title': 'E2E ' + template, 'domain_id': landing['ID'], 'content': content})
    qrpage = page('liveqr', {'headline': 'E2E QR', 'show_logo': False})
    from repair_demo_qr import upload as upload_fixture
    subprocess.run(['node', str(ROOT/'scripts/generate-test-qrs.cjs')],check=True)
    first_image=upload_fixture(client,WORK/'demo-qr-first.png')
    second_image=upload_fixture(client,WORK/'demo-qr-second.png')
    qr = link('liveqr', landing_domain_id=landing['ID'], landing_page_id=qrpage['ID'], strategy={'mode': 'round_robin', 'targets': [{'target_url': first_image+'?fixture=one.png', 'scan_limit': 1}, {'target_url': second_image+'?fixture=two.png'}]})
    check('live QR entry redirects to landing', landing['Host'] in visit(qr)[1].get('Location', ''))
    first, second = visit(qr, landing['Host']), visit(qr, landing['Host'])
    check('live QR landing renders', first[0] == 200 and 'one.png' in first[2] and first[1].get('Cache-Control') == 'no-store')
    check('sequential target switches at scan limit', 'two.png' in second[2])
    client.ok('/api/v1/links/' + str(qr['ID']), 'PUT', {'status': 'disabled'})
    check('disabled direct landing blocked', visit(qr, landing['Host'])[0] == 404)
    client.ok('/api/v1/links/' + str(qr['ID']), 'PUT', {'status': 'active', 'expire_at': '2020-01-01T00:00:00Z'})
    check('expired direct landing blocked', visit(qr, landing['Host'])[0] == 410)
    limited = link('liveqr', landing_domain_id=landing['ID'], landing_page_id=qrpage['ID'], strategy={'mode': 'weighted', 'targets': [{'target_url': first_image+'?fixture=limited.png', 'weight': 2, 'scan_limit': 1}]})
    check('weighted target works', 'limited.png' in visit(limited, landing['Host'])[2])
    check('scan limit exhausted', 'limited.png' not in visit(limited, landing['Host'])[2])
    unsafe = {'type': 'liveqr', 'entry_domain_id': entry['ID'], 'landing_domain_id': landing['ID'], 'landing_page_id': qrpage['ID'], 'strategy': {'mode': 'weighted', 'targets': [{'target_url': 'javascript:alert(1)'}]}}
    check('unsafe routing target rejected', client.call('/api/v1/links', 'POST', unsafe)[0] == 400)
    notice = page('redirect_notice', {'message': 'E2E Notice', 'countdown': 3})
    notice_link = link(landing_domain_id=landing['ID'], landing_page_id=notice['ID'])
    check('notice template renders', 'E2E Notice' in visit(notice_link, landing['Host'])[2])
    custom = page('custom', {'html': '<p>E2E Custom</p><script>alert(1)</script>'})
    custom_link = link(landing_domain_id=landing['ID'], landing_page_id=custom['ID'])
    content = visit(custom_link, landing['Host'])[2]
    check('custom template escapes executable script', 'E2E Custom' in content and '<script>alert(1)</script>' not in content and '&lt;script>alert(1)&lt;/script>' in content)
    check('domain in use protected', client.call('/api/admin/domains/' + str(entry['ID']), 'DELETE')[0] == 409)
    spare = client.ok('/api/admin/domains', 'POST', {'host': f'spare-{stamp}.test', 'type': 'entry'})
    client.ok('/api/admin/domains/' + str(spare['ID']), 'DELETE')
    check('unused domain deletion', all(d['ID'] != spare['ID'] for d in client.ok('/api/admin/domains')['items']))
    deleted = link()
    visit(deleted)
    client.ok('/api/v1/links/' + str(deleted['ID']), 'DELETE')
    check('delete invalidates cache', visit(deleted)[0] == 404)
    client.ok('/api/admin/configs', 'PUT', {'configs': {'public.home.title': 'E2E configured home'}})
    check('settings effective on public home', 'E2E configured home' in anonymous.call('/', host=entry['Host'], public=True)[2])
    for _ in range(10):
        summary = client.ok(f"/api/v1/stats/{short['ID']}/summary")
        if summary['today_pv'] >= 3:
            break
        time.sleep(0.2)
    check('real PV and UV captured', summary['today_pv'] >= 3 and summary['today_uv'] >= 1)
    check('daily stats endpoint', isinstance(client.ok(f"/api/v1/stats/{short['ID']}/daily"), list))
    check('hourly stats 24 buckets', len(client.ok(f"/api/v1/stats/{short['ID']}/hourly")) == 24)
    check('device stats endpoint', 'device' in client.ok(f"/api/v1/stats/{short['ID']}/device"))
    check('geo stats endpoint', client.call(f"/api/v1/stats/{short['ID']}/geo")[0] == 200)
    for _ in range(15):
        if int(sql(f"SELECT COUNT(*) FROM access_logs WHERE link_id={short['ID']};")) > 0:
            break
        time.sleep(1)
    check('async access log persisted', int(sql(f"SELECT COUNT(*) FROM access_logs WHERE link_id={short['ID']};")) > 0)
    sql("INSERT INTO users (auth_source,username,password_hash,role,status,created_at,updated_at) SELECT 'local','e2e_reader',password_hash,'user','active',NOW(),NOW() FROM users WHERE username='admin' ON DUPLICATE KEY UPDATE status='active',role='user';")
    reader = Client()
    reader.ok('/api/v1/auth/local/login', 'POST', {'username': 'e2e_reader', 'password': pwd})
    check('reader can read links', reader.call('/api/v1/links')[0] == 200)
    check('reader cannot read admin domains', reader.call('/api/admin/domains')[0] == 403)
    check('reader cannot create links', reader.call('/api/v1/links', 'POST', {})[0] == 403)
    users = client.ok('/api/admin/users')['items']
    reader_id = next(u['id'] for u in users if u['username'] == 'e2e_reader')
    client.ok(f'/api/admin/users/{reader_id}/role', 'PUT', {'role': 'admin'})
    check('role update effective', reader.call('/api/admin/domains')[0] == 200)
    check('admin cannot reset system', reader.call('/api/admin/system/reset', 'POST', {'confirmation': 'RESET GRAVITYLINK', 'password': pwd})[0] == 403)
    client.ok(f'/api/admin/users/{reader_id}/status', 'PUT', {'status': 'disabled'})
    check('disable revokes sessions', reader.call('/api/v1/links')[0] in (401, 403))
    check('reset requires confirmation', client.call('/api/admin/system/reset', 'POST', {'confirmation': 'wrong'})[0] == 400)
    check('reset requires correct password', client.call('/api/admin/system/reset', 'POST', {'confirmation': 'RESET GRAVITYLINK', 'password': 'wrong'})[0] == 401)
    client.ok('/api/v1/auth/logout', 'POST', {})
    check('logout revokes session', client.call('/api/v1/auth/me')[0] == 401)
    (WORK / 'e2e-fixture.json').write_text(json.dumps({'short_id': short['ID'], 'short_code': short['Code'], 'entry_host': entry['Host'], 'admin_url': ADMIN, 'public_url': PUBLIC}), encoding='utf-8')

if __name__ == '__main__':
    try:
        run()
    except Exception as err:
        check('suite completed (' + type(err).__name__ + ')', False)
        # Never include response bodies or credential-bearing exceptions.
    finally:
        (WORK / 'e2e-results.json').write_text(json.dumps(RESULTS, indent=2), encoding='utf-8')
    print(f"RESULT {sum(r['passed'] for r in RESULTS)}/{len(RESULTS)} passed")
    sys.exit(0 if RESULTS and all(r['passed'] for r in RESULTS) else 1)
