#!/usr/bin/env python3
"""Verify/package the saved A–E1 Brain; never train or replace it."""
import argparse
import gzip
import hashlib
import io
import json
from pathlib import Path
import tarfile

ROOT = Path(__file__).resolve().parents[1]
ARCHIVE = ROOT / "docs/research/canonical_integration_20261114/evidence/Horizon_Canonical_Continuation_E1_20261114.tar.xz"
ARCHIVE_SHA = "7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3"
PARENT = ROOT / "docs/research/canonical_learning_flexibility/evidence/Canonical_Learning_Flexibility_Evidence_20261114.tar.xz"
PARENT_SHA = "b9cda5961c5d59f5ec47fd436b39e9e4ed4cbd24f7ef93c3f0faa75a98c31dc0"
CHECKPOINT_SHA = "e73f8d34e80957dd442b1b5be81811270989e30ee50d03c783fab4dfeb137537"
JOURNAL_SHA = "ebe2ff1ed74c3adf2c0176b2aa5ab8ead09c8fbdf10c07243e724de9b8354ede"


def selected(archive, checksum, names):
    if hashlib.sha256(archive.read_bytes()).hexdigest() != checksum:
        raise ValueError("lineage archive checksum mismatch")
    payloads = {}
    with tarfile.open(archive) as source:
        manifest = json.load(source.extractfile("MANIFEST.json"))
        for key in names:
            data = source.extractfile(key).read()
            expected = manifest[key]
            if len(data) != expected["bytes"] or hashlib.sha256(data).hexdigest() != expected["sha256"]:
                raise ValueError("manifest payload mismatch: " + key)
            payloads[Path(key).name] = data
    return payloads


def verified_payloads():
    payloads = selected(ARCHIVE, ARCHIVE_SHA, ["e1-main-continuation/" + n for n in
        ("07-E1-restarted.json", "events.jsonl", "results.json", "lineage.json")])
    payloads.update(selected(PARENT, PARENT_SHA, ["flex-confirmation-20261114/populations.json"]))
    checkpoint = json.loads(payloads["07-E1-restarted.json"])
    brain = checkpoint["brain"]
    if (checkpoint["schema_version"] != 20 or brain["schema_version"] != 7
            or checkpoint["brain_identity"] != "horizon-primary-brain" or checkpoint["sequence"] != 75331
            or checkpoint["brain"]["native_control"]["updates"] != 10752
            or len(brain["neural_units"]) != 57 or len(brain["synapses"]) != 94
            or len(brain["temporal_patterns"]) != 24
            or checkpoint["checksum"] != "68a1518ebf1eea887041a339851754880d7638866a135361693205bab4bf4b82"):
        raise ValueError("unexpected saved A–E1 lineage; no fallback permitted")
    for name, digest in (("07-E1-restarted.json", CHECKPOINT_SHA), ("events.jsonl", JOURNAL_SHA)):
        if hashlib.sha256(payloads[name]).hexdigest() != digest:
            raise ValueError("canonical state SHA-256 mismatch: " + name)
    return payloads


def prepare(destination):
    if destination.exists():
        raise ValueError("destination exists; preserve prior checkpoint/journal")
    payloads = verified_payloads()
    destination.mkdir(parents=True)
    for name, data in payloads.items():
        (destination / name).write_bytes(data)
    (destination / "verified.json").write_text(json.dumps({
        "archive_sha256": ARCHIVE_SHA, "parent_archive_sha256": PARENT_SHA,
        "payload_sha256": {n: hashlib.sha256(v).hexdigest() for n, v in payloads.items()},
    }, indent=2) + "\n")


def production_bundle():
    """Recompress exact authenticated bytes for Go's standard-library embed.

    No state is converted. The production supervisor still authenticates the
    checkpoint and its committed journal before publishing a serving runtime.
    """
    payloads = verified_payloads()
    origin = json.loads(payloads["lineage.json"])
    origin.update(archive_sha256=ARCHIVE_SHA, checkpoint_sha256=CHECKPOINT_SHA,
                  journal_bytes=len(payloads["events.jsonl"]), brain_identity="horizon-primary-brain")
    files = {
        "brain_memory.json.checkpoint.json": payloads["07-E1-restarted.json"],
        "horizon_events.jsonl": payloads["events.jsonl"],
        "ae1-origin.json": (json.dumps(origin, indent=2, sort_keys=True) + "\n").encode(),
    }
    output = io.BytesIO()
    with gzip.GzipFile(filename="", mode="wb", fileobj=output, mtime=0) as compressed:
        with tarfile.open(fileobj=compressed, mode="w", format=tarfile.USTAR_FORMAT) as archive:
            for name, data in files.items():
                entry = tarfile.TarInfo(name)
                entry.size, entry.mode = len(data), 0o600
                archive.addfile(entry, io.BytesIO(data))
    return output.getvalue()


def check_production_bundle(path):
    # Compressor versions may differ; the authenticated member bytes must not.
    def members(stream):
        with tarfile.open(fileobj=stream, mode="r:gz") as archive:
            entries = archive.getmembers()
            if len(entries) != 3 or any(not e.isfile() for e in entries):
                raise ValueError("invalid production bundle members")
            return {entry.name: archive.extractfile(entry).read() for entry in entries}
    if members(io.BytesIO(path.read_bytes())) != members(io.BytesIO(production_bundle())):
        raise ValueError("production bundle differs from authenticated A–E1 archive")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--destination", type=Path)
    mode.add_argument("--production-bundle", type=Path)
    mode.add_argument("--check-production-bundle", type=Path)
    args = parser.parse_args()
    if args.destination:
        prepare(args.destination)
    elif args.production_bundle:
        with args.production_bundle.open("xb") as target:
            target.write(production_bundle())
    else:
        check_production_bundle(args.check_production_bundle)
