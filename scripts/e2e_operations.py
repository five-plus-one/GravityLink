"""Additive batch/landing workflow acceptance; never reset or delete data."""
import json,time
from e2e_local import Client,password,check,RESULTS,WORK

def run():
 c=Client();c.ok('/api/v1/auth/local/login','POST',{'username':'admin','password':password()})
 ds=c.ok('/api/admin/domains')['items']
 entry=next(d for d in ds if d['Host']=='localhost:18080' and d['Type']=='entry')
 landing=next(d for d in ds if d['Host']=='127.0.0.1:18080' and d['Type']=='landing')
 material=c.ok('/api/admin/materials')['items'][0]
 image='http://localhost:18080'+material['Path']
 stamp=str(time.time_ns())
 content={'headline':'操作验收原始标题','subtext':'说明','show_logo':False}
 page=c.ok('/api/v1/landing-pages','POST',{'template':'liveqr','title':'操作自动验收 '+stamp,'domain_id':landing['ID'],'content':content})
 qr=c.ok('/api/v1/links','POST',{'type':'liveqr','entry_domain_id':entry['ID'],'landing_domain_id':landing['ID'],'landing_page_id':page['ID'],'title':'批量操作验收 '+stamp,'strategy':{'mode':'round_robin','targets':[{'target_url':image,'scan_limit':1}]}})
 base=f"/api/admin/links/{qr['ID']}/targets"
 before=c.ok(base)['items']
 check('batch requires login',Client().call(base+'/batch','POST',{'urls':[image]})[0]==401)
 for name,payload in [('invalid URL',{'urls':[image,'javascript:x']}),('duplicate',{'urls':[image,image]}),('empty',{'urls':[]}),('over limit',{'urls':[image]*101}),('zero threshold',{'urls':[image],'scan_limit':0})]:
  check('reject '+name,c.call(base+'/batch','POST',payload)[0]==400)
 check('invalid batches leave no partial records',len(c.ok(base)['items'])==len(before))
 added=c.ok(base+'/batch','POST',{'urls':[image+'?batch=1',image+'?batch=2'],'scan_limit':2})
 check('batch creates all targets',added['total']==2 and all(t['ScanLimit']==2 and t['ScanCount']==0 for t in added['items']))
 target=c.ok(base)['items'][0]
 reset=f"{base}/{target['ID']}/reset-count"
 reset_payload={'expected_count':target['ScanCount'],'confirmation':'RESET TARGET COUNT'}
 check('active target cannot reset',c.call(reset,'POST',reset_payload)[0]==409)
 c.call('/'+qr['Code'],host=landing['Host'],public=True)
 target=c.ok(base)['items'][0];target['Status']='disabled';c.ok(base,'POST',target)
 check('stale counter reset rejected',c.call(reset,'POST',reset_payload)[0]==409)
 check('reset confirmation required',c.call(reset,'POST',{'expected_count':target['ScanCount']})[0]==400)
 c.ok(reset,'POST',{'expected_count':target['ScanCount'],'confirmation':'RESET TARGET COUNT'})
 after=next(t for t in c.ok(base)['items'] if t['ID']==target['ID'])
 check('reset clears count and keeps disabled',after['ScanCount']==0 and after['Status']=='disabled')
 before=c.ok(base)['items']
 preview=c.ok(f"/api/v1/landing-pages/{page['ID']}/preview")['html']
 check('preview contains selected layout','操作验收原始标题' in preview)
 check('preview does not allocate QR or count visits',c.ok(base)['items']==before)
 payload={'template':'liveqr','title':page['Title'],'domain_id':landing['ID'],'content':{**content,'headline':'操作验收更新标题'}}
 updated=c.ok(f"/api/v1/landing-pages/{page['ID']}",'PUT',payload)
 check('edit keeps page identity and content',updated['ID']==page['ID'] and updated['Content']['headline']=='操作验收更新标题' and updated['Content']['show_logo']==False)
 check('preview reflects edit','操作验收更新标题' in c.ok(f"/api/v1/landing-pages/{page['ID']}/preview")['html'])
 check('bound domain change rejected',c.call(f"/api/v1/landing-pages/{page['ID']}",'PUT',{**payload,'domain_id':entry['ID']})[0]==400)
 check('anonymous landing edit denied',Client().call(f"/api/v1/landing-pages/{page['ID']}",'PUT',payload)[0]==401)
 # Cross a Base62 alphabet boundary and the former 100-row list cutoff.
 original=c.ok('/api/v1/links')['items']
 created=[c.ok('/api/v1/links','POST',{'type':'short','entry_domain_id':entry['ID'],'target_url':'https://example.com/batch-acceptance','title':'连续创建验收 '+stamp+' '+str(i)}) for i in range(64)]
 check('64 consecutive automatic codes are unique',len({r['Code'].lower() for r in created})==64)
 listed=c.ok('/api/v1/links')['items']
 check('listing does not silently truncate at 100',len(listed)==len(original)+64 and {r['ID'] for r in original}.issubset({r['ID'] for r in listed}))
 WORK.joinpath('operations-e2e.json').write_text(json.dumps({'results':RESULTS,'qr_id':qr['ID'],'page_id':page['ID']},ensure_ascii=False,indent=2),encoding='utf8')
 if not all(r['passed'] for r in RESULTS):raise SystemExit(1)

if __name__=='__main__':run()
