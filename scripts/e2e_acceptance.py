"""Isolated Docker acceptance. Uses only gravitylink-acceptance containers and ports."""
import concurrent.futures
import datetime as dt
import json
import secrets
import subprocess
import time
from pathlib import Path
from urllib.parse import urlencode
import e2e_local as base
base.ADMIN='http://127.0.0.1:28081'
base.PUBLIC='http://127.0.0.1:28080'
WORK=base.WORK
RESULTS=[]
def check(name,value):
    RESULTS.append({'name':name,'passed':bool(value)})
    print(('PASS ' if value else 'FAIL ')+name,flush=True)
    if not value: raise AssertionError(name)
def sql(query):
    r=subprocess.run(['docker','exec','-i','gravitylink-acceptance-mysql-1','sh','-c','MYSQL_PWD="$MYSQL_PASSWORD" mysql -ugravitylink -N -B gravitylink'],input=query,text=True,capture_output=True,encoding='utf-8')
    if r.returncode: raise RuntimeError('Acceptance SQL failed')
    return r.stdout.strip()
def login():
    path=WORK/'acceptance-admin.dpapi'
    if not path.exists():path.write_bytes(base.protect(secrets.token_urlsafe(24).encode()))
    pwd=base.protect(path.read_bytes(),True).decode()
    c=base.Client()
    if c.ok('/api/setup/status')['setup_required']:
        c.ok('/api/setup/auth/local','POST',{'auth':{'username':'reviewer','password':pwd,'admin_base_url':base.ADMIN}})
        config={'database':{'host':'mysql','port':'3306','database':'gravitylink','user':'gravitylink','password':base.ENV['MYSQL_PASSWORD']},'redis':{'host':'redis','port':'6379'}}
        c.ok('/api/setup/database/test','POST',config)
        c.ok('/api/setup/complete','POST',config)
    c.ok('/api/v1/auth/local/login','POST',{'username':'reviewer','password':pwd})
    return c

def run():
    c=login();anon=base.Client();stamp=str(int(time.time()))
    check('anonymous administration denied',anon.call('/api/v1/links')[0]==401)
    domains=c.ok('/api/admin/domains')['items']
    def domain(host,kind):
        return next((d for d in domains if d['Host']==host),None) or c.ok('/api/admin/domains','POST',{'host':host,'type':kind,'scheme':'http'})
    entry=domain('127.0.0.1:28080','entry');landing=domain('localhost:28080','landing')
    link=c.ok('/api/v1/links','POST',{'type':'short','title':'秋季活动 · '+stamp,'entry_domain_id':entry['ID'],'target_url':'https://example.com','code':'v'+stamp})
    lid=link['ID']
    for _ in range(65):check_visit=anon.call('/'+link['Code'],public=True)[0]
    check('public redirect',check_visit==302)
    for _ in range(30):
        records=c.ok('/api/v1/stats/visitors?'+urlencode({'link_id':lid,'limit':50}))
        if records['total']>=65:break
        time.sleep(1)
    check('persisted visitor total',records['total']==65)
    page2=c.ok('/api/v1/stats/visitors?'+urlencode({'link_id':lid,'limit':50,'offset':50}))
    check('second visitor page',len(records['items'])==50 and len(page2['items'])==15 and not ({r['id'] for r in records['items']}&{r['id'] for r in page2['items']}))
    check('visitor timestamps include timezone',records['items'][0]['visited_at'].endswith('+08:00'))
    today=dt.datetime.now().strftime('%Y-%m-%d')
    sql(f"INSERT INTO access_logs(link_id,ip,device,os,browser,visited_at) VALUES ({lid},'203.0.113.20','desktop','Windows','Edge','{today} 00:30:00'),({lid},'203.0.113.21','tablet','iPadOS','Safari','{today} 02:30:00');")
    narrow=c.ok('/api/v1/stats/visitors?'+urlencode({'link_id':lid,'start':today+'T00:00:00+08:00','end':today+'T01:00:00+08:00'}))
    check('precise local time range',narrow['total']==1 and narrow['items'][0]['ip']=='203.0.113.20')
    keyword=c.ok('/api/v1/stats/visitors?'+urlencode({'keyword':link['Code']}))
    check('keyword search',keyword['total']==67)
    check('invalid time rejected',c.call('/api/v1/stats/visitors?start=invalid')[0]==400)
    empty=c.ok('/api/v1/stats/visitors?keyword=absent-'+stamp)
    check('empty records are an array',empty=={'items':[],'total':0})
    dev=c.ok(f'/api/v1/stats/{lid}/device?start={today}&end={today}')
    check('today devices and browsers',sum(x['value'] for x in dev['device'])==67 and len(dev['device'])==3)
    overview=c.ok(f'/api/v1/stats/overview/device?start={today}&end={today}')
    check('all-link device distribution',sum(x['value'] for x in overview['device'])>=67)
    hours=c.ok(f'/api/v1/stats/{lid}/hourly?start={today}&end={today}')
    check('hourly totals match visitors',sum(x['pv'] for x in hours)==67 and hours[0]['pv']==1)
    geo=c.ok(f'/api/v1/stats/overview/geo?start={today}&end={today}')
    check('geography includes unknown locations',sum(x['value'] for x in geo)>=67)
    yesterday=(dt.datetime.now()-dt.timedelta(days=1)).strftime('%Y-%m-%d')
    yesterday_key=yesterday.replace('-','')
    commands=f"MULTI\nSET stat:pv:{lid}:{yesterday_key} 7\nPFADD stat:uv:{lid}:{yesterday_key} visitor-a visitor-b\nSET stat:hourly:{lid}:{yesterday_key}:9 7\nHSET stat:dev:{lid}:{yesterday_key} mobile|iOS|Safari 7\nHSET stat:geo:{lid}:{yesterday_key} 中国|上海 7\nEXEC\n"
    subprocess.run(['docker','exec','-i','gravitylink-acceptance-redis-1','redis-cli'],input=commands,text=True,encoding='utf-8',capture_output=True,check=True)
    for _ in range(75):
        points=c.ok(f'/api/v1/stats/{lid}/daily?start={yesterday}&end={yesterday}')
        if points and points[0]['pv']==7:break
        time.sleep(1)
    check('scheduled worker persists historical visits',bool(points) and points[0]['pv']==7 and points[0]['uv']==2)
    historical=c.ok(f'/api/v1/stats/{lid}/device?start={yesterday}&end={today}')
    check('historical and today device totals combine once',sum(x['value'] for x in historical['device'])==74)
    historical_hours=c.ok(f'/api/v1/stats/{lid}/hourly?start={yesterday}&end={yesterday}')
    check('historical hourly selection',sum(x['pv'] for x in historical_hours)==7 and historical_hours[9]['pv']==7)
    historical_geo=c.ok(f'/api/v1/stats/{lid}/geo?start={yesterday}&end={yesterday}')
    check('historical geography from scheduled worker',historical_geo==[{'label':'上海','value':7}])
    old=(dt.datetime.now()-dt.timedelta(days=120)).strftime('%Y-%m-%d')
    sql(f"INSERT INTO access_logs_archive(id,link_id,ip,device,visited_at) VALUES (9000000+{lid},{lid},'203.0.113.99','desktop','{old} 10:00:00');")
    history=c.ok('/api/v1/stats/visitors?'+urlencode({'link_id':lid,'start':old,'end':old}))
    check('archived visitor records searchable',history['total']==1 and history['items'][0]['ip']=='203.0.113.99')
    check('invalid statistics range rejected',c.call('/api/v1/stats/overview/device?start=2026-09-13&end=2026-09-01')[0]==400)
    project=c.ok('/api/admin/kami/projects','POST',{'title':'会员兑换 · '+stamp,'type':'兑换码','password':'claim-'+stamp,'repeat_policy':'never'})
    pid=project['ID'];prefix=f'/api/admin/kami/projects/{pid}'
    codes=[{'content':f'DEMO-{stamp}-{n:03}'} for n in range(65)]
    imported=c.ok(prefix+'/items/import','POST',{'items':codes+[codes[0]]})
    check('import deduplicates within batch',imported['imported']==65 and imported['dup']==1)
    again=c.ok(prefix+'/items/import','POST',{'items':codes[:2]})
    check('import deduplicates existing stock',again['imported']==0 and again['dup']==2)
    stock=c.ok(prefix+'/items?limit=50&offset=50')
    check('stock server pagination',stock['total']==65 and len(stock['items'])==15)
    issue=f'/api/v1/kami/{pid}/issue'
    check('claim requires password',anon.call(issue,'POST',{},public=True)[0]==400)
    check('wrong claim password denied',anon.call(issue,'POST',{'password':'wrong'},public=True)[0]==403)
    def claim(_):return base.Client().call(issue,'POST',{'password':'claim-'+stamp},public=True)[0]
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:statuses=list(pool.map(claim,range(8)))
    check('concurrent same-visitor claims issue once',statuses.count(200)==1 and statuses.count(403)==7)
    issuance=c.ok(prefix+'/issuances')
    check('issuance appears with device',issuance['total']==1 and issuance['items'][0]['VisitorDevice']=='mobile')
    page=c.ok('/api/v1/landing-pages','POST',{'template':'kami','title':project['Title'],'domain_id':landing['ID'],'content':{'project_id':pid,'button_text':'立即领取'}})
    claimlink=c.ok('/api/v1/links','POST',{'type':'liveqr','title':project['Title'],'entry_domain_id':entry['ID'],'landing_domain_id':landing['ID'],'landing_page_id':page['ID'],'target_url':''})
    redirect=anon.call('/'+claimlink['Code'],public=True)
    check('claim link resolves to landing',redirect[0]==302 and 'localhost:28080' in redirect[1]['Location'])
    html=anon.call('/'+claimlink['Code'],host='localhost:28080',public=True)
    check('landing renders claim project',html[0]==200 and f'data-project-id="{pid}"' in html[2])
    (WORK/'acceptance-fixture.json').write_text(json.dumps({'link_id':lid,'code':link['Code'],'project_id':pid,'claim_url':redirect[1]['Location']},ensure_ascii=False),encoding='utf-8')
    (WORK/'acceptance-results.json').write_text(json.dumps(RESULTS,ensure_ascii=False,indent=2),encoding='utf-8')
if __name__=='__main__':run()
