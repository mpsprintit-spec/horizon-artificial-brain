#!/usr/bin/env python3
"""Supplementary file-backed journal ABBA, run AFTER the main Gate G matrix."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import time

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--baseline', type=Path, required=True)
p.add_argument('--candidate', type=Path, required=True)
p.add_argument('--output', type=Path, required=True)
a = p.parse_args()
a.baseline, a.candidate, a.output = (v.resolve() for v in (a.baseline, a.candidate, a.output))
a.output.mkdir(parents=True, exist_ok=False)
tmp = a.output / 'temporary-fixtures'
tmp.mkdir()
env = dict(os.environ, GOMAXPROCS='1', GOGC='100', TMPDIR=str(tmp))
meta = dict(go=subprocess.check_output(['go', 'version'], text=True).strip(),
            filesystem=subprocess.check_output(['stat', '-f', '-c', '%T', str(tmp)], text=True).strip(),
            GOMAXPROCS=1, GOGC=100, TMPDIR=str(tmp), ordered_runs=[])
if meta['filesystem'] != 'overlayfs':
    raise SystemExit('expected declared overlayfs; inspect environment before running')
for label, root in [('A1', a.baseline), ('B1', a.candidate), ('B2', a.candidate), ('A2', a.baseline)]:
    command = ['go', 'test', './services/ai/runtime', '-run', '^$',
               '-bench', '^BenchmarkGateGJournalAppend$', '-benchtime=100x', '-count=1']
    started = time.time()
    with (a.output / (label + '.txt')).open('w') as out:
        r = subprocess.run(command, cwd=root, env=env, stdout=out, stderr=subprocess.STDOUT, timeout=300)
    meta['ordered_runs'].append(dict(label=label, command=command, cwd=str(root),
                                     returncode=r.returncode, seconds=time.time()-started))
    (a.output / 'runs.json').write_text(json.dumps(meta, indent=2) + '\n')
    if r.returncode:
        raise SystemExit('append command failed; retain evidence')
print(json.dumps(meta))
