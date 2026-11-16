#!/usr/bin/env python3
"""Continue the verified S1 lineage for S2, or verify the saved S2 artifact.

Never invokes S1 Prepare/Train/Evaluate. The source archive and extracted source
are immutable inputs. A fresh S2 research directory is required for acquisition.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
S1_SHA256 = '7ca53ea11b2a56a149f466d65a1d31733581d893ea810b3003ba3a3895fb12e7'


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--source', type=Path, required=True, help='verified extracted S1 archive root')
    p.add_argument('--source-archive', type=Path, required=True)
    p.add_argument('--work', type=Path, required=True)
    p.add_argument('--main-ci', type=Path, help='GitHub push-run JSON at the post-merge baseline')
    p.add_argument('--go', default='go')
    p.add_argument('--verify-only', action='store_true')
    a = p.parse_args()
    source, work = a.source.resolve(), a.work.resolve()
    if hashlib.sha256(a.source_archive.read_bytes()).hexdigest() != S1_SHA256:
        p.error('S1 archive mismatch: BLOCKED; never reconstruct through training')
    manifest = json.loads((source/'MANIFEST.json').read_text())
    for name, entry in manifest.items():
        data = (source/name).read_bytes()
        if len(data) != entry['bytes'] or hashlib.sha256(data).hexdigest() != entry['sha256']:
            p.error(f'S1 payload mismatch: {name}')
    if source == work or source in work.parents or work in source.parents:
        p.error('work and source must be disjoint')
    if not a.verify_only and (work.exists() or not a.main_ci):
        p.error('acquisition requires new work and explicit successful main push CI evidence')
    env = dict(os.environ, HORIZON_S2_SOURCE=str(source), HORIZON_S2_WORK=str(work))
    if a.main_ci:
        env['HORIZON_S2_CI'] = str(a.main_ci.resolve())
    phases = ['Verify'] if a.verify_only else ['Prepare', 'Train', 'Evaluate', 'Retention', 'Persistence', 'RestartRetention', 'SavedS1Attribution', 'Verify']
    with tempfile.TemporaryDirectory(prefix='horizon-s2-') as temp:
        temp = Path(temp)
        binary = temp/'research.test'
        subprocess.run([a.go, 'test', '-c', './services/ai/runtime', '-o', str(binary)], cwd=ROOT, env=env, check=True)
        for phase in phases:
            log = temp/f's2-{phase.lower()}.log'
            with log.open('w') as output:
                result = subprocess.run([str(binary), '-test.run', f'^TestS2{phase}$', '-test.v', '-test.timeout', '15m'],
                                        cwd=ROOT/'services/ai/runtime', env=env, stdout=output, stderr=subprocess.STDOUT)
            if work.exists():
                evidence = work/'evidence'
                evidence.mkdir(exist_ok=True)
                # Verification is read-only for existing evidence and does not
                # overwrite the original acquisition/restart logs.
                dst = evidence/log.name
                if not dst.exists():
                    shutil.copy2(log, dst)
            print(log.read_text(), end='', flush=True)
            result.check_returncode()


if __name__ == '__main__':
    main()
