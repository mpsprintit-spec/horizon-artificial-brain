#!/usr/bin/env python3
"""Package or verify S2's complete checkpoint/WAL research artifact, without training."""
import argparse
import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import tarfile

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT/'docs/research/s2_self_evaluation_20261116'


def digest(data):
    return hashlib.sha256(data).hexdigest()


def verify(path):
    with tarfile.open(path, 'r:xz') as archive:
        members = archive.getmembers()
        names = [m.name for m in members]
        if len(names) != len(set(names)):
            raise ValueError('duplicate archive names')
        for m in members:
            if not m.isfile() or PurePosixPath(m.name).is_absolute() or '..' in PurePosixPath(m.name).parts:
                raise ValueError('unsafe archive member')
        manifest = json.load(archive.extractfile('MANIFEST.json'))
        if set(names) != set(manifest)|{'MANIFEST.json'}:
            raise ValueError('manifest does not cover all payloads')
        for name, expected in manifest.items():
            h, size = hashlib.sha256(), 0
            stream = archive.extractfile(name)
            for chunk in iter(lambda: stream.read(1 << 20), b''):
                h.update(chunk)
                size += len(chunk)
            if size != expected['bytes'] or h.hexdigest() != expected['sha256']:
                raise ValueError('payload mismatch: '+name)
        cp = json.load(archive.extractfile('s2/lineage/s2-continuation.json'))
        result = json.load(archive.extractfile('s2/results.json'))
        protocol = archive.extractfile('s2/PROTOCOL.json').read()
        if digest(protocol) != archive.extractfile('s2/PROTOCOL.sha256').read().decode():
            raise ValueError('frozen protocol mismatch')
        if cp['brain_identity'] != 'horizon-primary-brain' or cp['sequence'] != result['continuation_sequence']:
            raise ValueError('lineage mismatch')
        if result.get('production_promoted'):
            raise ValueError('research artifact was promoted')
        out = {'payloads_verified': len(manifest), 'sequence': cp['sequence'],
               'native_checksum': cp['checksum'], 'status': result['status'],
               'bytes': path.stat().st_size, 'sha256': digest(path.read_bytes())}
        print(json.dumps(out))
        return out


def package(work, source, output):
    if output.exists():
        raise ValueError('refuse to overwrite published artifact')
    files = {}
    for path in sorted(work.rglob('*')):
        if path.is_file() and not path.name.endswith('.lock'):
            files['s2/'+path.relative_to(work).as_posix()] = path
    # Exact source bytes needed to verify/recover S2 independently. The S2 WAL
    # itself also retains the complete original A–E1 -> S1 journal prefix.
    for name in ('s1/lineage/s1-continuation.json', 's1/lineage/events.jsonl',
                 'source/populations.json', 'source/07-E1-restarted.json'):
        files['source/'+name] = source/name
    for name in ('DESIGN.md', 'READINESS.md', 'REPORT.md', 'RESTORE.md', 'REPRESENTATION.md'):
        files[name] = DOCS/name
    files['ARTIFACT.md'] = work/'ARTIFACT.md'
    for path in sorted((DOCS/'figures').glob('*')):
        if path.is_file():
            files['figures/'+path.name] = path
    for name in ('run_s2_self_evaluation.py', 'package_s2_self_evaluation.py',
                 'analyze_s2_self_evaluation.py'):
        files['reproduce/'+name] = ROOT/'scripts'/name
    for name in ('s2_self_evaluation_test.go', 's1_internal_dynamics_test.go', 's1_final_audit_test.go'):
        files['reproduce/'+name] = ROOT/'services/ai/runtime'/name
    manifest = {name: {'bytes': p.stat().st_size, 'sha256': digest(p.read_bytes())} for name, p in files.items()}
    output.parent.mkdir(parents=True, exist_ok=True)
    with tarfile.open(output, 'w:xz', preset=6) as archive:
        for name, path in files.items():
            data = path.read_bytes()
            if len(data) != manifest[name]['bytes'] or digest(data) != manifest[name]['sha256']:
                raise ValueError('packaging input changed while writing: '+name)
            entry = tarfile.TarInfo(name)
            entry.size, entry.mode, entry.mtime = len(data), 0o600, 0
            archive.addfile(entry, io.BytesIO(data))
        data = (json.dumps(manifest, indent=2, sort_keys=True)+'\n').encode()
        entry = tarfile.TarInfo('MANIFEST.json')
        entry.size, entry.mode, entry.mtime = len(data), 0o600, 0
        archive.addfile(entry, io.BytesIO(data))
    (output.with_suffix(output.suffix+'.sha256')).write_text(f'{digest(output.read_bytes())}  {output.name}\n')
    verify(output)


if __name__ == '__main__':
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--verify', type=Path)
    p.add_argument('--work', type=Path)
    p.add_argument('--source', type=Path)
    p.add_argument('--output', type=Path)
    a = p.parse_args()
    if a.verify:
        verify(a.verify)
    elif a.work and a.source and a.output:
        package(a.work.resolve(), a.source.resolve(), a.output.resolve())
    else:
        p.error('use --verify ARCHIVE or --work DIR --source DIR --output FILE')
