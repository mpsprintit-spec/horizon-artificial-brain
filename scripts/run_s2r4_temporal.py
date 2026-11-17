#!/usr/bin/env python3
"""Run opt-in S2-R4 native phases on disjoint immutable-source copies."""
import argparse,hashlib,json,os,subprocess
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
PHASES={'development':'Development','development-learning':'DevelopmentLearning','train':'Train','acquire-final':'AcquireFinal','evaluate-final':'EvaluateFinal','persistence':'Persistence'}
def digest(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--source',type=Path,required=True);p.add_argument('--work',type=Path,required=True);p.add_argument('--go',type=Path,required=True);p.add_argument('--phase',choices=list(PHASES),required=True);p.add_argument('--seed',type=int,choices=[37,83,151]);p.add_argument('--binary',type=Path);a=p.parse_args()
 source,work=a.source.resolve(),a.work.resolve()
 if source==work or source in work.parents or work in source.parents:p.error('source/work must be disjoint')
 ex=work.parent/(work.name+'-execution');ex.mkdir(exist_ok=True);binary=a.binary.resolve() if a.binary else ex/'s2r4.test'
 if not a.binary:
  with (ex/('build-'+a.phase+'.log')).open('w') as log:subprocess.run([str(a.go),'test','-tags','horizon_s2r2 horizon_s2r3 horizon_s2r4','-c','./services/ai/runtime','-o',str(binary)],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT,check=True)
 seeds=([a.seed] if a.seed else [37,83,151]) if a.phase=='train' else [None]
 for seed in seeds:
  name=a.phase+(f'-{seed}' if seed else '');log=ex/(name+'.log')
  if log.exists():raise RuntimeError('preserve previous execution log; use another work tree: '+str(log))
  env=os.environ.copy();env.update(GOMAXPROCS='2',HORIZON_S2R4_SOURCE=str(source),HORIZON_S2R4_WORK=str(work))
  if seed:env['HORIZON_S2R4_SEED']=str(seed)
  with log.open('w') as f:r=subprocess.run([str(binary),'-test.run','^TestS2R4'+PHASES[a.phase]+'$','-test.v','-test.timeout','60m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=f,stderr=subprocess.STDOUT)
  text=log.read_text()
  if r.returncode or '--- SKIP:' in text or '--- PASS:' not in text:raise RuntimeError('phase incomplete; evidence retained: '+str(log))
  print('completed',name,flush=True)
if __name__=='__main__':main()
