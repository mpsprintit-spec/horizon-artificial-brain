#!/usr/bin/env python3
"""Reproduce preregistered S2-R2 on the original S2 continuation; never promotes."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT=Path(__file__).resolve().parents[1]
PHASES={
    'prepare':'Prepare','mechanism':'Mechanism','train':'Train',
    'retry':'IdenticalRetryAndDuplicateCommand','freeze':'RetentionAndFreeze',
    'final':'FinalIndependentHoldout','persistence':'PersistenceAndContinuation',
    'saved-audit':'SavedReadoutAndS1Attribution',
}

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--source',type=Path,required=True)
    p.add_argument('--work',type=Path,required=True)
    p.add_argument('--go',required=True,help='absolute Go 1.22+ compiler path')
    p.add_argument('--phase',choices=['all',*PHASES],default='all')
    a=p.parse_args();source=a.source.resolve();work=a.work.resolve()
    if source==work or source in work.parents or work in source.parents:p.error('source and work must be isolated')
    if a.phase=='all' and work.exists():p.error('all requires a fresh work path; do not overwrite an experiment')
    manifest=json.loads((source/'MANIFEST.json').read_text())
    for name,v in manifest.items():
        raw=(source/name).read_bytes()
        if len(raw)!=v['bytes'] or hashlib.sha256(raw).hexdigest()!=v['sha256']:raise ValueError('BLOCKED: original source mismatch '+name)
    evidence=work.parent/(work.name+'-execution');evidence.mkdir(exist_ok=True)
    binary=evidence/'s2r2.test'
    with (evidence/'build.log').open('w') as log:
        subprocess.run([a.go,'test','-tags','horizon_s2r2','-c','./services/ai/runtime','-o',str(binary)],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT,check=True)
    env=os.environ.copy();env.update(HORIZON_S2R2_SOURCE=str(source),HORIZON_S2R2_WORK=str(work))
    for phase in (PHASES if a.phase=='all' else [a.phase]):
        logfile=evidence/(phase+'.log')
        if logfile.exists():raise ValueError('preserve prior phase log; choose a new run, or audit a failed infrastructure step explicitly')
        with logfile.open('w') as log:
            subprocess.run([str(binary),'-test.run','^TestS2R2'+PHASES[phase]+'$','-test.v','-test.timeout','20m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=log,stderr=subprocess.STDOUT,check=True)
        print('completed',phase,flush=True)
    if (work/'results.json').exists():print((work/'results.json').read_text())
    if (work/'persistence.json').exists() and (work/'saved-readout-audit.json').exists():
        persistence=json.loads((work/'persistence.json').read_text())
        audit=json.loads((work/'saved-readout-audit.json').read_text())
        result=json.loads((work/'results.json').read_text())
        keys=['restart_exact','full_suffix_replay_exact','rollback_exact','continuation_replay_exact','historical_journal_prefix_exact']
        if all(persistence[k] for k in keys) and audit['pass'] and result['gates']['AE1_retention'] and result['gates']['S1_retention']:
            result['repair_status']='PASS'
            result['continuation_sequence']=persistence['checkpoints'][-1]['sequence']
            (work/'results.json').write_text(json.dumps(result,indent=2,sort_keys=True)+'\n')

if __name__=='__main__':main()
