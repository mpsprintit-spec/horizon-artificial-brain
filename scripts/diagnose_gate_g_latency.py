#!/usr/bin/env python3
"""Profile failed Gate G resource cells on the ORIGINAL measured revisions.

Diagnostic results do not replace the retained ABBA or change its gate verdict.
Compile before measurement; execute serially without another heavy local task.
"""
import argparse
import json
import os
from pathlib import Path
import resource
import subprocess
import time

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--baseline', type=Path, required=True)
p.add_argument('--candidate', type=Path, required=True)
p.add_argument('--output', type=Path, required=True)
p.add_argument('--binaries', type=Path, required=True)
a = p.parse_args()
a.output.mkdir(parents=True, exist_ok=False)
a.binaries.mkdir(parents=True, exist_ok=True)
env = dict(os.environ, GOMAXPROCS='1', GOGC='100')
roots = {'baseline': a.baseline.resolve(), 'candidate': a.candidate.resolve()}
meta = {'purpose': 'root-cause diagnosis, not a new confirmation', 'runs': [], 'sources': {}}
for name, root in roots.items():
    meta['sources'][name] = subprocess.check_output(['git','-C',str(root),'rev-parse','HEAD'],text=True).strip()
    subprocess.run(['go','test','-c','-o',str((a.binaries/name).resolve()),'./services/ai/runtime'],
                   cwd=root,env=env,check=True)

cases = [
    ('sparse40k', '^BenchmarkGateGThought$/nodes=40000$/dense=false$/journal=false$',100),
    ('dense4k-wal', '^BenchmarkGateGThought$/nodes=4000$/dense=true$/journal=true$',100),
    ('native4k-dense', '^BenchmarkGateGNativeStages$/nodes=4000$/dense=true$',20),
    ('checkpoint1k', '^BenchmarkNativeActionPersistence$/nodes=1000$/mode=checkpoint$',3),
]
for case, pattern, samples in cases:
    for revision, root in roots.items():
        label=case+'-'+revision
        binary=(a.binaries/revision).resolve()
        profile=(a.output/(label+'.cpu')).resolve()
        command=[str(binary),'-test.run=^$','-test.bench='+pattern,
                 '-test.benchtime='+str(samples)+'x','-test.count=1',
                 '-test.cpuprofile='+str(profile)]
        before=resource.getrusage(resource.RUSAGE_CHILDREN)
        started=time.monotonic()
        with (a.output/(label+'.txt')).open('w') as log:
            result=subprocess.run(command,cwd=root,env=env,stdout=log,stderr=subprocess.STDOUT,timeout=300)
        after=resource.getrusage(resource.RUSAGE_CHILDREN)
        row=dict(label=label,command=command,returncode=result.returncode,
                 seconds=time.monotonic()-started,
                 child_user_seconds=after.ru_utime-before.ru_utime,
                 child_system_seconds=after.ru_stime-before.ru_stime)
        meta['runs'].append(row)
        (a.output/'runs.json').write_text(json.dumps(meta,indent=2)+'\n')
        print(json.dumps(row),flush=True)
        if result.returncode:
            raise SystemExit('profile command failed; preserve evidence')
for row in meta['runs']:
    revision=row['label'].rsplit('-',1)[1]
    with (a.output/(row['label']+'-top.txt')).open('w') as log:
        subprocess.run(['go','tool','pprof','-top','-nodecount=25',str((a.binaries/revision).resolve()),
                        str((a.output/(row['label']+'.cpu')).resolve())],
                       env=env,stdout=log,stderr=subprocess.STDOUT,check=True)
