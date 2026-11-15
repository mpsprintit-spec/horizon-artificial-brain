#!/usr/bin/env python3
"""Summarize independent Go experiment replicas, never individual-episode CIs."""
import argparse
import json
import math
from pathlib import Path
import statistics as st


def ci(values):
    if len(values) != 5:
        raise ValueError("sealed final design requires five independent seeds")
    mean = st.mean(values)
    half = 2.7764451051977987 * st.stdev(values) / math.sqrt(5)
    return {"mean": mean, "ci95": [mean - half, mean + half]}


def main():
    p = argparse.ArgumentParser()
    p.add_argument("evidence", type=Path)
    p.add_argument("output", type=Path)
    args = p.parse_args()
    seeds = [101, 211, 307, 401, 503]
    mc = [json.loads((args.evidence / "final-verified-mc" / f"seed{s}.json").read_text()) for s in seeds]
    td = [json.loads((args.evidence / "final-verified-td0" / f"seed{s}.json").read_text()) for s in seeds]
    out = {"seeds": seeds, "train_episodes_per_seed": 4000, "holdout_cases_per_distribution_per_seed": 256,
           "ci_method": "two-sided Student t, df=4, independent seed means; paired differences within seed",
           "greedy_evaluation": {}, "adaptation_512_episodes": {}, "paired_test_differences": {}}
    evaluations = lambda x: x["greedy_or_baseline_evaluation"]
    for split, arms in evaluations(mc[0]).items():
        out["greedy_evaluation"][split] = {
            arm: {metric: ci([evaluations(x)[split][arm][metric] for x in mc])
                  for metric in ["net_return", "accuracy", "operations", "stops", "truncations"]}
            for arm in arms}
    for arm in ["untrained", "random", "static3", "existing-think2"]:
        out["paired_test_differences"]["trained_minus_" + arm] = ci([
            evaluations(x)["test"]["trained"]["net_return"] - evaluations(x)["test"][arm]["net_return"] for x in mc])
    out["paired_test_differences"]["lambda1_minus_lambda0"] = ci([
        evaluations(x)["test"]["trained"]["net_return"] - evaluations(y)["test"]["trained"]["net_return"] for x, y in zip(mc, td)])
    for key in mc[0]["adaptation"]:
        out["adaptation_512_episodes"][key] = ci([x["adaptation"][key]["net_return"] for x in mc])
    out["stochastic_training_return"] = ci([x["stochastic_training"]["net_return"] for x in mc])
    out["parameter_changes_per_seed"] = [x["parameter_changes"] for x in mc]
    out["parameter_max_signed_stored_magnitude_delta"] = [x["parameter_max_delta"] for x in mc]
    out["world_synapses_unchanged_all_seeds"] = all(x["world_synapses_unchanged"] for x in mc + td)
    out["causal_diagnostic"] = {
        "relevant_ablation_return": ci([x["causal_diagnostic"]["relevant_world_edge_ablation"]["net_return"] for x in mc]),
        "unrelated_ablation_return": ci([x["causal_diagnostic"]["unrelated_world_edge_ablation"]["net_return"] for x in mc]),
        "credit_precision_excluding_bias": [x["causal_diagnostic"]["credit_precision_excluding_bias"] for x in mc],
        "active_distractor_updates": [x["causal_diagnostic"]["active_distractor_parameter_updates"] for x in mc],
        "inactive_updates": [x["causal_diagnostic"]["inactive_parameter_updates"] for x in mc],
        "fixed_action_distractor_reward_effect": [x["causal_diagnostic"]["fixed_action_distractor_reward_effect"] for x in mc]}
    args.output.write_text(json.dumps(out, indent=2) + "\n")
    print(json.dumps({k: v for k, v in out.items() if k not in ["greedy_evaluation"]}, indent=2))


if __name__ == "__main__":
    main()
