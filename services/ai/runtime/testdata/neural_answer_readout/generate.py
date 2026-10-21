"""Freeze the synthetic ISF001 readout protocol into a NEW output directory.

Usage: python3 generate.py /path/to/ISF001.zip /path/to/new-fixtures
Only reads the source archive; never edits a Brain, journal or source file.
"""

import gzip
import hashlib
import json
import sys
import zipfile
from pathlib import Path


def main():
    archive, destination = map(Path, sys.argv[1:])
    archive_bytes = archive.read_bytes()
    assert hashlib.sha256(archive_bytes).hexdigest() == (
        "578c679a67fdd6b23b8fc766e4b2ecf7510a2ee94bd5fa57aaf59de1f7c086f2"
    ), "unexpected source archive/version"
    with zipfile.ZipFile(archive) as source:
        checkpoint = source.read("run-v2/latihan-benar-checkpoint.json")
        digest = hashlib.sha256(checkpoint).hexdigest()
        assert digest == "1d22349376aca4488f875b8f372fbddb9ea27eab7da0be08ff519e3b7961331b"
        cases = []
        for name in sorted(source.namelist()):
            if not (name.startswith("readout-check/run/normal/") and name.endswith(".json")):
                continue
            row = json.loads(source.read(name))
            cases.append({
                "id": row["Case"]["ID"],
                "split": row["Case"]["Split"],
                "events": row["InputEvents"],
                "target_population": row["ExpectedPopulation"],
                "outputs": row["Outputs"],
                "after_neural_sha256": row["AfterNeuralSHA256"],
                "legacy_answer_population": row["Interpretation"]["Answer"]["NodeIDs"],
            })
        assert len(cases) == 42
    document = {
        "schema": 1,
        "source_commit": "0b893e905742e27e3c9c4ad2a79c5c4fb4747cc0",
        "checkpoint_sha256": digest,
        "cases": cases,
    }
    raw = (json.dumps(document, ensure_ascii=False, separators=(",", ":")) + "\n").encode()
    destination.mkdir(parents=True, exist_ok=False)
    (destination / "isf001-checkpoint.json.gz").write_bytes(gzip.compress(checkpoint, mtime=0))
    (destination / "isf001-cases.json.gz").write_bytes(gzip.compress(raw, mtime=0))


if __name__ == "__main__":
    main()
