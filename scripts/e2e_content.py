"""Additive local integration checks: real URLs, no Host-header substitution.
Leaves named review fixtures; never resets configuration or deletes business data.
"""
import base64
import json
import time
import subprocess
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor
from urllib.parse import urlparse
from e2e_local import Client, password, PUBLIC, check, RESULTS, WORK

def run():
    c=Client()
    c.ok('/api/v1/auth/local/login','POST',{'username':'admin','password':password()})
    from repair_demo_qr import upload as upload_fixture
    subprocess.run(['node',str(WORK.parent/'scripts/generate-test-qrs.cjs')],check=True)
    first_image=upload_fixture(c,WORK/'demo-qr-first.png')
    second_image=upload_fixture(c,WORK/'demo-qr-second.png')
    first_path=urlparse(first_image).path;second_path=urlparse(second_image).path
    port=urlparse(PUBLIC).port
    domains=c.ok('/api/admin/domains')['items']
    def domain(host,kind):
        found=next((d for d in domains if d['Host']==host and d['Type']==kind),None)
        return found or c.ok('/api/admin/domains','POST',{'host':host,'type':kind,'scheme':'http','remark':'真实 URL 自动验收'})
    entry=domain(f'localhost:{port}','entry')
    landing=domain(f'127.0.0.1:{port}','landing')
    check('explicit public port retained',entry['Host']==f'localhost:{port}')
    stamp=str(int(time.time()))
    link=c.ok('/api/v1/links','POST',{'type':'short','entry_domain_id':entry['ID'],'title':'真实跳转验收 '+stamp,'target_url':'https://example.com/verified?from=gravitylink'})
    row=next(x for x in c.ok('/api/v1/links')['items'] if x['ID']==link['ID'])
    def visit(address):
        try: r=c.opener.open(address,timeout=15)
        except urllib.error.HTTPError as e: r=e
        return r.status,r.headers,r.read().decode(errors='replace')
    status,headers,_=visit(row['PublicURL'])
    check('copied canonical URL redirects without injected Host',status==302 and headers['Location']=='https://example.com/verified?from=gravitylink')
    page=c.ok('/api/v1/landing-pages','POST',{'template':'liveqr','title':'群码验收 '+stamp,'domain_id':landing['ID'],'content':{'headline':'自动验收群码','subtext':'请长按识别'}})
    qr=c.ok('/api/v1/links','POST',{'type':'liveqr','entry_domain_id':entry['ID'],'landing_domain_id':landing['ID'],'landing_page_id':page['ID'],'title':'群码验收 '+stamp,'strategy':{'mode':'round_robin','targets':[{'target_url':first_image+'?fixture=first.png','scan_limit':1},{'target_url':second_image+'?fixture=second.png'}]}})
    url=f"http://{entry['Host']}/{qr['Code']}"
    status,headers,_=visit(url)
    check('liveqr entry redirects to configured landing port',status==302 and headers.get('Location','').startswith(f"http://{landing['Host']}/"))
    dest=headers['Location']
    status,_,html=visit(dest);check('first QR image rendered',status==200 and first_path in html)
    status,_,html=visit(dest);check('threshold switches to next QR',status==200 and second_path in html)
    targets=c.ok(f"/api/admin/links/{qr['ID']}/targets")['items']
    second=targets[1]; second.update({'Label':'到期测试','Status':'active','Owner':'验收群主','ExpireAt':'2020-01-01T00:00:00Z'})
    c.ok(f"/api/admin/links/{qr['ID']}/targets",'POST',second)
    status,_,html=visit(dest);check('expired and exhausted targets unavailable',status==200 and first_path not in html and second_path not in html and '暂无' in html)
    second.update({'ExpireAt':None,'ScanLimit':None,'Status':'active'})
    c.ok(f"/api/admin/links/{qr['ID']}/targets",'POST',second)
    check('target edit preserves scan count',c.ok(f"/api/admin/links/{qr['ID']}/targets")['items'][1]['ScanCount']==second['ScanCount'])
    check('target resumes after expiry cleared',visit(dest)[0]==200)
    concurrent=c.ok('/api/v1/links','POST',{'type':'liveqr','entry_domain_id':entry['ID'],'landing_domain_id':landing['ID'],'landing_page_id':page['ID'],'title':'并发阈值验收 '+stamp,'strategy':{'mode':'round_robin','targets':[{'target_url':first_image+'?fixture=limited.png','scan_limit':3},{'target_url':second_image+'?fixture=fallback.png'}]}})
    concurrent_url=f"http://{landing['Host']}/{concurrent['Code']}"
    with ThreadPoolExecutor(max_workers=10) as pool:
        pages=list(pool.map(lambda _:visit(concurrent_url)[2],range(20)))
    check('concurrent threshold never over-allocates',sum(first_path in x for x in pages)==3 and sum(second_path in x for x in pages)==17)
    pixel=(WORK/'demo-qr-first.png').read_bytes()
    boundary='GravityLinkBoundary'
    def upload(data,filename):
        body=(f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="{filename}"\r\nContent-Type: application/octet-stream\r\n\r\n').encode()+data+f'\r\n--{boundary}--\r\n'.encode()
        from e2e_local import ADMIN
        req=urllib.request.Request(ADMIN+'/api/admin/materials',data=body,headers={'Content-Type':'multipart/form-data; boundary='+boundary},method='POST')
        try:r=c.opener.open(req,timeout=15)
        except urllib.error.HTTPError as e:r=e
        return r.status,json.loads(r.read())
    status,body=upload(pixel,'pixel.png');check('image upload accepted',status==200)
    path=body['data']['Path']
    check('uploaded image publicly reachable',visit(f"http://{entry['Host']}"+path)[0]==200)
    check('HTML upload rejected',upload(b'<html><script>alert(1)</script></html>','image.png')[0]==400)
    card=c.ok('/api/admin/share-cards','POST',{'DomainID':entry['ID'],'Title':'分享卡片验收 '+stamp,'Description':'标题摘要与封面预览','ImageURL':f"http://{entry['Host']}"+path,'TargetURL':'https://example.com','Status':'active'})
    cardurl=f"http://{entry['Host']}/card/{card['ID']}"
    status,_,html=visit(cardurl);check('share page contains content and SDK setup',status==200 and '分享卡片验收' in html and 'updateAppMessageShareData' in html)
    check('share signature rejects foreign origin',visit(cardurl+'/signature?url=https%3A%2F%2Fevil.example%2Fcard%2F1')[0]==400)
    card['Status']='disabled';c.ok(f"/api/admin/share-cards/{card['ID']}",'PUT',card)
    check('disabled share card unavailable',visit(cardurl)[0]==404)
    card['Status']='active';c.ok(f"/api/admin/share-cards/{card['ID']}",'PUT',card)
    check('anonymous material API denied',Client().call('/api/admin/materials')[0]==401)
    WORK.joinpath('content-e2e.json').write_text(json.dumps({'results':RESULTS,'short_url':row['PublicURL'],'qr_url':url,'card_url':cardurl},ensure_ascii=False,indent=2),encoding='utf-8')
    if not all(x['passed'] for x in RESULTS):raise SystemExit(1)

if __name__=='__main__':run()
