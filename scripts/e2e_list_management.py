"""Local acceptance for link labels and Kami settings; fixtures confined to acceptance."""
import json,time,secrets
import e2e_acceptance as a
c=a.login();checks=[]
def check(name,value):
 checks.append({'name':name,'passed':bool(value)});print(('PASS ' if value else 'FAIL ')+name,flush=True)
 if not value:raise AssertionError(name)
stamp=str(int(time.time()));entry=next(d for d in c.ok('/api/admin/domains')['items'] if d['Host']=='127.0.0.1:28080')
links=[c.ok('/api/v1/links','POST',{'type':'short','title':'列表验收 '+stamp+' '+str(i),'entry_domain_id':entry['ID'],'target_url':'https://example.com'}) for i in range(2)]
ids=[l['ID'] for l in links];cat='分组'+stamp;tag='标签'+stamp
mut=lambda **p:c.ok('/api/v1/link-labels','POST',p)
mut(kind='category',action='add',names=[cat]);check('empty category persists',cat in c.ok('/api/v1/link-labels')['categories'])
c.ok('/api/v1/links/bulk','POST',{'ids':ids,'category':cat,'tag_mode':'append','tags':[tag,tag]})
rows=lambda:[l for l in c.ok('/api/v1/links')['items'] if l['ID'] in ids]
check('bulk category and deduplicated tags',all(l['Category']==cat and l['Tags']==[tag] for l in rows()))
mut(kind='category',action='move',names=[cat],target=cat+'新');check('category transfer',all(l['Category']==cat+'新' for l in rows()))
mut(kind='tag',action='move',names=[tag],target=tag+'新');check('tag transfer',all(l['Tags']==[tag+'新'] for l in rows()))
c.ok('/api/v1/links/bulk','POST',{'ids':ids,'status':'disabled'});check('bulk disable',all(l['Status']=='disabled' for l in rows()))
check('atomic missing ID rejection',c.call('/api/v1/links/bulk','POST',{'ids':[ids[0],999999999],'category':'错误'})[0]==400 and rows()[0]['Category']==cat+'新')
mut(kind='category',action='remove',names=[cat+'新']);mut(kind='tag',action='remove',names=[tag+'新']);check('delete associations preserves links',len(rows())==2 and all(not l['Category'] and not l['Tags'] for l in rows()))
p=c.ok('/api/admin/kami/projects','POST',{'title':'口令验收 '+stamp,'password':'old-'+stamp,'repeat_policy':'allow','repeat_interval_sec':0});pid=p['ID']
check('password omitted from response','Password' not in p and 'password' not in p)
c.ok(f'/api/admin/kami/projects/{pid}/items/import','POST',{'items':[{'content':secrets.token_hex(8)} for _ in range(3)]})
c.ok(f'/api/admin/kami/projects/{pid}','PUT',{'password':'new-'+stamp})
anon=a.base.Client();issue=lambda password:anon.call(f'/api/v1/kami/{pid}/issue','POST',{'password':password},public=True)
check('old password rejected',issue('old-'+stamp)[0]!=200);check('new password accepted',issue('new-'+stamp)[0]==200)
c.ok(f'/api/admin/kami/projects/{pid}','PUT',{'password':''});check('cleared password accepts empty',issue('')[0]==200)
check('password status updated',not next(p for p in c.ok('/api/admin/kami/projects')['items'] if p['id']==pid)['password_configured'])
# Remove only this run's test fixtures through the application API.
for lid in ids:c.ok(f'/api/v1/links/{lid}','DELETE')
c.ok(f'/api/admin/kami/projects/{pid}','DELETE')
a.WORK.joinpath('list-management-results.json').write_text(json.dumps(checks,ensure_ascii=False,indent=2),encoding='utf-8')
