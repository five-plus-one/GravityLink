"""Fresh initialization acceptance in an isolated temporary app and new database.
Leaves the test database for review; never resets the main application.
"""
import json
import subprocess
import time
import e2e_local as core

def run():
    stamp=str(int(time.time()))
    database='gravitylink_acceptance_'+stamp
    name='gravitylink-acceptance-'+stamp
    sql=f"CREATE DATABASE `{database}` CHARACTER SET utf8mb4; GRANT ALL ON `{database}`.* TO 'gravitylink'@'%';"
    p=subprocess.run(['docker','exec','-i','gravitylink-mysql-1','sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot'],input=sql,text=True,capture_output=True)
    if p.returncode:raise RuntimeError('Cannot create isolated acceptance database')
    p=subprocess.run(['docker','run','--rm','-d','--name',name,'--network','gravitylink_default','-p','127.0.0.1:18082:8080','-p','127.0.0.1:18083:8081','--mount','type=volume,destination=/data','-e','CONFIG_FILE=/data/gravitylink.json','gravitylink-gravitylink'],capture_output=True)
    if p.returncode:raise RuntimeError('Cannot start isolated acceptance app')
    try:
        core.ADMIN='http://127.0.0.1:18083';core.PUBLIC='http://127.0.0.1:18082'
        c=core.Client()
        for _ in range(30):
            try:
                state=c.ok('/api/setup/status');break
            except Exception:time.sleep(1)
        else:raise RuntimeError('Acceptance app did not start')
        core.check('fresh setup required',state['setup_required'])
        c.ok('/api/setup/auth/local','POST',{'auth':{'username':'admin','password':core.password(),'admin_base_url':core.ADMIN}})
        config={'database':{'host':'mysql','port':'3306','database':database,'user':'gravitylink','password':core.ENV['MYSQL_PASSWORD']},'redis':{'host':'redis','port':'6379','db':14}}
        tested=c.ok('/api/setup/database/test','POST',config)
        core.check('fresh database and Redis connectivity',tested['mysql'] and tested['redis'])
        c.ok('/api/setup/complete','POST',config)
        core.check('fresh schema initialized',not c.ok('/api/setup/status')['setup_required'])
        core.check('repeat initialization blocked',c.call('/api/setup/complete','POST',config)[0]==409)
        import e2e_content
        e2e_content.run()
        # Only the disposable acceptance app loses its network, never production.
        subprocess.run(['docker','network','disconnect','gravitylink_default',name],check=True,capture_output=True)
        try:
            subprocess.run(['docker','restart',name],check=True,capture_output=True)
            failed_start=False
            for _ in range(30):
                logs=subprocess.run(['docker','logs','--tail','20',name],capture_output=True,text=True)
                if 'normal runtime unavailable' in logs.stdout+logs.stderr:
                    failed_start=True;break
                time.sleep(1)
            core.check('isolated startup dependency failure reproduced',failed_start)
        finally:
            subprocess.run(['docker','network','connect','gravitylink_default',name],check=True,capture_output=True)
        recovered=False
        for _ in range(40):
            try:
                if not c.ok('/api/setup/status')['setup_required']:
                    recovered=True;break
            except Exception:pass
            time.sleep(1)
        core.check('startup recovers without reinitialization',recovered)
        c.ok('/api/v1/auth/local/login','POST',{'username':'admin','password':core.password()})
        core.check('recovered app retains business data',len(c.ok('/api/v1/links')['items'])>=3)
        core.WORK.joinpath('initialization-e2e.json').write_text(json.dumps({'database':database,'results':core.RESULTS},ensure_ascii=False,indent=2),encoding='utf-8')
    finally:
        subprocess.run(['docker','stop',name],capture_output=True)

if __name__=='__main__':run()
