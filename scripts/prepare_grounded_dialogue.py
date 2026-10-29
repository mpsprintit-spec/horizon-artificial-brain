#!/usr/bin/env python3
"""Verify and unpack the opt-in trained Brain into a NEW private state directory."""

import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import shutil


def prepare(bundle: Path, state: Path) -> None:
    manifest = json.loads((bundle / "manifest.json").read_text(encoding="utf-8"))
    if manifest.get("version") != "grounded-dialogue-bundle-v1":
        raise ValueError("unsupported dialogue bundle version")
    required = {"language-brain.json", "events.jsonl", "codec.json", "evaluation.json", "start.json"}
    verified = {}
    for entry in manifest["files"]:
        source, target = entry["source"], entry["target"]
        if (not isinstance(source, str) or source in {"", ".", ".."} or Path(source).name != source
                or target not in required or target in verified):
            raise ValueError("invalid or repeated bundle path")
        size, unpacked_size = entry["bytes"], entry["unpacked_bytes"]
        if not (0 < size <= 16 * 1024 * 1024 and 0 < unpacked_size <= 16 * 1024 * 1024):
            raise ValueError("bundle file exceeds size budget")
        path = bundle / source
        if path.stat().st_size != size:
            raise ValueError(f"size mismatch: {source}")
        packed = path.read_bytes()
        if hashlib.sha256(packed).hexdigest() != entry["sha256"]:
            raise ValueError(f"checksum mismatch: {source}")
        if entry["encoding"] == "gzip":
            with gzip.GzipFile(fileobj=io.BytesIO(packed)) as stream:
                data = stream.read(unpacked_size + 1)
        elif entry["encoding"] == "identity":
            data = packed
        else:
            raise ValueError("unsupported bundle encoding")
        if len(data) != unpacked_size or hashlib.sha256(data).hexdigest() != entry["unpacked_sha256"]:
            raise ValueError(f"unpacked checksum mismatch: {source}")
        verified[target] = data
    if verified.keys() != required:
        raise ValueError("bundle is missing a required checkpoint, journal, codec or evaluation")

    # Exclusive directory creation refuses every existing directory or symlink.
    # Never merge a research checkpoint with a deployment's existing journal.
    state.mkdir(mode=0o700)
    try:
        for name, data in verified.items():
            with (state / name).open("xb") as stream:
                os.chmod(stream.name, 0o600)
                stream.write(data)
    except BaseException:
        shutil.rmtree(state)
        raise


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bundle", type=Path, default=Path(__file__).resolve().parents[1] / "models/grounded-dialogue-v1")
    parser.add_argument("--state", type=Path, required=True, help="new, nonexistent state directory; parent must exist")
    args = parser.parse_args()
    try:
        prepare(args.bundle, args.state)
    except (OSError, ValueError, KeyError, TypeError, EOFError) as error:
        parser.exit(1, f"Dialogue bundle preparation refused: {error}\n")
    print(f"Prepared verified dialogue state: {args.state}")


if __name__ == "__main__":
    main()
