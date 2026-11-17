#!/usr/bin/env python3
"""Independent native recovery/retention per model; identical complete tests."""
import argparse,concurrent.futures,json,os,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def main():
 p=argparse.ArgumentParser();p.add_argument('--source',type=Path,required=True);p.add_argument('--work',type=Path,required=True);p.add_argument('--binary',type=Path,required=True);p.add_argument('--workers',type=int,default=4);a=p.parse_args();w=a.work.resolve();ex=w.parent/(w.name+'-execution');ex.mkdir(exist_ok=True)
 jobs=[(s,r) for s in [37,83,151] for r in ['A32','B32','C32','B0-32','D32','D0-32','A64','C64']]
 def run(job):
  seed,arm=job;log=ex/f'persistence-{seed}-{arm}.log'
  if log.exists():raise RuntimeError('never overwrite existing evidence '+str(log))
  env=os.environ.copy();env.update(GOMAXPROCS='2',HORIZON_S2R4_SOURCE=str(a.source.resolve()),HORIZON_S2R4_WORK=str(w),HORIZON_S2R4_SEED=str(seed),HORIZON_S2R4_PERSISTENCE_ARM=arm)
  with log.open('w') as f:r=subprocess.run([str(a.binary.resolve()),'-test.run','^TestS2R4Persistence$','-test.v','-test.timeout','30m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=f,stderr=subprocess.STDOUT)
  if r.returncode or '--- PASS:' not in log.read_text():raise RuntimeError('native proof incomplete '+str(log))
  return job
 with concurrent.futures.ThreadPoolExecutor(max_workers=a.workers) as pool:
  for s,r in pool.map(run,jobs):print('exact native recovery/retention completed',s,r,flush=True)
 merged={}
 for seed,arm in jobs:
  v=json.loads((w/f'persistence-individual/{seed}-{arm}.json').read_text());assert len(v)==1;assert not(set(v)&set(merged));merged.update(v)
 assert len(merged)==24
 (w/'persistence.json').write_text(json.dumps(merged,indent=2,sort_keys=True)+'\n')
 print('all 24 independent complete proofs saved',flush=True)
if __name__=='__main__':main()
