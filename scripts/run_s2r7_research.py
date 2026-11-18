#!/usr/bin/env python3
"""Run one bounded isolated R7 phase; preserve every invocation and failure."""
import argparse,hashlib,json,os,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def sha(path):
 h=hashlib.sha256()
 with open(path,'rb') as f:
  for b in iter(lambda:f.read(1<<20),b''):h.update(b)
 return h.hexdigest()
def main():
 p=argparse.ArgumentParser();p.add_argument('--source',type=Path,required=True);p.add_argument('--work',type=Path,required=True);p.add_argument('--binary',type=Path,required=True);p.add_argument('--phase',choices=['prepare','information','train','iid','evaluate','persistence'],required=True);p.add_argument('--stage',choices=['development','confirmatory'],default='development');p.add_argument('--seed',type=int,choices=[37,83,151,211,307,419]);p.add_argument('--budget',type=int,choices=[512,1024,2048],default=2048);p.add_argument('--label',default='v1');a=p.parse_args()
 src,work,binary=a.source.resolve(),a.work.resolve(),a.binary.resolve()
 if src==work or src in work.parents or work in src.parents:p.error('source and work must be disjoint')
 ex=work.parent/(work.name+'-execution');ex.mkdir(exist_ok=True);name=f'{a.phase}-{a.stage}-{a.seed or 0}-{a.label}'
 if a.stage=='confirmatory':
  lock=work/('TRAINING_LOCK.json' if a.phase in ['train','iid'] else 'CONFIRMATORY_LOCK.json')
  meta=json.loads(lock.read_text())
  for rel,digest in meta['Files'].items():
   if sha(Path(rel))!=digest:raise RuntimeError('frozen file changed: '+rel)
 env=os.environ.copy();env.update(GOMAXPROCS='2',HORIZON_S2R7_SOURCE=str(src),HORIZON_S2R7_WORK=str(work),HORIZON_S2R7_PHASE=a.stage,HORIZON_S2R7_BUDGET=str(a.budget))
 if a.seed:env['HORIZON_S2R7_SEED']=str(a.seed)
 record={'phase':a.phase,'stage':a.stage,'seed':a.seed,'budget':a.budget,'binary_sha256':sha(binary),'source':str(src),'work':str(work),'commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'sources':{str(f.relative_to(ROOT)):sha(f) for pattern in ['*s2r7*.go','*regularized*.go','*s2r7*.py'] for f in ROOT.rglob(pattern)}}
 with (ex/(name+'-invocation.json')).open('x') as f:json.dump(record,f,indent=2)
 with (ex/(name+'.log')).open('x') as log:
  result=subprocess.run([str(binary),'-test.run','^TestS2R7'+{'prepare':'Prepare','information':'Information','train':'Train','iid':'IIDTrain','evaluate':'Evaluate','persistence':'Persistence'}[a.phase]+'$','-test.v','-test.timeout','60m'],cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT)
 text=(ex/(name+'.log')).read_text()
 if result.returncode or '--- PASS:' not in text or '--- SKIP:' in text:raise RuntimeError(str(ex/(name+'.log')))
 print(str(ex/(name+'.log')),flush=True)
if __name__=='__main__':main()
