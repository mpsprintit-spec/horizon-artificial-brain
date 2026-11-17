#!/usr/bin/env python3
"""Package full S2-R2 checkpoints, journals, controls, code and original source."""
import argparse
import hashlib
import io
import json
import lzma
from pathlib import Path,PurePosixPath
import tarfile

ROOT=Path(__file__).resolve().parents[1]
DOCS=ROOT/'docs/research/s2_r2_repair_20261117'
def digest(raw):return hashlib.sha256(raw).hexdigest()

def verify(path):
    # Sequential decompression: authenticate every payload without repeated
    # seeking through a large compressed full-lineage journal bundle.
    actual={};manifest=None
    with tarfile.open(path,'r|xz') as archive:
        for entry in archive:
            name=entry.name;p=PurePosixPath(name)
            if not entry.isfile() or p.is_absolute() or '..' in p.parts or name in actual:raise ValueError('unsafe/duplicate payload '+name)
            h=hashlib.sha256();size=0;stream=archive.extractfile(entry)
            if name=='MANIFEST.json':
                if manifest is not None:raise ValueError('duplicate manifest')
                manifest=json.load(stream);continue
            for chunk in iter(lambda:stream.read(1<<20),b''):h.update(chunk);size+=len(chunk)
            actual[name]={'bytes':size,'sha256':h.hexdigest()}
    if manifest!=actual:raise ValueError('archive manifest mismatch')
    result={'payloads_verified':len(actual),'bytes':path.stat().st_size,'sha256':digest(path.read_bytes())}
    print(json.dumps(result));return result

def package(work,source,evidence,output,failed=None):
    if output.exists():raise ValueError('never overwrite a published archive')
    files={}
    roots=[('s2r2',work),('source',source),('execution',evidence),('docs',DOCS)]
    if failed:roots.append(('failed-engineering-revision',failed))
    for prefix,base in roots:
        for path in sorted(base.rglob('*')):
            if path.is_file() and not path.name.endswith(('.lock','.test')) and not (prefix=='docs' and path.name=='ARTIFACT.md'):files[prefix+'/'+path.relative_to(base).as_posix()]=path
    for name in ['scripts/run_s2r2_repair.py','scripts/package_s2r2_repair.py','scripts/analyze_s2r2_repair.py','services/ai/runtime/s2r2_repair_test.go','services/ai/runtime/s2r2_saved_audit_test.go','services/ai/runtime/neural_metric_bounded_test.go','services/ai/knowledge/neural_metric_learning.go','services/ai/knowledge/neural_metric_proposal.go','services/ai/knowledge/neural_metric_proposal_test.go','services/ai/knowledge/neural_resolution.go','services/ai/knowledge/neural_metric.go','services/ai/knowledge/neural_resolution_validation.go','services/ai/knowledge/neural_composition.go','services/ai/knowledge/s2r2_probe.go']:files['implementation/'+name]=ROOT/name
    manifest={name:{'bytes':path.stat().st_size,'sha256':digest(path.read_bytes())} for name,path in files.items()}
    # Long identical checkpoint/journal prefixes are losslessly compressed in
    # one dictionary. Keep every complete file; do not replace it with a hash.
    filters=[{'id':lzma.FILTER_LZMA2,'preset':3,'dict_size':128<<20}]
    with lzma.open(output,'wb',filters=filters) as compressed, tarfile.open(fileobj=compressed,mode='w|') as archive:
        for name,path in files.items():
            raw=path.read_bytes()
            if digest(raw)!=manifest[name]['sha256']:raise ValueError('input changed '+name)
            entry=tarfile.TarInfo(name);entry.size=len(raw);entry.mode=0o600;entry.mtime=0;archive.addfile(entry,io.BytesIO(raw))
        raw=(json.dumps(manifest,indent=2,sort_keys=True)+'\n').encode();entry=tarfile.TarInfo('MANIFEST.json');entry.size=len(raw);entry.mode=0o600;entry.mtime=0;archive.addfile(entry,io.BytesIO(raw))
    result=verify(output);output.with_suffix(output.suffix+'.sha256').write_text(result['sha256']+'  '+output.name+'\n')

def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--verify',type=Path);p.add_argument('--work',type=Path);p.add_argument('--source',type=Path);p.add_argument('--evidence',type=Path);p.add_argument('--output',type=Path);p.add_argument('--failed',type=Path);a=p.parse_args()
    if a.verify:verify(a.verify)
    else:
        if not all([a.work,a.source,a.evidence,a.output]):p.error('work, source, evidence and output required')
        package(a.work,a.source,a.evidence,a.output,a.failed)

if __name__=='__main__':main()
