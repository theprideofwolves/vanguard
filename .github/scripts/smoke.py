import html,json,os,re,subprocess,sys,tempfile,threading,queue,urllib.request,urllib.parse
from pathlib import Path
executable=sys.argv[1]
def multipart(fields):
 boundary='onboarding-smoke-boundary';data=b''
 for name,value in fields.items():data+=(f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"\r\n\r\n{value}\r\n').encode()
 return data+f'--{boundary}--\r\n'.encode(),'multipart/form-data; boundary='+boundary
def launch(env):
 p=subprocess.Popen([str(Path(executable).resolve())],env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,encoding='utf-8');q=queue.Queue();threading.Thread(target=lambda:q.put(p.stdout.readline()),daemon=True).start()
 try:line=q.get(timeout=30)
 except queue.Empty:p.kill();p.wait();raise RuntimeError('no launcher URL')
 match=re.search(r'http://\S+',line)
 if not match:p.kill();p.wait();raise RuntimeError('launcher failed: '+line)
 return p,match.group(0)
def get(url):return urllib.request.urlopen(url,timeout=30).read().decode()
def post(url,fields,multi=False):
 if multi:data,ct=multipart(fields)
 else:data,ct=urllib.parse.urlencode(fields).encode(),'application/x-www-form-urlencoded'
 return urllib.request.urlopen(urllib.request.Request(url,data=data,headers={'Content-Type':ct}),timeout=30).read().decode()
with tempfile.TemporaryDirectory(prefix='vanguard-',dir=os.environ.get('VANGUARD_SMOKE_TEMP')) as root:
 env=dict(os.environ,VANGUARD_DATA_DIR=root,VANGUARD_NO_BROWSER='1')
 p,url=launch(env)
 try:
  page=get(url);assert 'Try the guided demo' in page and 'Use my data' in page and 'ZgotmplZ' not in page
  token=urllib.parse.parse_qs(urllib.parse.urlparse(url).query)['token'][0]
  def route(path):return urllib.parse.urljoin(url,path)+'?'+urllib.parse.urlencode({'token':token})
  setup=get(route('/setup'));assert 'Set up your first task' in setup
  assert not list(Path(root).glob('*/onboarding.json')), 'viewing setup marked guide complete'
  report=get(route('/onboarding/sample')+'&name=report.json')
  policy=get(route('/onboarding/sample')+'&name=policy.yaml')
  assert json.loads(report)['findings'] and 'apiVersion: vanguard/v1' in policy
  fields={'policy':policy,'report':report}
  page=post(url+'&onboarding=demo',fields,True)
  assert 'id="results"' in page and 'href="#results"' in page
  assert 'Your first result is ready' in page and 'Guided demo - synthetic data' in page and 'ZgotmplZ' not in page
  assert '<h2>FAIL</h2>' in page
  link=re.search(r'href="(/export[^\"]+)"',page);assert link,'missing export'
  payload=get(urllib.parse.urljoin(url,html.unescape(link.group(1))));value=json.loads(payload)
  assert value['Passed'] is False and value['PolicyName']=='example-baseline'
  critical=next(row for row in value['Rows'] if row['Severity']=='critical')
  assert critical['Observed']==1 and critical['Allowed']==0 and critical['Exceeded']
  assert list(Path(root).glob('*/onboarding.json')),'completion not remembered'
  assert 'Try the guided demo' in get(route('/welcome'))
  assert json.loads(get(urllib.parse.urljoin(url,html.unescape(link.group(1)))))==value,'restart guide modified results'
  try:get(url.split('?')[0]);raise AssertionError('missing token allowed')
  except urllib.error.HTTPError as e:assert e.code==403
 finally:p.terminate();p.wait(timeout=15)
 for lock in Path(root).glob('*/application.lock'):lock.unlink()
 p,url=launch(env)
 try:
  page=get(url);assert 'Try the guided demo' not in page and 'Welcome back' in page
  assert 'Session results are not saved' in page and '<h2>FAIL</h2>' not in page
  token=urllib.parse.parse_qs(urllib.parse.urlparse(url).query)['token'][0]
  try:get(route('/export'));raise AssertionError('session decision persisted')
  except urllib.error.HTTPError as e:assert e.code==400
  page=post(url,{'policy':policy,'report':'{"findings":[]}'},True)
  assert '<h2>PASS</h2>' in page
  assert json.loads(get(route('/export')))['Passed'] is True
  page=post(url,{'policy':policy,'report':'{"findings":[{"severity":"urgent"}]}'},True)
  assert 'unsupported severity' in page and '<h2>PASS</h2>' not in page
  try:get(route('/export'));raise AssertionError('invalid evaluation exposed stale decision')
  except urllib.error.HTTPError as e:assert e.code==400
 finally:p.terminate();p.wait(timeout=15)
 cli_policy=Path(root,'policy.yaml');cli_policy.write_text(policy,encoding='utf-8')
 cli_env=dict(env,VANGUARD_SLACK_WEBHOOK='')
 for report_input,expected in [('{"findings":[]}',0),(report,1),('{"findings":[{"severity":"critical"}],"Findings":[]}',2)]:
  args=[str(Path(executable).resolve()),'-p',str(cli_policy),'-json','-']
  run=subprocess.run(args,input=report_input,env=cli_env,text=True,capture_output=True,timeout=30)
  assert run.returncode==expected,(run.returncode,expected,run.stderr)
  if expected<2:assert json.loads(run.stdout)['Passed']==(expected==0)
  else:assert not run.stdout and 'duplicate JSON object key' in run.stderr
 run=subprocess.run([str(Path(executable).resolve()),'-p',str(cli_policy),'-json','-notify','-'],input=report,env=cli_env,text=True,capture_output=True,timeout=30)
 assert run.returncode==1 and json.loads(run.stdout)['Passed'] is False and 'invalid Slack' in run.stderr
 print('Vanguard: packaged onboarding, FAIL/PASS decisions, export, remembered guide, session isolation and invalid-input stale-result protection PASS')
