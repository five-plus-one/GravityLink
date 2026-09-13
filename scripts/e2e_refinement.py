"""Integration checks against the isolated local acceptance services."""
import datetime as dt,json,secrets,subprocess,time,urllib.request,urllib.error,hashlib,hmac,base64
from urllib.parse import urlencode,urlparse
import e2e_acceptance as a
c=a.login();checks=[]
def check(name,value):
 checks.append({'name':name,'passed':bool(value)});print(('PASS ' if value else 'FAIL ')+name,flush=True)
 if not value:raise AssertionError(name)
check('version migration recorded',a.sql('SELECT version FROM schema_versions')=='2026091401')
items=c.ok('/api/v1/links')['items'];lid=next(i['ID'] for i in items if i['Type']=='short')
c.ok(f'/api/v1/links/{lid}','PUT',{'category':'秋季推广','tags':['公众号',' 秋季 ','公众号']})
item=next(i for i in c.ok('/api/v1/links')['items'] if i['ID']==lid)
check('labels saved and deduplicated',item['Category']=='秋季推广' and item['Tags']==['公众号','秋季'])
check('kami subtype returned',any(i['Kind']=='kami' for i in items))
q=urlencode({'link_id':lid,'start':'2026-09-13T00:00:00+08:00','end':'2026-09-13T01:00:00+08:00'})
w=c.ok('/api/v1/stats/window?'+q);v=c.ok('/api/v1/stats/visitors?'+q)
check('exact statistics match visitor records',w['pv']==v['total']==1 and w['uv']==1)
check('device totals use exact range',sum(i['value'] for i in w['device']['device'])==1)
hours=c.ok(f'/api/v1/stats/{lid}/hourly');start=dt.datetime.fromisoformat(hours[0]['start']);end=dt.datetime.fromisoformat(hours[-1]['end'])
check('24 consecutive one-hour buckets',len(hours)==24 and end-start==dt.timedelta(hours=24) and all(hours[i]['end']==hours[i+1]['start'] for i in range(23)))
q=urlencode({'link_id':lid,'start':start.isoformat(),'end':end.isoformat()})
check('rolling hourly total matches records',sum(i['pv'] for i in hours)==c.ok('/api/v1/stats/visitors?'+q)['total'])
check('future hours excluded',end<=dt.datetime.now(dt.timezone.utc))
# Separate local S3 service; credentials remain in memory.
key='acceptance';secret=secrets.token_hex(24)
r=subprocess.run(['docker','run','-d','--name','gravitylink-acceptance-s3','--network','gravitylink-acceptance_default','-p','127.0.0.1:29000:9000','-e','MINIO_ROOT_USER='+key,'-e','MINIO_ROOT_PASSWORD='+secret,'quay.io/minio/minio:RELEASE.2024-08-17T01-24-54Z','server','/data'],capture_output=True,text=True)
if r.returncode:
 info=json.loads(subprocess.check_output(['docker','inspect','gravitylink-acceptance-s3']))[0]
 env=dict(v.split('=',1) for v in info['Config']['Env'] if '=' in v);key=env['MINIO_ROOT_USER'];secret=env['MINIO_ROOT_PASSWORD']
def signed(method,path,data=b'',query=''):
 now=dt.datetime.now(dt.timezone.utc);stamp=now.strftime('%Y%m%dT%H%M%SZ');day=stamp[:8];sha=hashlib.sha256(data).hexdigest();scope=day+'/us-east-1/s3/aws4_request';host='127.0.0.1:29000';names='host;x-amz-content-sha256;x-amz-date'
 canonical='\n'.join([method,path,query,f'host:{host}\nx-amz-content-sha256:{sha}\nx-amz-date:{stamp}\n',names,sha]);to_sign='AWS4-HMAC-SHA256\n'+stamp+'\n'+scope+'\n'+hashlib.sha256(canonical.encode()).hexdigest()
 signing=('AWS4'+secret).encode()
 for part in [day,'us-east-1','s3','aws4_request']:signing=hmac.new(signing,part.encode(),hashlib.sha256).digest()
 signature=hmac.new(signing,to_sign.encode(),hashlib.sha256).hexdigest()
 req=urllib.request.Request('http://'+host+path+('?' + query if query else ''),data=data if method=='PUT' else None,method=method,headers={'x-amz-date':stamp,'x-amz-content-sha256':sha,'Authorization':f'AWS4-HMAC-SHA256 Credential={key}/{scope}, SignedHeaders={names}, Signature={signature}'})
 return urllib.request.urlopen(req,timeout=15).read()
for _ in range(25):
 try:signed('PUT','/images');break
 except urllib.error.HTTPError as e:
  if e.code==409:break
  time.sleep(1)
 except Exception:time.sleep(1)
else:raise RuntimeError('S3 not ready')
# Upload a local image then move it to S3.
png=base64.b64decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=')
def upload():
 boundary='acceptance'+secrets.token_hex(8);body=('--'+boundary+'\r\nContent-Disposition: form-data; name="file"; filename="sample.png"\r\nContent-Type: image/png\r\n\r\n').encode()+png+('\r\n--'+boundary+'--\r\n').encode();req=urllib.request.Request(a.base.ADMIN+'/api/admin/materials',data=body,headers={'Content-Type':'multipart/form-data; boundary='+boundary});return json.load(c.opener.open(req))['data']
c.ok('/api/admin/storage','PUT',{'enabled':False})
local=upload();cfg={'enabled':True,'endpoint':'http://gravitylink-acceptance-s3:9000','region':'us-east-1','bucket':'images','prefix':'img','cdn':'http://127.0.0.1:29000/images','access_key':key,'secret_key':secret,'path_style':True}
c.ok('/api/admin/storage','PUT',cfg)
status=c.ok('/api/admin/storage');check('storage secret masked',status['secret_configured'] and 'secret_key' not in status)
# Config endpoints must never expose encrypted storage data.
configs=c.ok('/api/admin/configs');check('storage secret excluded from general config','storage.s3.encrypted' not in json.dumps(configs))
remote=c.ok(f"/api/admin/materials/{local['ID']}/migrate",'POST')
check('local image migrated to CDN URL',remote['Path']==cfg['cdn']+'/img/'+local['Name'])
check('object bytes verified by signed S3 GET',signed('GET','/images/img/'+local['Name'])==png)
check('repeat migration idempotent',c.ok(f"/api/admin/materials/{local['ID']}/migrate",'POST')['Path']==remote['Path'])
response=c.call(local['Path'],public=True);check('old image URL redirects',response[0]==302 and response[1]['Location']==remote['Path'])
new=upload();check('new upload uses object storage',new['Path'].startswith(cfg['cdn']) and signed('GET','/images/img/'+new['Name'])==png)
# Leave acceptance images reachable to the browser.
policy={'Version':'2012-10-17','Statement':[{'Effect':'Allow','Principal':{'AWS':['*']},'Action':['s3:GetObject'],'Resource':['arn:aws:s3:::images/*']}]}
signed('PUT','/images',json.dumps(policy).encode(),query='policy=')
check('CDN image is publicly readable',urllib.request.urlopen(remote['Path']).read()==png)
check('invalid storage settings rejected',c.call('/api/admin/storage','PUT',{'enabled':True})[0]==400)
fixture=a.WORK/'qr-review-fixture.json'
if fixture.exists():qr=json.loads(fixture.read_text(encoding='utf8'))
else:
 png=(a.WORK/'demo-qr-first.png').read_bytes();image=upload()
 domains=c.ok('/api/admin/domains')['items'];entry=next(d for d in domains if d['Type']=='entry');landing=next(d for d in domains if d['Type']=='landing')
 page=c.ok('/api/v1/landing-pages','POST',{'template':'liveqr','title':'活动群二维码','domain_id':landing['ID'],'content':{'headline':'加入活动群'}})
 qr=c.ok('/api/v1/links','POST',{'type':'liveqr','title':'活动群二维码','entry_domain_id':entry['ID'],'landing_domain_id':landing['ID'],'landing_page_id':page['ID'],'strategy':{'mode':'round_robin','targets':[{'label':'活动群','target_url':image['Path']}]}})
 fixture.write_text(json.dumps(qr),encoding='utf8')
check('group QR configuration loads',len(c.ok(f"/api/admin/links/{qr['ID']}/targets")['items'])==1)
(a.WORK/'refinement-results.json').write_text(json.dumps(checks,ensure_ascii=False,indent=2),encoding='utf8')
print('Completed',len(checks),'checks')
