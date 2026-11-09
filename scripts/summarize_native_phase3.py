#!/usr/bin/env python3
"""Summarize sealed Phase III experiments; never train or select models."""
import json
import math
from pathlib import Path
import statistics
import sys

root = Path(sys.argv[1])
out = {"source": "matched factual action contrasts", "experiments": {}}

def ci(values):
    if len(values) != 5:
        raise ValueError("confirmation requires five independent seeds")
    mean = statistics.mean(values)
    half = 2.776445 * statistics.stdev(values) / math.sqrt(5)
    return {"mean": mean, "ci95": [mean-half, mean+half], "seed_values": values}

for name, prefix in [("one_step", "confirmation-512"), ("delayed", "delayed-confirmation")]:
    experiment = {"raw": {}, "summary": {}, "paired_differences": {}}
    for noise in ["noisy", "clean"]:
        raw = {}
        summary = {}
        for arm in ["action", "feature", "unpaired", "unmatched"]:
            rows = [json.loads(f.read_text()) for f in sorted((root / f"{prefix}-{arm}-{noise}").glob("seed*.json"))]
            if len(rows) != 5:
                raise ValueError(f"incomplete {prefix}-{arm}-{noise}")
            raw[arm] = rows
            summary[arm] = {
                "return": ci([r["After"]["net_return"] for r in rows]),
                **{m: statistics.mean(r["After"][m] for r in rows) for m in ["accuracy", "operations", "stops", "truncations"]},
                **{m: statistics.mean(r[m] for r in rows) for m in ["regret", "advantage_rmse", "effect_precision", "effect_recall", "nuisance_effect", "signal_effect_error", "parameter_changes"]},
                "before_return": statistics.mean(r["Before"]["net_return"] for r in rows),
                "intervention_training_return": statistics.mean(r["randomized_intervention_training"]["net_return"] for r in rows),
                "baselines": {b: statistics.mean(r["baselines"][b]["net_return"] for r in rows) for b in rows[0]["baselines"]},
                "task_groups": {g: {m: statistics.mean(r["task_groups"][g][m] for r in rows) for m in ["net_return", "accuracy", "operations"]} for g in rows[0].get("task_groups", {})}
            }
        differences = {}
        for control in ["unpaired", "feature", "unmatched"]:
            if [r["seed"] for r in raw["action"]] != [r["seed"] for r in raw[control]]:
                raise ValueError("unmatched seed order")
            differences[control] = ci([a["After"]["net_return"]-b["After"]["net_return"] for a,b in zip(raw["action"],raw[control])])
        experiment["raw"][noise] = raw
        experiment["summary"][noise] = summary
        experiment["paired_differences"][noise] = differences
    a = experiment["summary"]["noisy"]["action"]
    diffs = experiment["paired_differences"]
    experiment["preregistered_scoped_criteria"] = {
        "noisy_behavior_superiority": all(diffs["noisy"][c]["mean"] >= .01 and diffs["noisy"][c]["ci95"][0] > 0 for c in ["unpaired", "feature"]),
        "regret_and_trajectory_accuracy": a["regret"] <= .002 and a["accuracy"] >= .95,
        "effect_precision_recall_rmse": a["effect_precision"] >= .95 and a["effect_recall"] >= .95 and a["advantage_rmse"] <= .01,
        "clean_noninferiority": diffs["clean"]["unpaired"]["ci95"][0] >= -.002,
        "note": "Technical contract checks are recorded separately. These are not passes of full Gates C/D/E/G/H."
    }
    out["experiments"][name] = experiment

out["benchmarks"] = []
for file in sorted(root.glob("bench-*.txt")):
    for line in file.read_text().splitlines():
        if not line.startswith("Benchmark"):
            continue
        fields = line.split()
        if len(fields) < 4 or not fields[1].isdigit():
            continue
        metrics = {fields[i+1]: float(fields[i]) for i in range(2,len(fields)-1,2)}
        if metrics.get("ns/op", 0):
            metrics["operations_per_second"] = 1e9 / metrics["ns/op"]
        out["benchmarks"].append({"file": file.name,"name":fields[0],"iterations":int(fields[1]),"metrics":metrics})
print(json.dumps(out, indent=2))
