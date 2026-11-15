#!/usr/bin/env python3
"""Summarize the fixed Gate G ABBA run without pooling run-level percentiles.

Reads the runner's raw files; writes explicit comparisons and resource tables.
An incomplete or failed run can be inspected with --partial, never called passed.
"""
import argparse
from collections import defaultdict
import json
from pathlib import Path
import re
from statistics import mean


def parse(path):
    rows = {}
    for line in path.read_text().splitlines():
        match = re.match(r'^(Benchmark\S+)\s+(\d+)\s+(.+)$', line)
        if not match:
            continue
        name, count, rest = match.groups()
        tokens = rest.split()
        values = {}
        for value, unit in zip(tokens[::2], tokens[1::2]):
            try:
                values[unit] = float(value)
            except ValueError:
                raise ValueError(f'{path}: invalid benchmark metric {value!r} {unit!r}')
        rows[name] = {'samples': int(count), 'metrics': values}
    return rows


def aggregate(groups):
    by_name = defaultdict(list)
    for rows in groups:
        for name, row in rows.items():
            by_name[name].append(row)
    return {name: {
        'runs': len(rows),
        'samples_per_run': [row['samples'] for row in rows],
        'mean_run_metrics': {key: mean(row['metrics'][key] for row in rows)
                             for key in rows[0]['metrics']},
        'individual_runs': rows,
    } for name, rows in by_name.items()}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('directory', type=Path)
    p.add_argument('--partial', action='store_true')
    args = p.parse_args()
    meta_path = args.directory / 'runs.json'
    meta = json.loads(meta_path.read_text())
    recorded = {row['label']: row for row in meta['ordered_runs']}
    expected_labels = [f'{run}-{part}' for run in ('A1', 'B1', 'B2', 'A2')
                       for part in ('thought', 'context-continuity', 'native', 'append', 'persistence')]
    expected_labels += ['candidate-native-idle-thought', 'candidate-native-idle-context-continuity',
                        'candidate-admission', 'candidate-handoff',
                        meta.get('baseline_diagnostic_label', 'baseline-mixed-refusal')]
    missing = [label for label in expected_labels if label not in recorded]
    failed = [row for row in recorded.values()
              if (row['returncode'] == 0) != (row['expected_exit'] == 0)]
    if (missing or failed) and not args.partial:
        raise SystemExit(f'incomplete/failed confirmation: missing={missing}, failed={failed}')
    parsed = {label: parse(args.directory / (label + '.txt')) for label in recorded}
    base = aggregate([rows for label, rows in parsed.items() if label.startswith(('A1-', 'A2-'))])
    cand = aggregate([rows for label, rows in parsed.items() if label.startswith(('B1-', 'B2-'))])
    special = aggregate([rows for label, rows in parsed.items() if label.startswith('candidate-')])
    comparisons = []

    def compare(name, other, metric, factor, allowance, kind, values):
        if other not in base or metric not in base[other]['mean_run_metrics']:
            raise ValueError(f'missing baseline metric {other}:{metric}')
        bval = base[other]['mean_run_metrics'][metric]
        cval = values['mean_run_metrics'][metric]
        limit = bval * factor + allowance
        comparisons.append(dict(kind=kind, benchmark=name, baseline_benchmark=other,
                                metric=metric, baseline=bval, candidate=cval,
                                factor=factor, additive_allowance=allowance,
                                limit=limit, passed=cval <= limit,
                                candidate_runs=values['runs'], baseline_runs=base[other]['runs']))

    for name, values in cand.items():
        if name.startswith(('BenchmarkGateGThought/', 'BenchmarkGateGContextContinuity/')):
            compare(name, name, 'thought_p95_ms', 1.15, .10, 'production-thought', values)
            compare(name, name, 'B/op', 1.05, 4096, 'production-thought', values)
        elif name.startswith('BenchmarkGateGNativeStages/'):
            for metric in ('operation_p95_ms', 'learning_p95_ms'):
                compare(name, name, metric, 1.20, .10, 'native-stages', values)
            compare(name, name, 'B/op', 1.10, 4096, 'native-pair-allocation', values)
        elif name.startswith('BenchmarkNativeActionPersistence/'):
            # This unchanged historical benchmark reports a three-operation mean,
            # not individual tails. Preserve that limitation explicitly.
            compare(name, name, 'ns/op', 1.20, 100000, 'persistence-mean-descriptive', values)
            compare(name, name, 'B/op', 1.10, 4096, 'persistence-allocation', values)
    for name, values in special.items():
        other = name.replace('BenchmarkGateGThoughtNativeIdle/', 'BenchmarkGateGThought/')
        other = other.replace('BenchmarkGateGContextContinuityNativeIdle/', 'BenchmarkGateGContextContinuity/')
        if other != name:
            compare(name, other, 'thought_p95_ms', 1.15, .10, 'native-idle-thought', values)
            compare(name, other, 'B/op', 1.05, 4096, 'native-idle-thought', values)
        elif name.startswith('BenchmarkGateGAdmission'):
            for metric, limit in [('admission_p95_ms', .01), ('B/op', 512)]:
                value = values['mean_run_metrics'][metric]
                comparisons.append(dict(kind='admission', benchmark=name, metric=metric,
                                        candidate=value, limit=limit, passed=value <= limit))

    failures = [row for row in comparisons if not row['passed']]
    result = dict(metadata=meta, incomplete_labels=missing, unexpected_exits=failed,
                  complete=not missing and not failed,
                  observed_budget_pass=not missing and not failed and not failures,
                  baseline=base, candidate=cand, candidate_only=special,
                  comparisons=comparisons, failed_comparisons=failures,
                  limits=[
                      'Means of per-run percentiles are compared; observations are not pooled.',
                      'Native stage tails use 20 pairs/run and are descriptive.',
                      'Persistence uses three operations/run and reports means, not p95/p99.',
                      'RSS is cumulative process peak; GC statistics also include untimed seeding.',
                      'No baseline latency exists for mixed handoff: baseline rejects it.',
                      'This statistical resource check alone cannot certify the other Gate G requirements.',
                  ])
    (args.directory / 'summary.json').write_text(json.dumps(result, indent=2) + '\n')
    lines = ['# Gate G performance confirmation', '',
             f"Complete: **{result['complete']}**. Passed locked observed comparisons: "
             f"**{result['observed_budget_pass']}** ({len(comparisons)-len(failures)}/{len(comparisons)}).", '',
             'Values are means of per-run measurements. Raw samples/runs and every failed comparison remain in summary.json and the original logs.', '',
             '| Workload | Metric | Baseline | Candidate | Limit | Result |',
             '|---|---|---:|---:|---:|---|']
    for row in comparisons:
        lines.append(f"| {row['benchmark']} | {row['metric']} | {row.get('baseline', float('nan')):.6g} | "
                     f"{row['candidate']:.6g} | {row['limit']:.6g} | {'PASS' if row['passed'] else 'FAIL'} |")
    lines += ['', '## Descriptive resources', '',
              '| Revision | Workload | p50 ms | p95 ms | p99 ms | B/op | allocs/op | Peak RSS MB | GC count/cell | GC pause ms/cell |',
              '|---|---|---:|---:|---:|---:|---:|---:|---:|---:|']
    for group_name, group in [('baseline', base), ('candidate', cand), ('candidate-only', special)]:
        for name, row in group.items():
            m = row['mean_run_metrics']
            prefix = next((x for x in ('thought', 'pair', 'handoff', 'admission', 'append')
                           if x + '_p50_ms' in m), None)
            show = lambda key: f'{m[key]:.6g}' if key in m else '—'
            tails = [show(f'{prefix}_p{p}_ms') for p in (50, 95, 99)] if prefix else ['—'] * 3
            lines.append('| ' + ' | '.join([group_name, name] + tails +
                         [show(k) for k in ('B/op', 'allocs/op', 'process_peak_RSS_MB', 'cell_GCs', 'cell_GC_pause_ms')]) + ' |')
    lines += ['', '## Interpretation limits', ''] + ['- ' + s for s in result['limits']]
    (args.directory / 'SUMMARY.md').write_text('\n'.join(lines) + '\n')
    print(json.dumps(dict(complete=result['complete'], passed=result['observed_budget_pass'],
                          comparisons=len(comparisons), failures=len(failures), missing=missing)))


if __name__ == '__main__':
    main()
