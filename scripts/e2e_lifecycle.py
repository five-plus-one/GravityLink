"""Statistics flush, restart and reset/reinitialization checks on test project."""
from datetime import datetime, timedelta, timezone
import json
import subprocess
import sys
import time
from e2e_local import ADMIN, ENV, WORK, Client, RESULTS, check, password, sql

def redis(*args):
    result = subprocess.run(['docker', 'exec', 'gravitylink-redis-1', 'redis-cli', '--raw', *map(str, args)], text=True, capture_output=True)
    if result.returncode:
        raise RuntimeError('Redis fixture command failed')
    return result.stdout.strip()

def ready(client):
    for _ in range(30):
        try:
            if client.call('/api/v1/health')[0] == 200:
                return
        except Exception:
            pass
        time.sleep(1)
    raise RuntimeError('Runtime did not become ready')

def run():
    fixture = json.loads((WORK / 'e2e-fixture.json').read_text())
    link_id = fixture['short_id']
    client = Client()
    pwd = password()
    def login():
        return client.ok('/api/v1/auth/local/login', 'POST', {'username': 'admin', 'password': pwd})
    login()
    day = (datetime.now(timezone.utc) - timedelta(days=1)).strftime('%Y%m%d')
    date = f'{day[:4]}-{day[4:6]}-{day[6:]}'
    redis('SET', f'stat:pv:{link_id}:{day}', 7)
    redis('PFADD', f'stat:uv:{link_id}:{day}', 'e2e-visitor-a', 'e2e-visitor-b')
    redis('SET', f'stat:hourly:{link_id}:{day}:09', 7)
    redis('HSET', f'stat:dev:{link_id}:{day}', 'mobile|iOS|Safari', 7)
    redis('HSET', f'stat:geo:{link_id}:{day}', 'TestCountry|TestProvince', 7)
    print('Waiting for real minute worker to flush yesterday fixtures...', flush=True)
    for _ in range(38):
        if redis('EXISTS', f'stat:geo:{link_id}:{day}') == '0':
            break
        time.sleep(2)
    summary = client.ok(f'/api/v1/stats/{link_id}/summary')
    check('yesterday PV flushed to SQL', summary['yesterday_pv'] == 7)
    points = client.ok(f'/api/v1/stats/{link_id}/daily?start={date}&end={date}')
    check('historical daily PV/UV', any(p['pv'] == 7 and p['uv'] == 2 for p in points))
    hours = client.ok(f'/api/v1/stats/{link_id}/hourly?date={date}')
    check('historical hourly aggregation', hours[9]['pv'] == 7)
    devices = client.ok(f'/api/v1/stats/{link_id}/device?start={date}&end={date}')
    check('device aggregation persisted', any(p['label'] == 'mobile' and p['value'] == 7 for p in (devices['device'] or [])))
    geo = client.ok(f'/api/v1/stats/{link_id}/geo?start={date}&end={date}')
    check('geo aggregation persisted (synthetic geography)', any(p['label'] == 'TestProvince' and p['value'] == 7 for p in (geo or [])))
    check('flushed Redis keys removed', redis('EXISTS', f'stat:pv:{link_id}:{day}', f'stat:uv:{link_id}:{day}', f'stat:hourly:{link_id}:{day}:09') == '0')
    count = client.ok('/api/v1/links')['total']
    restarted = subprocess.run(['docker', 'restart', 'gravitylink-gravitylink-1'], capture_output=True)
    check('application restart', restarted.returncode == 0)
    ready(client)
    check('session survives restart', client.call('/api/v1/auth/me')[0] == 200)
    check('business data survives restart', client.ok('/api/v1/links')['total'] == count)
    check('settings survive restart', client.ok('/api/admin/configs')['configs'].get('public.home.title') == 'E2E configured home')
    check('statistics survive restart', client.ok(f'/api/v1/stats/{link_id}/summary')['yesterday_pv'] == 7)
    client.ok('/api/admin/system/reset', 'POST', {'confirmation': 'RESET GRAVITYLINK', 'password': pwd})
    check('reset returns to initialization', client.ok('/api/setup/status')['setup_required'])
    check('old session no longer grants access', client.call('/api/v1/links')[0] != 200)
    client.ok('/api/setup/auth/local', 'POST', {'auth': {'username': 'admin', 'password': pwd, 'admin_base_url': ADMIN}})
    config = {'database': {'host': 'mysql', 'port': '3306', 'database': 'gravitylink', 'user': 'gravitylink', 'password': ENV['MYSQL_PASSWORD']}, 'redis': {'host': 'redis', 'port': '6379'}}
    client.ok('/api/setup/database/test', 'POST', config)
    client.ok('/api/setup/complete', 'POST', config)
    check('reinitialize existing database', not client.ok('/api/setup/status')['setup_required'])
    check('administrator login after reinitialize', login()['role'] == 'super_admin')
    check('reset preserves business records', client.ok('/api/v1/links')['total'] == count)
    check('reset clears system settings', not client.ok('/api/admin/configs')['configs'])
    check('reset preserves statistics', client.ok(f'/api/v1/stats/{link_id}/summary')['yesterday_pv'] == 7)
    check('single active super administrator', sql("SELECT COUNT(*) FROM users WHERE role='super_admin' AND status='active';") == '1')

if __name__ == '__main__':
    try:
        run()
    except Exception as err:
        check('lifecycle completed (' + type(err).__name__ + ')', False)
    finally:
        (WORK / 'e2e-lifecycle-results.json').write_text(json.dumps(RESULTS, indent=2), encoding='utf-8')
    print(f"RESULT {sum(r['passed'] for r in RESULTS)}/{len(RESULTS)} passed")
    sys.exit(0 if RESULTS and all(r['passed'] for r in RESULTS) else 1)
