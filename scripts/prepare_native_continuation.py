#!/usr/bin/env python3
"""Verify the saved PR69 Brain for integration checks; never train or replace it."""
import argparse
import hashlib
import json
from pathlib import Path
import tarfile

ROOT = Path(__file__).resolve().parents[1]
ARCHIVE = ROOT / "docs/research/canonical_integration_20261114/evidence/Horizon_Canonical_Continuation_E1_20261114.tar.xz"
ARCHIVE_SHA = "7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3"
PARENT = ROOT / "docs/research/canonical_learning_flexibility/evidence/Canonical_Learning_Flexibility_Evidence_20261114.tar.xz"
PARENT_SHA = "b9cda5961c5d59f5ec47fd436b39e9e4ed4cbd24f7ef93c3f0faa75a98c31dc0"


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


def prepare(destination):
    if destination.exists():
        raise ValueError("destination exists; preserve prior checkpoint/journal")
    payloads = selected(ARCHIVE, ARCHIVE_SHA, ["e1-main-continuation/" + n for n in
        ("07-E1-restarted.json", "events.jsonl", "results.json", "lineage.json")])
    payloads.update(selected(PARENT, PARENT_SHA, ["flex-confirmation-20261114/populations.json"]))
    checkpoint = json.loads(payloads["07-E1-restarted.json"])
    if (checkpoint["brain_identity"] != "horizon-primary-brain" or checkpoint["sequence"] != 75331
            or checkpoint["brain"]["native_control"]["updates"] != 10752
            or checkpoint["checksum"] != "68a1518ebf1eea887041a339851754880d7638866a135361693205bab4bf4b82"):
        raise ValueError("unexpected saved A–E1 lineage; no fallback permitted")
    destination.mkdir(parents=True)
    for name, data in payloads.items():
        (destination / name).write_bytes(data)
    (destination / "verified.json").write_text(json.dumps({
        "archive_sha256": ARCHIVE_SHA, "parent_archive_sha256": PARENT_SHA,
        "payload_sha256": {n: hashlib.sha256(v).hexdigest() for n, v in payloads.items()},
    }, indent=2) + "\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--destination", type=Path, required=True)
    prepare(parser.parse_args().destination)
