import hashlib,html,http.cookiejar,json,os,re,subprocess,sys,tempfile,threading,queue,urllib.request,urllib.parse
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
  urllib.request.install_opener(urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar())))
  page=get(url);assert 'Try the guided demo' in page and 'Use my data' in page and 'ZgotmplZ' not in page
  assert 'name="candidate"' in page and 'proposed-baseline' in page, 'demo is missing proposed policy'
  assert 'data-theme="light"' in page and 'Dark mode' in page
  token=urllib.parse.parse_qs(urllib.parse.urlparse(url).query)['token'][0]
  def route(path):return urllib.parse.urljoin(url,path)+'?'+urllib.parse.urlencode({'token':token})
  setup=get(route('/setup'));assert 'Set up your first task' in setup
  themed=post(route('/theme'),{'theme':'dark','return':'/setup'})
  assert 'data-theme="dark"' in themed and 'Light mode' in themed and 'Set up your first task' in themed
  assert 'data-theme="dark"' in get(url)
  assert 'data-theme="light"' in post(route('/theme'),{'theme':'light','return':'/setup'})
  assert not list(Path(root).glob('*/onboarding.json')), 'viewing setup marked guide complete'
  report=get(route('/onboarding/sample')+'&name=report.json')
  policy=get(route('/onboarding/sample')+'&name=policy.yaml')
  candidate=get(route('/onboarding/sample')+'&name=proposed-policy.yaml')
  assert json.loads(report)['findings'] and 'apiVersion: vanguard/v1' in policy
  fields={'policy':policy,'report':report,'candidate':candidate}
  page=post(url,{'policy':policy,'report':'null'},True)
  assert 'id="results"' not in page
  assert not list(Path(root).glob('*/onboarding.json')), 'invalid input marked guide complete'
  page=post(url+'&onboarding=demo',fields,True)
  assert 'id="results"' in page and 'href="#results"' in page
  assert 'Your first result is ready' in page and 'Guided demo - synthetic data' in page and 'ZgotmplZ' not in page
  assert '<h2>FAIL</h2>' in page
  link=re.search(r'href="(/export[^\"]+)"',page);assert link,'missing export'
  payload=get(urllib.parse.urljoin(url,html.unescape(link.group(1))));value=json.loads(payload)
  assert value['Passed'] is False and value['PolicyName']=='example-baseline'
  assert value['InputHash']==hashlib.sha256(report.encode()).hexdigest()
  assert value['PolicyHash']==hashlib.sha256(policy.encode()).hexdigest()
  assert value['Violations']==['critical: found 1 (max allowed 0)']
  critical=next(row for row in value['Rows'] if row['Severity']=='critical')
  assert critical['Observed']==1 and critical['Allowed']==0 and critical['Exceeded']
  assert list(Path(root).glob('*/onboarding.json')),'completion not remembered'
  assert 'data-theme="dark"' in post(route('/theme'),{'theme':'dark','return':'/'})
  assert json.loads(get(route('/export')))==value,'theme changed decision'
  assert 'Try the guided demo' in get(route('/welcome'))
  assert json.loads(get(urllib.parse.urljoin(url,html.unescape(link.group(1)))))==value,'restart guide modified results'
  page=post(url,dict(fields,candidate=candidate,action='compare'),True)
  assert 'REVIEW REQUIRED' in page and 'Newly passing: 1' in page
  comparison=json.loads(get(route('/export')))
  assert comparison['Version']=='vanguard-impact/v1' and comparison['ReviewRequired']
  assert comparison['Reports'][0]['Base']['InputHash']==value['InputHash']==comparison['Reports'][0]['Candidate']['InputHash']
  assert comparison['BasePolicyHash']==value['PolicyHash']
  assert comparison['CandidatePolicyHash']==hashlib.sha256(candidate.encode()).hexdigest()
  assert 'data-theme="light"' in post(route('/theme'),{'theme':'light','return':'/'})
  assert comparison==json.loads(get(route('/export'))),'theme changed impact evidence'
  assert 'Try the guided demo' in get(route('/welcome'))
  assert comparison==json.loads(get(route('/export'))),'guide changed impact evidence'
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
  broken=urllib.request.Request(url,data=b'--broken\r\ninvalid',headers={'Content-Type':'multipart/form-data; boundary=broken'})
  try:urllib.request.urlopen(broken,timeout=30);raise AssertionError('malformed upload accepted')
  except urllib.error.HTTPError as e:assert e.code==400
  try:get(route('/export'));raise AssertionError('malformed upload exposed stale decision')
  except urllib.error.HTTPError as e:assert e.code==400
  page=post(url,{'policy':policy,'report':'{"findings":[]}'},True)
  assert '<h2>PASS</h2>' in page
  page=post(url,{'policy':policy,'report':'{"findings":[{"severity":"urgent"}]}'},True)
  assert 'unsupported severity' in page and '<h2>PASS</h2>' not in page
  try:get(route('/export'));raise AssertionError('invalid evaluation exposed stale decision')
  except urllib.error.HTTPError as e:assert e.code==400
  page=post(url,{'policy':policy,'candidate':candidate,'report':report,'action':'compare'},True)
  assert 'REVIEW REQUIRED' in page
  page=post(url,{'policy':policy,'candidate':'null','report':report,'action':'compare'},True)
  try:get(route('/export'));raise AssertionError('invalid comparison exposed stale evidence')
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
 # Exercise release-bundled fixtures through both file and explicit stdin intake.
 repo=Path(__file__).resolve().parents[2]
 for name,expected in [('report-pass.json',0),('report-fail.json',1)]:
  fixture=repo/'testdata'/name
  args=[str(Path(executable).resolve()),'-p',str(repo/'policy.yaml'),'-json']
  for source in [str(fixture),'-']:
   run=subprocess.run(args+[source],input=fixture.read_bytes() if source=='-' else None,env=cli_env,capture_output=True,timeout=30)
   assert run.returncode==expected,(name,source,run.returncode,run.stderr)
   value=json.loads(run.stdout)
   assert value['Passed']==(expected==0)
   assert value['InputHash']==hashlib.sha256(fixture.read_bytes()).hexdigest()
   assert value['PolicyHash']==hashlib.sha256((repo/'policy.yaml').read_bytes()).hexdigest()
 # Real CLI policy review: mixed changes preserve aggregate passes but flip members.
 impact_root=repo/'testdata'/'impact'
 base_args=[str(Path(executable).resolve()),'compare','-base',str(impact_root/'base.yaml'),'-candidate']
 for candidate_name,expected in [('candidate.yaml',1),('tightened.yaml',0),('base.yaml',0)]:
  run=subprocess.run(base_args+[str(impact_root/candidate_name),'-json',str(impact_root/'reports')],env=cli_env,capture_output=True,timeout=30)
  assert run.returncode==expected,(candidate_name,run.returncode,run.stderr)
  value=json.loads(run.stdout)
  assert value['ReviewRequired']==(expected==1) and value['Summary']['Reports']==4
  assert value['BasePolicyHash']==hashlib.sha256((impact_root/'base.yaml').read_bytes()).hexdigest()
  assert value['CandidatePolicyHash']==hashlib.sha256((impact_root/candidate_name).read_bytes()).hexdigest()
  for row in value['Reports']:
   digest=hashlib.sha256((impact_root/'reports'/row['Name']).read_bytes()).hexdigest()
   assert row['Base']['InputHash']==digest==row['Candidate']['InputHash']
  if candidate_name=='candidate.yaml':
   assert value['Summary']==dict(Reports=4,BasePassed=2,CandidatePassed=2,NewlyPassing=1,NewlyFailing=1,UnchangedPassed=1,UnchangedFailed=1)
   assert [r['Transition'] for r in value['Reports']]==['pass_to_pass','fail_to_pass','pass_to_fail','fail_to_fail']
 bad=Path(root,'bad.json');bad.write_text('null',encoding='utf-8')
 run=subprocess.run(base_args+[str(impact_root/'candidate.yaml'),'-json',str(bad)],env=cli_env,capture_output=True,timeout=30)
 assert run.returncode==2 and not run.stdout and run.stderr
 print('Vanguard: packaged evaluation, policy impact, themes, export, guide/session isolation and CLI 0/1/2 PASS')
