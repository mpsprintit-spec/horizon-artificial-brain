#!/usr/bin/env python3
"""Reproduce S2-R1 from the immutable S2 bundle, without training or promotion."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess

ROOT=Path(__file__).resolve().parents[1]


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--source',type=Path,required=True,help='verified extracted S2 archive root')
    p.add_argument('--work',type=Path,required=True,help='new isolated output directory')
    p.add_argument('--go',default='go',help='Go 1.22+ compiler executable')
    a=p.parse_args();source=a.source.resolve();work=a.work.resolve()
    if work.exists():p.error('refuse to overwrite existing diagnostic output; choose a new work directory')
    if source==work or source in work.parents or work in source.parents:p.error('source and work must be isolated sibling trees')
    if not (source/'s2/lineage/s2-continuation.json').is_file():p.error('original S2 continuation is required; never reconstruct by training')
    generated=ROOT/'services/ai/knowledge/zz_s2r1_trace_generated.go'
    if generated.exists():p.error('generated trace already exists; preserve or remove it explicitly before reproduction')
    work.mkdir(parents=True);evidence=work/'evidence';evidence.mkdir()
    manifest=json.loads((source/'MANIFEST.json').read_text())
    def authenticate():
        for name,expected in manifest.items():
            path=source/name;data=path.read_bytes()
            if len(data)!=expected['bytes'] or hashlib.sha256(data).hexdigest()!=expected['sha256']:
                raise ValueError('BLOCKED: original payload integrity failure: '+name)
    authenticate()
    version=subprocess.check_output([a.go,'version'],text=True)
    if not version.startswith('go version go'):raise ValueError('--go must name the Go compiler')
    (evidence/'runtime-version.txt').write_text(version)
    protocol=ROOT/'docs/research/s2_r1_diagnostic_20261116'
    (evidence/'protocol-hashes.json').write_text(json.dumps({path.name:hashlib.sha256(path.read_bytes()).hexdigest() for path in sorted(protocol.glob('*PROTOCOL.json'))},indent=2)+'\n')
    spec=importlib.util.spec_from_file_location('s2r1_prepare',ROOT/'scripts/prepare_s2r1_diagnostic.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    try:
        module.generate()
        with (evidence/'build.log').open('w') as log:
            subprocess.run([a.go,'test','-tags','horizon_s2r1','-c','./services/ai/runtime','-o',str(work/'diagnostic.test')],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT,check=True)
        env=os.environ.copy();env.update(HORIZON_S2R1_SOURCE=str(source),HORIZON_S2R1_WORK=str(work))
        with (evidence/'diagnostic.log').open('w') as log:
            subprocess.run([str(work/'diagnostic.test'),'-test.run','^TestS2R1','-test.v','-test.timeout','20m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=log,stderr=subprocess.STDOUT,check=True)
        authenticate()
        subprocess.run(['python3',str(ROOT/'scripts/analyze_s2r1_diagnostic.py'),'--work',str(work),'--output',str(work/'analysis')],cwd=ROOT,check=True)
        (evidence/'source-after.json').write_text(json.dumps({'payloads_verified':len(manifest),'source_unchanged':True,'new_learning_commands':0,'final_holdout_evaluations':0},indent=2)+'\n')
    finally:
        generated.unlink(missing_ok=True)
    print(work/'analysis/summary.json')


if __name__=='__main__':main()
