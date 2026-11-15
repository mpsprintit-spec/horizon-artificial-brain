#!/usr/bin/env python3
"""Verify and copy the saved PR68 lineage; never initialize a replacement Brain."""
import argparse
import hashlib
import json
from pathlib import Path
import tarfile

ARCHIVE_SHA256 = "b9cda5961c5d59f5ec47fd436b39e9e4ed4cbd24f7ef93c3f0faa75a98c31dc0"
PREFIX = "flex-confirmation-20261114/"
FILES = ("05-restart-continuation.json", "events.jsonl", "populations.json", "source-and-lineage.json")


def prepare(archive: Path, destination: Path) -> None:
    if hashlib.sha256(archive.read_bytes()).hexdigest() != ARCHIVE_SHA256:
        raise ValueError("PR68 archive checksum mismatch; no fallback Brain allowed")
    if destination.exists():
        raise ValueError("destination already exists; preserve the previous lineage")
    payloads = {}
    with tarfile.open(archive) as source:
        manifest = json.load(source.extractfile("MANIFEST.json"))
        for name in FILES:
            key = PREFIX + name
            data = source.extractfile(key).read()
            expected = manifest[key]
            if len(data) != expected["bytes"] or hashlib.sha256(data).hexdigest() != expected["sha256"]:
                raise ValueError("PR68 payload checksum mismatch: " + name)
            payloads[name] = data
    checkpoint = json.loads(payloads[FILES[0]])
    brain = checkpoint["brain"]
    if (checkpoint["brain_identity"] != "horizon-primary-brain" or checkpoint["sequence"] != 10817
            or checkpoint["checksum"] != "5b44d5164b126356ddd1332302864b3256ae9a1b4da6ea825584b2c4d6115561"
            or brain["native_control"]["updates"] != 1536 or len(brain["neural_units"]) != 57
            or len(brain["synapses"]) != 94 or len(brain["temporal_patterns"]) != 24):
        raise ValueError("unexpected saved PR68 lineage")
    # Only exact named manifest members are copied; no arbitrary archive paths.
    destination.mkdir(parents=True)
    for name, data in payloads.items():
        (destination / name).write_bytes(data)
    (destination / "origin.json").write_text(json.dumps({
        "archive_sha256": ARCHIVE_SHA256,
        "payload_sha256": {name: hashlib.sha256(data).hexdigest() for name, data in payloads.items()},
    }, indent=2) + "\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--archive", type=Path, default=Path(__file__).resolve().parents[1] / "docs/research/canonical_learning_flexibility/evidence/Canonical_Learning_Flexibility_Evidence_20261114.tar.xz")
    parser.add_argument("--destination", type=Path, required=True)
    args = parser.parse_args()
    prepare(args.archive, args.destination)
