"""Checks materials against the configured acceptance storage without changing credentials."""
import sys,json,urllib.request,time,secrets,base64
import e2e_acceptance as a
c=a.login();checks=[];storage=c.ok('/api/admin/storage')
def check(n,v):
 checks.append({'name':n,'passed':bool(v)});print(('PASS ' if v else 'FAIL ')+n,flush=True)
 if not v:raise AssertionError(n)
data=base64.b64decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j5WQAAAAASUVORK5CYII=')
boundary=secrets.token_hex(20);body=(f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="storage-review.png"\r\nContent-Type: image/png\r\n\r\n'.encode()+data+f'\r\n--{boundary}--\r\n'.encode())
r=c.opener.open(urllib.request.Request(a.base.ADMIN+'/api/admin/materials',data=body,headers={'Content-Type':'multipart/form-data; boundary='+boundary}),timeout=40)
item=json.loads(r.read())['data'];mid=item['ID']
check('ordinary image upload uses configured CDN',item['Path'].startswith(storage['cdn'].rstrip('/')+'/'+storage['prefix'].strip('/')+'/'))
check('uploaded bytes readable via CDN',urllib.request.urlopen(item['Path'],timeout=20).read()==data)
c.ok(f'/api/admin/materials/{mid}','PUT',{'name':'图片验收.png'})
check('material rename',next(i for i in c.ok('/api/admin/materials')['items'] if i['ID']==mid)['Name']=='图片验收.png')
name=item['Path'].rsplit('/',1)[1]
check('original upload address redirects after rename',c.call('/uploads/'+name,public=True)[0]==302)
c.ok('/api/admin/materials/remove','POST',{'ids':[mid]})
check('removed material excluded from library',all(i['ID']!=mid for i in c.ok('/api/admin/materials')['items']))
check('existing address preserved after removal',c.call('/uploads/'+name,public=True)[0]==302 and urllib.request.urlopen(item['Path'],timeout=20).read()==data)
check('anonymous material removal denied',a.base.Client().call('/api/admin/materials/remove','POST',{'ids':[mid]})[0]==401)
a.WORK.joinpath('material-management-results.json').write_text(json.dumps(checks,ensure_ascii=False,indent=2),encoding='utf-8')
