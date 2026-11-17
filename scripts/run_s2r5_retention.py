#!/usr/bin/env python3
"""Reproduce opt-in native S2-R5 phases; never overwrite an experiment log."""
import argparse,os,subprocess,concurrent.futures
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
PHASES={'prepare':'Prepare','development':'Development','fixed':'FixedDevelopment','anchors':'Anchors','coverage':'CoverageDevelopment','structure':'StructureDevelopment','capacity':'CapacityDevelopment','negative-block':'NegativeBlockDevelopment','train':'Train','calibration-controls':'CalibrationControls','preflight':'RecoveryPreflight','acquire-final':'AcquireFinal','evaluate-final':'EvaluateFinal','persistence':'Persistence'}
def main():
 p=argparse.ArgumentParser();p.add_argument('--source',type=Path,required=True);p.add_argument('--work',type=Path,required=True);p.add_argument('--go',type=Path,required=True);p.add_argument('--binary',type=Path);p.add_argument('--r4',type=Path);p.add_argument('--phase',choices=PHASES,required=True);p.add_argument('--seed',type=int,choices=[37,83,151,211,307,419]);p.add_argument('--jobs',type=int,default=3);a=p.parse_args();w=a.work.resolve();src=a.source.resolve();ex=w.parent/(w.name+'-execution');ex.mkdir(exist_ok=True)
 if w==src or w in src.parents or src in w.parents:p.error('source and work must be disjoint')
 binary=a.binary.resolve() if a.binary else ex/'s2r5.test'
 if not a.binary:
  with (ex/('build-'+a.phase+'.log')).open('x') as f:subprocess.run([str(a.go),'test','-tags','horizon_s2r2 horizon_s2r3 horizon_s2r4 horizon_s2r5','-c','./services/ai/runtime','-o',str(binary)],cwd=ROOT,stdout=f,stderr=subprocess.STDOUT,check=True)
 def run(seed):
  env=os.environ.copy();env.update(GOMAXPROCS='2',HORIZON_S2R5_SOURCE=str(src),HORIZON_S2R5_WORK=str(w))
  if seed:env['HORIZON_S2R5_SEED']=str(seed)
  if a.r4:env['HORIZON_S2R5_R4']=str(a.r4.resolve())
  log=ex/(a.phase+(f'-{seed}' if seed else '')+'.log')
  with log.open('x') as f:r=subprocess.run([str(binary),'-test.run','^TestS2R5'+PHASES[a.phase]+'$','-test.v','-test.timeout','60m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=f,stderr=subprocess.STDOUT)
  text=log.read_text()
  if r.returncode or '--- PASS:' not in text or '--- SKIP:' in text:raise RuntimeError(str(log))
  return str(log)
 seeds=([a.seed] if a.seed else [37,83,151,211,307,419]) if a.phase in ['train','evaluate-final','persistence'] else [None]
 with concurrent.futures.ThreadPoolExecutor(max_workers=a.jobs) as pool:
  for future in concurrent.futures.as_completed([pool.submit(run,s) for s in seeds]):print(future.result(),flush=True)
if __name__=='__main__':main()
