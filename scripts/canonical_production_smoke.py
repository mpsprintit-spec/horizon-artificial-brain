#!/usr/bin/env python3
"""Exercise the shipped executable from an empty cwd, HTTP input and restart.

No fixture environment variable or source directory is available to the child.
Only disposable deployment state receives one observation; no training runs.
"""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import tempfile
import time
from urllib.error import URLError
from urllib.request import Request, urlopen


def smoke(binary):
    with tempfile.TemporaryDirectory(prefix="horizon-production-") as temporary:
        cwd = Path(temporary)
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
        base = f"http://127.0.0.1:{port}"
        env = {k: v for k, v in os.environ.items() if not k.startswith("HORIZON_")}
        env.update(HORIZON_BRIDGE_ADDRESS=f"127.0.0.1:{port}",
                   HORIZON_AI_COGNITIVE_INTERVAL="24h", HORIZON_BRIDGE_TOKEN="production-regression")

        def request(path, body=None):
            req = Request(base + path, data=None if body is None else json.dumps(body).encode(),
                          headers={"Authorization": "Bearer production-regression", "Content-Type": "application/json"})
            with urlopen(req, timeout=20) as response:
                return json.load(response)

        def start(name):
            log = (cwd / name).open("w")
            process = subprocess.Popen([str(binary)], cwd=cwd, env=env, stdout=log, stderr=subprocess.STDOUT)
            deadline = time.monotonic() + 90
            while time.monotonic() < deadline:
                if process.poll() is not None:
                    log.close()
                    raise AssertionError((cwd / name).read_text())
                try:
                    health = request("/health")
                    records = [json.loads(line) for line in (cwd / name).read_text().splitlines()]
                    ready = next((r for r in records if r["msg"] == "Horizon brain ready"), None)
                    if ready and any(r["msg"] == "Horizon cognitive cycle" for r in records):
                        return process, log, ready, health
                except (URLError, TimeoutError, ValueError):
                    pass
                time.sleep(.1)
            process.kill()
            process.wait()
            log.close()
            raise AssertionError("production startup timed out: " + (cwd / name).read_text())

        def stop(process, log):
            process.send_signal(signal.SIGTERM)
            try:
                process.wait(timeout=30)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
                raise
            finally:
                log.close()
            assert process.returncode == 0, process.returncode

        process, log, first, _ = start("first.log")
        try:
            assert first["sequence"] == 75331 and first["stimulus_count"] == 0, first
            before = request("/v1/brain/snapshot")
            observed = request("/v1/observations", {
                "schema_version": 1, "brain_identity": "horizon-primary-brain", "event_id": "production-smoke-observation",
                "session_id": "production-smoke", "sequence": 1, "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
                "modality": "sensor", "source": "production-regression",
                "provenance": {"source": "production-regression", "modality": "sensor", "capture_device": "ci", "adapter": "ci", "synthetic": True},
                "payload": {"bytes": [1, 2, 3], "kind": "raw-sensor"},
            })
            assert observed["state_revision"] > before["state_revision"] >= 75331
        finally:
            stop(process, log)
        state = cwd / "horizon-state"
        origin = json.loads((state / "ae1-origin.json").read_text())
        journal = state / "horizon_events.jsonl"
        prefix = journal.read_bytes()[:origin["journal_bytes"]]
        assert hashlib.sha256(prefix).hexdigest() == origin["journal_sha256"]
        memory = json.loads((state / "brain_memory.json").read_text())
        assert memory["sequence"] == observed["state_revision"]
        assert memory["brain"]["native_control"]["updates"] == 10752
        process, log, restarted, _ = start("restart.log")
        try:
            assert restarted["sequence"] == observed["state_revision"], restarted
            assert restarted["stimulus_count"] == 0
            after = request("/v1/brain/snapshot")
            assert after["state_revision"] > observed["state_revision"]
        finally:
            stop(process, log)
        events = [json.loads(line) for line in journal.read_text().splitlines()]
        assert sum(e.get("event", {}).get("id") == "production-smoke-observation" for e in events) == 1
        assert not any(e.get("event", {}).get("id") == "bootstrap" for e in events)
        assert hashlib.sha256(journal.read_bytes()[:origin["journal_bytes"]]).hexdigest() == origin["journal_sha256"]
        print(json.dumps({"startup_sequence": first["sequence"], "observation_sequence": observed["state_revision"],
                          "restart_sequence": restarted["sequence"], "continued_sequence": after["state_revision"],
                          "native_updates_preserved": 10752, "bootstrap_events": 0, "original_journal_prefix_preserved": True}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    smoke(parser.parse_args().binary.resolve(strict=True))
