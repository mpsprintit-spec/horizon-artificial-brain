#!/usr/bin/env python3
"""Run isolated S2-R3 phases; immutable source, no promotion or final-data reruns."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT=Path(__file__).resolve().parents[1]
PHASES={'prepare':'Prepare','historical':'HistoricalReplay','train':'Train','controls':'Controls','control-audit':'ControlAudit','freeze':'Freeze','final':'Final','development-input-audit':'DevelopmentInputAudit','persistence':'Persistence','metadata-audit':'FeatureMetadataAudit','cost':'Cost','learning-cost':'LearningCost'}

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--source',type=Path,required=True);p.add_argument('--work',type=Path,required=True)
    p.add_argument('--go',type=Path,required=True);p.add_argument('--phase',choices=['all',*PHASES],default='all')
    p.add_argument('--seed',type=int,choices=[37,83,151]);p.add_argument('--capacity',type=int,choices=[32,64]);a=p.parse_args()
    source,work=a.source.resolve(),a.work.resolve()
    if source==work or source in work.parents or work in source.parents:p.error('source and work must be disjoint')
    if a.phase=='all' and work.exists():p.error('all requires a fresh work path')
    execution=work.parent/(work.name+'-execution');execution.mkdir(exist_ok=True)
    binary=execution/'s2r3.test'
    with (execution/'build-latest.log').open('w') as log:
        subprocess.run([str(a.go),'test','-tags','horizon_s2r2 horizon_s2r3','-c','./services/ai/runtime','-o',str(binary)],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT,check=True)
    env=os.environ.copy();env.update(GOMAXPROCS='2',HORIZON_S2R3_SOURCE=str(source),HORIZON_S2R3_WORK=str(work))
    for phase in PHASES if a.phase=='all' else [a.phase]:
        seeds=([a.seed] if a.seed else [37,83,151]) if phase in ['train','cost','learning-cost'] else [None]
        caps=([a.capacity] if a.capacity else [32,64]) if phase in ['cost','learning-cost'] else [None]
        for seed in seeds:
            for cap in caps:
                name=phase+(f'-{seed}' if seed else '')+(f'-{cap}' if cap else '')
                log=execution/(name+'.log')
                if log.exists():raise RuntimeError('preserve previous phase; do not overwrite '+str(log))
                current=env.copy()
                if seed:current['HORIZON_S2R3_SEED']=str(seed)
                if cap:current['HORIZON_S2R3_CAP']=str(cap)
                with log.open('w') as stream:
                    result=subprocess.run([str(binary),'-test.run','^TestS2R3'+PHASES[phase]+'$','-test.v','-test.timeout','30m' if phase=='train' else '20m'],cwd=ROOT/'services/ai/runtime',env=current,stdout=stream,stderr=subprocess.STDOUT)
                text=log.read_text()
                if result.returncode or '--- SKIP:' in text or '--- PASS:' not in text:raise RuntimeError('phase incomplete; evidence retained: '+str(log))
                if phase=='metadata-audit':subprocess.run(['python3',str(ROOT/'scripts/finalize_s2r3_metadata_audit.py'),'--work',str(work)],check=True)
                print('completed',name,flush=True)
    manifest={}
    for f in execution.glob('*.log'):manifest[f.name]={'bytes':f.stat().st_size,'sha256':hashlib.sha256(f.read_bytes()).hexdigest()}
    (execution/'logs-manifest.json').write_text(json.dumps(manifest,indent=2,sort_keys=True)+'\n')

if __name__=='__main__':main()
