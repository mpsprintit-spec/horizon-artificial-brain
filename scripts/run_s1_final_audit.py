#!/usr/bin/env python3
"""Audit an extracted, saved S1 lineage; never run acquisition or final holdout.

Usage: python3 scripts/run_s1_final_audit.py --archive S1.tar.xz
       --root EXTRACTED --output NEW_EVALUATION_DIRECTORY [--go /path/to/go]
The source and production state are never replaced. Output contains disposable
time controls, numeric evidence and logs. Keep the original archive immutable.
"""
import argparse
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--archive', type=Path, required=True)
    p.add_argument('--root', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--go', default='go')
    args = p.parse_args()
    extracted, output = args.root.resolve(), args.output.resolve()
    if output.exists() or output.is_relative_to(extracted):
        p.error('output must be new and outside the immutable extracted inputs')
    env = dict(os.environ, HORIZON_S1_SOURCE=str(extracted/'source'),
               HORIZON_S1_WORK=str(extracted/'s1'), HORIZON_S1_AUDIT=str(output))
    with tempfile.TemporaryDirectory(prefix='s1-final-readonly-') as temp:
        binary = Path(temp)/'audit.test'
        subprocess.run([args.go, 'test', '-c', './services/ai/runtime', '-o', str(binary)],
                       cwd=ROOT, env=env, check=True)
        subprocess.run([str(binary), '-test.run', '^TestS1(VerifyArtifact|FinalRetentionAudit|FinalPersistenceAudit|FinalPassiveTimeAudit)$',
                        '-test.v', '-test.timeout', '10m'],
                       cwd=ROOT/'services/ai/runtime', env=env, check=True)
    subprocess.run(['python3', str(ROOT/'scripts/audit_s1_payloads.py'),
                    '--archive', str(args.archive.resolve()), '--root', str(extracted),
                    '--audit-dir', str(output), '--output', str(output/'payload-audit.json')],
                   cwd=ROOT, check=True)


if __name__ == '__main__':
    main()
