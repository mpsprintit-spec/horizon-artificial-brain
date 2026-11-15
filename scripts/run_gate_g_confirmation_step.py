#!/usr/bin/env python3
"""Execute ONE frozen confirmation command; preserve an interrupted attempt.

Short-lived processes avoid losing a long orchestrator session. No completed
measurement may be overwritten or selected out, and source/command order stays
fixed. Resume only after the original managed session is unavailable.
"""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import time

p=argparse.ArgumentParser(description=__doc__)
p.add_argument('directory',type=Path)
p.add_argument('--resume-after-session-loss',action='store_true')
p.add_argument('--baseline',type=Path)
p.add_argument('--candidate',type=Path)
a=p.parse_args();root=a.directory.resolve();meta_path=root/'runs.json'
meta=json.loads(meta_path.read_text());tasks=[]
def checkout(explicit, prefix):
    if explicit is not None:return explicit.resolve()
    previous=next((row['cwd'] for row in meta['ordered_runs'] if row['label'].startswith(prefix)),None)
    if previous is None:raise SystemExit('supply checkout path for '+prefix)
    return Path(previous).resolve()
baseline=checkout(a.baseline,'A1-')
candidate=checkout(a.candidate,'B1-')
for name,path,key in [('baseline',baseline,'baseline_head'),('candidate',candidate,'candidate_head')]:
    actual=subprocess.check_output(['git','-C',str(path),'rev-parse','HEAD'],text=True).strip()
    if actual!=meta[key]:raise SystemExit('frozen source HEAD changed: '+name)
    harness=path/'services/ai/runtime/gate_g_benchmark_test.go'
    if hashlib.sha256(harness.read_bytes()).hexdigest()!=meta['harness_sha256']:
        raise SystemExit('common harness changed: '+name)
for run,path in [('A1',baseline),('B1',candidate),('B2',candidate),('A2',baseline)]:
    for part,pattern,n in [
        ('thought','^BenchmarkGateGThought$',100),
        ('context-continuity','^BenchmarkGateGContextContinuity$/nodes=40000$/dense=false$/journal=true$',100),
        ('native','^BenchmarkGateGNativeStages$',20),
        ('append','^BenchmarkGateGJournalAppend$',100),
        ('persistence','^BenchmarkNativeActionPersistence$',3)]:
        tasks.append((run+'-'+part,path,pattern,n,0))
tasks += [
    ('candidate-native-idle-thought',candidate,'^BenchmarkGateGThoughtNativeIdle$',100,0),
    ('candidate-native-idle-context-continuity',candidate,'^BenchmarkGateGContextContinuityNativeIdle$/nodes=40000$/dense=false$/journal=true$',100,0),
    ('candidate-admission',candidate,'^BenchmarkGateGAdmission$',100000,0),
    ('candidate-handoff',candidate,'^BenchmarkGateGMixedHandoff$',100,0),
    ('baseline-global-guard-refusal',baseline,'^BenchmarkGateGThoughtNativeIdle$/nodes=1000$/dense=false$/journal=true$',1,1)]
done=meta['ordered_runs']
for i,row in enumerate(done):
    if row['label']!=tasks[i][0] or (row['returncode']==0)!=(row['expected_exit']==0):
        raise SystemExit('completed command order/failure needs review')
if len(done)==len(tasks):print(json.dumps({'complete':True}));raise SystemExit(0)
label,cwd,pattern,n,expected=tasks[len(done)]
active_path=root/'active.json'
if active_path.exists():
    if not a.resume_after_session_loss:raise SystemExit('active command exists; do not duplicate')
    active=json.loads(active_path.read_text())
    if active['label']!=label:raise SystemExit('active/order mismatch')
    old=root/(label+'.txt')
    stamp=datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
    archive=root/(label+'.interrupted-'+stamp+'.txt')
    if old.exists():old.rename(archive)
    meta.setdefault('interrupted_runs',[]).append(dict(active=active,
        reason='managed session unavailable; no terminal command result; complete command repeated unchanged',
        partial_log=archive.name,recorded_at_utc=stamp))
    active_path.unlink()
    meta_path.write_text(json.dumps(meta,indent=2)+'\n')
command=['go','test','./services/ai/runtime','-run','^$','-bench',pattern,
         '-benchtime='+str(n)+'x','-count=1','-timeout=20m']
env=dict(os.environ,GOMAXPROCS='1',GOGC='100');started=time.time()
pending=dict(label=label,command=command,cwd=str(cwd),started_unix=started,orchestration='one-command step')
with (root/(label+'.txt')).open('w') as out:
    child=subprocess.Popen(command,cwd=cwd,env=env,stdout=out,stderr=subprocess.STDOUT,start_new_session=True)
    pending['process_group']=child.pid;active_path.write_text(json.dumps(pending,indent=2))
    timed_out=False
    try:code=child.wait(timeout=1200)
    except subprocess.TimeoutExpired:
        timed_out=True;os.killpg(child.pid,signal.SIGKILL);code=child.wait()
row=dict(label=label,command=command,cwd=str(cwd),returncode=code,
         externally_timed_out=timed_out,expected_exit=expected,seconds=time.time()-started,
         orchestration='one-command step')
meta['ordered_runs'].append(row);meta_path.write_text(json.dumps(meta,indent=2)+'\n');active_path.unlink()
print(json.dumps(dict(row,remaining=len(tasks)-len(meta['ordered_runs']))),flush=True)
if timed_out or (expected==0)!=(code==0):raise SystemExit('unexpected command result; retain evidence')
