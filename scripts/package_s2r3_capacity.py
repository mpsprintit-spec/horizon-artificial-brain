#!/usr/bin/env python3
"""Lossless S2-R3 owner recovery bundle: full checkpoints/WAL/data/negative evidence."""
import argparse
import hashlib
import io
import json
import lzma
from pathlib import Path
import subprocess
import tarfile

from package_s2r2_repair import verify

ROOT=Path(__file__).resolve().parents[1]
DOCS=ROOT/'docs/research/s2_r3_capacity_20261117'
SOURCE_SHA='e51b61bf41107f4f74e7d72d6360b22cd4671d0f6ce84e867212090290e0aa46'
BASE='7196c8a7f4a619040fe3933fe1c5363e5be661bc'
def sha(path):
    h=hashlib.sha256()
    with path.open('rb') as f:
        for b in iter(lambda:f.read(1<<20),b''):h.update(b)
    return h.hexdigest()

def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--verify',type=Path)
    p.add_argument('--work',type=Path);p.add_argument('--execution',type=Path);p.add_argument('--source-archive',type=Path);p.add_argument('--preflight',type=Path);p.add_argument('--output',type=Path);a=p.parse_args()
    if a.verify:verify(a.verify);return
    if not all([a.work,a.execution,a.source_archive,a.output]):p.error('work, execution, source-archive, output required')
    if a.output.exists():raise ValueError('never overwrite an owner artifact')
    if sha(a.source_archive)!=SOURCE_SHA:raise ValueError('BLOCKED: source archive changed')
    files={}
    roots=[('s2r3',a.work),('execution',a.execution),('docs',DOCS)]
    if a.preflight:roots.append(('preflight-no-acquisition',a.preflight))
    for prefix,base in roots:
        for path in sorted(base.rglob('*')):
            if not path.is_file() or path.name.endswith(('.lock','.test')) or path.name.startswith('horizon-') and 'android-arm64' in path.name:continue
            if prefix=='docs' and path.name=='ARTIFACT.md':continue
            files[prefix+'/'+path.relative_to(base).as_posix()]=path
    files['source/'+a.source_archive.name]=a.source_archive
    for name in ['run_s2r3_capacity.py','analyze_s2r3_capacity.py','package_s2r3_capacity.py','package_s2r2_repair.py','finalize_s2r3_metadata_audit.py']:
        files['reproduce/'+name]=ROOT/'scripts'/name
    # The external repository owns source. A revision-pinned patch records the
    # scientific implementation delta without making a second repository copy.
    commit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip()
    patch=subprocess.check_output(['git','diff',BASE,commit,'--','services/ai/knowledge','services/ai/runtime','scripts'],cwd=ROOT)
    metadata={'repository':'mpsprintit-spec/horizon-artificial-brain','base':BASE,'source_commit':commit,'branch':'research/s2-r3-neural-capacity-20261117','source_archive_sha256':SOURCE_SHA,'source_sequence':80735,'code_delta':'implementation.patch','original_full_archive_included':True,'main_changed':False,'production_promotion':False}
    extras={'implementation.patch':patch,'BUNDLE.json':(json.dumps(metadata,indent=2,sort_keys=True)+'\n').encode()}
    manifest={name:{'bytes':path.stat().st_size,'sha256':sha(path)} for name,path in files.items()}
    manifest.update({name:{'bytes':len(raw),'sha256':hashlib.sha256(raw).hexdigest()} for name,raw in extras.items()})
    with lzma.open(a.output,'wb',filters=[{'id':lzma.FILTER_LZMA2,'preset':3,'dict_size':128<<20}]) as z,tarfile.open(fileobj=z,mode='w|') as tar:
        for name,path in files.items():
            if sha(path)!=manifest[name]['sha256']:raise ValueError('payload changed '+name)
            entry=tarfile.TarInfo(name);entry.size=path.stat().st_size;entry.mode=0o600
            with path.open('rb') as f:tar.addfile(entry,f)
        extras['MANIFEST.json']=(json.dumps(manifest,indent=2,sort_keys=True)+'\n').encode()
        for name,raw in extras.items():
            entry=tarfile.TarInfo(name);entry.size=len(raw);entry.mode=0o600;tar.addfile(entry,io.BytesIO(raw))
    result=verify(a.output);a.output.with_suffix(a.output.suffix+'.sha256').write_text(result['sha256']+'  '+a.output.name+'\n')

if __name__=='__main__':main()
