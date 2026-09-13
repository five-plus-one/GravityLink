"""Acceptance deletion uses newly created fixtures only."""
import sys,json,base64,secrets,urllib.request,subprocess
import e2e_acceptance as a
c=a.login();data=base64.b64decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j5WQAAAAASUVORK5CYII=')
boundary=secrets.token_hex(12);body=f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="delete-check.png"\r\nContent-Type: image/png\r\n\r\n'.encode()+data+f'\r\n--{boundary}--\r\n'.encode()
req=urllib.request.Request(a.base.ADMIN+'/api/admin/materials',data=body,headers={'Content-Type':'multipart/form-data; boundary='+boundary})
item=json.loads(c.opener.open(req,timeout=40).read())['data']
name='delete-'+secrets.token_hex(12)+'.png'
r=subprocess.run(['docker','exec','-i','gravitylink-acceptance-gravitylink-1','sh','-c','cat > /data/uploads/'+name],input=data,capture_output=True);assert r.returncode==0
mid=int(a.sql(f"INSERT INTO materials(name,path,created_at) VALUES ('{name}','/uploads/{name}',NOW());SELECT LAST_INSERT_ID();"))
result=c.ok('/api/admin/materials/delete','POST',{'ids':[item['ID'],mid,99999999]})['items']
assert [i['deleted'] for i in result]==[True,True,False]
assert all(i['ID'] not in [item['ID'],mid] for i in c.ok('/api/admin/materials')['items'])
assert c.call('/uploads/'+name,public=True)[0]==404
assert subprocess.run(['docker','exec','gravitylink-acceptance-gravitylink-1','test','!','-e','/data/uploads/'+name]).returncode==0
assert a.base.Client().call('/api/admin/materials/delete','POST',{'ids':[mid]})[0]==401
print('PASS real OSS delete, local file delete, record removal, per-item failure, anonymous denial')
a.WORK.joinpath('material-deletion-results.json').write_text(json.dumps(result,ensure_ascii=False,indent=2),encoding='utf-8')
