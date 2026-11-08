"""Summarize the prespecified new calibration seeds and matched retention.

Usage: python3 scripts/summarize_native_phase2_followup.py EVIDENCE_DIRECTORY
Missing files or unexpected seeds fail rather than silently dropping an arm.
The t intervals describe five independent model seeds, not individual cases.
"""

import json
import math
from pathlib import Path
import statistics
import sys

ROOT = Path(sys.argv[1])
SEEDS = [3041, 3061, 3083, 3109, 3121]
PRIMARY_SEEDS = [2027, 2039, 2053, 2063, 2081]
ARMS = ["unpaired", "paired", "paired-centered"]


def interval(values):
    assert len(values) == 5
    mean = statistics.mean(values)
    half = 2.7764451051977987 * statistics.stdev(values) / math.sqrt(5)
    return {"mean": mean, "ci95": [mean - half, mean + half]}


calibration = {}
for budget in [32, 512]:
    data = {}
    for arm in ARMS:
        data[arm] = []
        for seed in SEEDS:
            path = ROOT / f"calibration-boundary-{budget}-{arm}/seed{seed}/boundary.json"
            row = json.loads(path.read_text())
            assert row["stream"] == 10800000 + seed
            assert row["metrics"]["cases"] == 512
            data[arm].append(row)
    result = {
        arm: {
            "net_return": interval([r["metrics"]["net_return"] for r in rows]),
            "regret": interval([r["regret"] for r in rows]),
        }
        for arm, rows in data.items()
    }
    for comparator in ["unpaired", "paired"]:
        result[f"centered-minus-{comparator}"] = interval([
            a["metrics"]["net_return"] - b["metrics"]["net_return"]
            for a, b in zip(data["paired-centered"], data[comparator])
        ])
    calibration[str(budget)] = result
(ROOT / "calibration-summary.json").write_text(json.dumps(calibration, indent=2) + "\n")

matched = json.loads((ROOT / "matched-retention.json").read_text())
retention = {}
for arm in ["relational", "identity", "softmax"]:
    rows = matched[arm]
    assert [r["seed"] for r in rows] == PRIMARY_SEEDS
    retention[arm] = {
        "before": interval([r["before"]["net_return"] for r in rows]),
    }
    for shift in ["longer", "reversal"]:
        key = f"after_{shift}"
        retention[arm][key] = interval([r[key]["net_return"] for r in rows])
        retention[arm][f"{shift}_paired_delta"] = interval([
            r[key]["net_return"] - r["before"]["net_return"] for r in rows
        ])
(ROOT / "retention-summary.json").write_text(json.dumps(retention, indent=2) + "\n")
print("Summarized all calibration arms and matched retention on five seeds each.")
