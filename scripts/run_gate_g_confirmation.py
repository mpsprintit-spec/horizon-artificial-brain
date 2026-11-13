#!/usr/bin/env python3
"""Sequential, frozen ABBA resource checks for the Gate G integration.

Both checkouts must already contain the identical gate_g_benchmark_test.go.
Use a NEW output directory. No production checkpoint or journal is accepted.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import time

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--baseline', type=Path, required=True)
p.add_argument('--candidate', type=Path, required=True)
p.add_argument('--output', type=Path, required=True)
a = p.parse_args()
a.baseline, a.candidate, a.output = (x.resolve() for x in (a.baseline, a.candidate, a.output))
a.output.mkdir(parents=True, exist_ok=False)
env = dict(os.environ, GOMAXPROCS='1', GOGC='100')

def git(root, *args):
    return subprocess.check_output(['git', '-C', str(root), *args], text=True).strip()

harness = 'services/ai/runtime/gate_g_benchmark_test.go'
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
if sha(a.baseline / harness) != sha(a.candidate / harness):
    raise SystemExit('baseline/candidate benchmark harnesses differ')
meta = dict(baseline_head=git(a.baseline, 'rev-parse', 'HEAD'),
            candidate_head=git(a.candidate, 'rev-parse', 'HEAD'),
            baseline_tree=git(a.baseline, 'rev-parse', 'HEAD^{tree}'),
            candidate_tree=git(a.candidate, 'rev-parse', 'HEAD^{tree}'),
            harness_sha256=sha(a.candidate / harness),
            go=subprocess.check_output(['go', 'version'], text=True).strip(),
            platform=platform.platform(), GOMAXPROCS=1, GOGC=100,
            ordered_runs=[])

def run(label, root, bench, samples, expected=0):
    command = ['go', 'test', './services/ai/runtime', '-run', '^$', '-bench', bench,
               f'-benchtime={samples}x', '-count=1', '-timeout=20m']
    started = time.time()
    with (a.output / (label + '.txt')).open('w') as out:
        result = subprocess.run(command, cwd=root, env=env, stdout=out, stderr=subprocess.STDOUT)
    row = dict(label=label, command=command, cwd=str(root), returncode=result.returncode,
               expected_exit=expected, seconds=time.time() - started)
    meta['ordered_runs'].append(row)
    (a.output / 'runs.json').write_text(json.dumps(meta, indent=2))
    print(json.dumps(row), flush=True)
    if (expected == 0) != (result.returncode == 0):
        raise SystemExit('unexpected benchmark result; retain evidence and diagnose')

for label, root in [('A1', a.baseline), ('B1', a.candidate), ('B2', a.candidate), ('A2', a.baseline)]:
    run(label + '-thought', root, '^BenchmarkGateGThought$', 100)
    run(label + '-native', root, '^BenchmarkGateGNativeStages$', 20)
    run(label + '-append', root, '^BenchmarkGateGJournalAppend$', 100)
    run(label + '-persistence', root, '^BenchmarkNativeActionPersistence$', 3)
run('candidate-native-idle-thought', a.candidate, '^BenchmarkGateGThoughtNativeIdle$', 100)
run('candidate-admission', a.candidate, '^BenchmarkGateGAdmission$', 100000)
run('candidate-handoff', a.candidate, '^BenchmarkGateGMixedHandoff$', 100)
# An authentic baseline failure is recorded explicitly, never assigned a latency.
run('baseline-mixed-refusal', a.baseline, '^BenchmarkGateGMixedHandoff/nodes=1000$', 1, expected=1)
