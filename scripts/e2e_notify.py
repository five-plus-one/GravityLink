"""验收通知体系 API：渠道读写、事件配置。不发送真实邮件。"""
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
from e2e_local import Client, password, check, RESULTS  # noqa: E402

c = Client()
c.ok('/api/v1/auth/local/login', 'POST', {'username': 'admin', 'password': password()})

# 健康
status, _, body = c.call('/api/v1/health')
check('health ok', status == 200 and body.get('code') == 0)

# 渠道列表：应含 wecom/http/smtp
status, _, body = c.call('/api/admin/notify/channels')
check('list channels', status == 200)
items = body['data']['items']
types = {i['type'] for i in items}
check('channel types', types >= {'wecom', 'http', 'smtp'})
smtp = next(i for i in items if i['type'] == 'smtp')
check('smtp qq defaults', smtp['settings'].get('host') == 'smtp.qq.com' and smtp['settings'].get('port') == 465)
check('smtp secret not leaked', 'password_enc' not in smtp['settings'] and 'password' not in smtp['settings'])

# 事件配置
ev_status, _, ev_body = c.call('/api/admin/notify/events')
check('list events', ev_status == 200)
cfg = ev_body['data']
check('events default enabled', cfg['events']['target_expiring_soon']['enabled'] is True)
check('lead hours default', cfg['expiring_lead_hours'] == 24)
check('qr default days', cfg['qr_default_expire_days'] == 7)

# 保存 SMTP（不启用，不填授权码）
payload = {
    'channels': [
        {'type': 'wecom', 'enabled': False, 'settings': {'webhook_url': ''}},
        {'type': 'http', 'enabled': False, 'settings': {'url': ''}},
        {
            'type': 'smtp',
            'enabled': False,
            'settings': {
                'host': 'smtp.qq.com',
                'port': 465,
                'encryption': 'ssl',
                'from': 'test@example.com',
                'username': 'test@example.com',
                'to': ['ops@example.com'],
            },
        },
    ]
}
status, _, body = c.call('/api/admin/notify/channels', 'PUT', payload)
check('save channels disabled smtp', status == 200)

# 启用但缺授权码应拒绝
payload['channels'][2]['enabled'] = True
status, _, body = c.call('/api/admin/notify/channels', 'PUT', payload)
check('enable smtp without secret rejected', status == 400)

# 测试发送未配置渠道
status, _, body = c.call('/api/admin/notify/channels/smtp/test', 'POST')
check('test smtp without secret fails clearly', status == 400)

# 恢复关闭
payload['channels'][2]['enabled'] = False
status, _, body = c.call('/api/admin/notify/channels', 'PUT', payload)
check('restore channels', status == 200)

failed = [r for r in RESULTS if not r['passed']]
print('PASS' if not failed else 'FAILED %d' % len(failed))
sys.exit(1 if failed else 0)
