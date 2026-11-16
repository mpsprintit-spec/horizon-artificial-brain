#!/usr/bin/env python3
"""Package or verify the recoverable S1 research lineage, without training."""
import argparse
import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import tarfile

ROOT = Path(__file__).resolve().parents[1]
RESEARCH = ROOT / 'docs/research/s1_internal_dynamics_20261116'


def digest(data):
    return hashlib.sha256(data).hexdigest()


def package(work, source, output):
    if output.exists():
        raise ValueError('refuse to overwrite a published evidence archive')
    files = {}
    for path in sorted(work.rglob('*')):
        if path.is_file() and not path.name.endswith('.lock'):
            files['s1/' + path.relative_to(work).as_posix()] = path
    for name in ('07-E1-restarted.json', 'events.jsonl', 'populations.json',
                 'verified.json', 'lineage.json'):
        files['source/' + name] = source / name
    for name in ('PROTOCOL.md', 'PROTOCOL_INITIAL.md', 'REPORT.md'):
        files[name] = RESEARCH / name
    for name in ('run_s1_internal_dynamics.py', 'package_s1_internal_dynamics.py'):
        files['reproduce/' + name] = ROOT / 'scripts' / name
    for name in ('s1_internal_dynamics_test.go', 'native_numeric_coexistence_test.go'):
        files['reproduce/' + name] = ROOT / 'services/ai/runtime' / name
    manifest = {name: {'bytes': path.stat().st_size, 'sha256': digest(path.read_bytes())}
                for name, path in files.items()}
    output.parent.mkdir(parents=True, exist_ok=True)
    with tarfile.open(output, 'w:xz', preset=6) as archive:
        for name, path in files.items():
            data = path.read_bytes()
            entry = tarfile.TarInfo(name)
            entry.size, entry.mode, entry.mtime = len(data), 0o600, 0
            archive.addfile(entry, io.BytesIO(data))
        data = (json.dumps(manifest, indent=2, sort_keys=True) + '\n').encode()
        entry = tarfile.TarInfo('MANIFEST.json')
        entry.size, entry.mode, entry.mtime = len(data), 0o600, 0
        archive.addfile(entry, io.BytesIO(data))
    (output.with_suffix(output.suffix + '.sha256')).write_text(
        f'{digest(output.read_bytes())}  {output.name}\n')
    verify(output)


def verify(path):
    with tarfile.open(path, 'r:xz') as archive:
        members = archive.getmembers()
        names = [m.name for m in members]
        if len(names) != len(set(names)):
            raise ValueError('duplicate archive path')
        for m in members:
            if not m.isfile() or PurePosixPath(m.name).is_absolute() or '..' in PurePosixPath(m.name).parts:
                raise ValueError('unsafe archive member')
        manifest = json.load(archive.extractfile('MANIFEST.json'))
        if set(names) != set(manifest) | {'MANIFEST.json'}:
            raise ValueError('manifest does not cover all payloads')
        for name, expected in manifest.items():
            h, size = hashlib.sha256(), 0
            stream = archive.extractfile(name)
            for chunk in iter(lambda: stream.read(1 << 20), b''):
                size += len(chunk)
                h.update(chunk)
            if size != expected['bytes'] or h.hexdigest() != expected['sha256']:
                raise ValueError('payload mismatch: ' + name)
        result = json.load(archive.extractfile('s1/results.json'))
        checkpoint = json.load(archive.extractfile('s1/lineage/s1-continuation.json'))
        if checkpoint['brain_identity'] != 'horizon-primary-brain' or checkpoint['sequence'] != 75866:
            raise ValueError('unexpected published lineage')
        print(json.dumps({'verified_payloads': len(manifest), 'status': result['Status'],
                          'sequence': checkpoint['sequence'],
                          'archive_sha256': digest(path.read_bytes())}))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--verify', type=Path)
    parser.add_argument('--work', type=Path)
    parser.add_argument('--source', type=Path)
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    if args.verify:
        verify(args.verify)
    elif args.work and args.source and args.output:
        package(args.work.resolve(), args.source.resolve(), args.output.resolve())
    else:
        parser.error('use --verify ARCHIVE or --work DIR --source DIR --output FILE')
