#!/usr/bin/env python3
"""Run the frozen S1 protocol, or verify its saved state without training."""
import argparse
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--source', type=Path, required=True,
                   help='authenticated prepare_native_continuation.py output')
    p.add_argument('--work', type=Path, required=True,
                   help='new isolated lineage, or extracted s1 artifact for verification')
    p.add_argument('--go', default='go')
    p.add_argument('--verify-only', action='store_true')
    p.add_argument('--continue-training', action='store_true',
                   help='resume the same pretraining checkpoint/WAL after interruption')
    a = p.parse_args()
    work, source = a.work.resolve(), a.source.resolve()
    if not a.verify_only and not a.continue_training and work.exists():
        p.error('work exists: use --verify-only or --continue-training; never overwrite a lineage')
    env = dict(os.environ, HORIZON_S1_SOURCE=str(source), HORIZON_S1_WORK=str(work))
    runtime = ROOT / 'services/ai/runtime'
    with tempfile.TemporaryDirectory(prefix='horizon-s1-') as tmp:
        binary = str(Path(tmp) / 's1.test')
        subprocess.run([a.go, 'test', '-c', './services/ai/runtime', '-o', binary],
                       cwd=ROOT, env=env, check=True)
        phases = ['VerifyArtifact'] if a.verify_only else (
            ([] if a.continue_training else ['Prepare']) +
            ['Train', 'Evaluate', 'Restore', 'RetentionDiagnosis', 'VerifyArtifact'])
        for phase in phases:
            # Completed held-out evaluation is immutable and must not be rerun.
            if phase == 'Evaluate' and (work / 'results.json').exists():
                continue
            if phase == 'Train' and (work / 'lineage/s1-final.json').exists():
                continue
            if phase == 'Restore' and (work / 'lineage/s1-continuation.json').exists():
                continue
            subprocess.run([binary, '-test.run', f'^TestS1{phase}$',
                            '-test.v', '-test.timeout', '10m'],
                           cwd=runtime, env=env, check=True)


if __name__ == '__main__':
    main()
