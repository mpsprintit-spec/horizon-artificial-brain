#!/usr/bin/env python3
"""Sequential Go benchmark runner; raw outputs and Linux wait4 RSS/CPU retained.

Build baseline-runtime.test, perf-runtime.test and native-runtime.test with the
identical foundation_benchmark_test.go before running. Do not run training or
other repository tests concurrently. This script never changes a Brain.
"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import time


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("evidence", type=Path)
    args = parser.parse_args()
    root = args.evidence.resolve()
    runs = []

    def run(label, binary, bench, count, extra=()):
        command = [str(root / binary), "-test.run=^$", "-test.bench=" + bench,
                   "-test.benchmem", f"-test.benchtime={count}x", "-test.cpu=8",
                   "-test.timeout=20m", *extra]
        start = time.monotonic()
        with (root / (label + ".txt")).open("w") as output:
            child = subprocess.Popen(command, stdout=output, stderr=subprocess.STDOUT)
            _, status, usage = os.wait4(child.pid, 0)
            child.returncode = os.waitstatus_to_exitcode(status)
        result = dict(label=label, command=command, exit_code=child.returncode,
                      wall_seconds=time.monotonic() - start,
                      user_seconds=usage.ru_utime, system_seconds=usage.ru_stime,
                      peak_rss_kib=usage.ru_maxrss, voluntary_switches=usage.ru_nvcsw,
                      involuntary_switches=usage.ru_nivcsw)
        runs.append(result)
        (root / "performance-runs.json").write_text(json.dumps(runs, indent=2))
        print(label, result, flush=True)
        if child.returncode:
            raise SystemExit(child.returncode)

    for i, arm in enumerate(["baseline", "perf", "perf", "baseline"]):
        run(f"paired-{i + 1}-{arm}", arm + "-runtime.test",
            "^BenchmarkFoundationTransitions$", 100)
    run("native-final-bench", "native-runtime.test", "^BenchmarkNativeSubstrate$", 50)
    run("persistence-final-bench", "perf-runtime.test", "^BenchmarkFoundationPersistence$", 5)
    for arm in ["baseline", "perf"]:
        prefix = root / (arm + "-profile")
        run(arm + "-profile", arm + "-runtime.test",
            "^BenchmarkFoundationTransitions$/nodes=40000/journal=false$", 200,
            ["-test.cpuprofile=" + str(prefix) + ".cpu",
             "-test.memprofile=" + str(prefix) + ".heap",
             "-test.mutexprofile=" + str(prefix) + ".mutex",
             "-test.blockprofile=" + str(prefix) + ".block"])


if __name__ == "__main__":
    main()
