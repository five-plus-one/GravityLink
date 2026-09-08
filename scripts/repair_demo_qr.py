"""Repair only known test fixture image URLs; do not replace real group images."""
import json
import urllib.request
from e2e_local import Client,password,ADMIN,PUBLIC,WORK

def upload(c,path):
    boundary='GravityLinkDemoRepair'
    body=(f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="{path.name}"\r\nContent-Type: image/png\r\n\r\n').encode()+path.read_bytes()+f'\r\n--{boundary}--\r\n'.encode()
    req=urllib.request.Request(ADMIN+'/api/admin/materials',data=body,headers={'Content-Type':'multipart/form-data; boundary='+boundary},method='POST')
    return PUBLIC+json.loads(c.opener.open(req).read())['data']['Path']

if __name__=='__main__':
    c=Client();c.ok('/api/v1/auth/local/login','POST',{'username':'admin','password':password()})
    images=[upload(c,WORK/f'demo-qr-{name}.png') for name in ('first','second')]
    known={'https://example.com/'+p+'.png' for p in ('limited','fallback','first','second','one','two')}
    known.add('http://localhost:18080/uploads/6b7b5e3fc610c04e611cfe17a3ea40dc.png')
    changed=[]
    for link in c.ok('/api/v1/links')['items']:
        if link['Type']!='liveqr' or not (link['Title'] or '').startswith(('E2E','群码验收','并发阈值验收','浏览器群活码验收')):continue
        for i,t in enumerate(c.ok(f"/api/admin/links/{link['ID']}/targets")['items']):
            if t['TargetURL'] not in known or t['Status'] not in ('active','disabled'):continue
            t.update({'TargetURL':images[i%2],'Label':f'演示二维码 {i+1}（非真实微信群）'})
            c.ok(f"/api/admin/links/{link['ID']}/targets",'POST',t);changed.append(t['ID'])
    print('Repaired demo targets:',len(changed))
    WORK.joinpath('demo-qr-repair.json').write_text(json.dumps({'target_ids':changed,'images':images}),encoding='utf-8')
