#!/usr/bin/env python3
"""Isolated phased R6 runner. Logs are exclusive; no production paths accepted."""
import argparse,hashlib,json,os,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def main():
 p=argparse.ArgumentParser();p.add_argument('--source',type=Path,required=True);p.add_argument('--work',type=Path,required=True);p.add_argument('--binary',type=Path,required=True);p.add_argument('--phase',choices=['prepare','coverage','development','dev-recovery','train','preflight','acquire','evaluate','persistence'],required=True);p.add_argument('--seed',type=int,choices=[37,83,151,211,307,419]);a=p.parse_args()
 src,work,binary=a.source.resolve(),a.work.resolve(),a.binary.resolve()
 if src==work or src in work.parents or work in src.parents:p.error('source and work must be disjoint')
 ex=work.parent/(work.name+'-execution');ex.mkdir(exist_ok=True)
 env=os.environ.copy();env.update(GOMAXPROCS='2',HORIZON_S2R6_SOURCE=str(src),HORIZON_S2R6_WORK=str(work))
 if a.seed:env['HORIZON_S2R6_SEED']=str(a.seed)
 name=a.phase+(f'-{a.seed}' if a.seed else '')
 record={'phase':a.phase,'seed':a.seed,'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'source':str(src),'work':str(work),'code_commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'source_files':{str(p.relative_to(ROOT)):hashlib.sha256(p.read_bytes()).hexdigest() for p in ROOT.rglob('*s2r6*') if p.is_file() and p.suffix in ['.go','.py']}}
 with (ex/(name+'-invocation.json')).open('x') as f:json.dump(record,f,indent=2)
 with (ex/(name+'.log')).open('x') as log:r=subprocess.run([str(binary),'-test.run','^TestS2R6'+{'prepare':'Prepare','coverage':'CoveragePrepare','development':'Development','dev-recovery':'DevelopmentPersistence','train':'Train','preflight':'Preflight','acquire':'AcquireFinal','evaluate':'EvaluateFinal','persistence':'Persistence'}[a.phase]+'$','-test.v','-test.timeout','60m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=log,stderr=subprocess.STDOUT)
 text=(ex/(name+'.log')).read_text()
 if r.returncode or '--- PASS:' not in text or '--- SKIP:' in text:raise RuntimeError(str(ex/(name+'.log')))
 print(ex/(name+'.log'),flush=True)
if __name__=='__main__':main()
