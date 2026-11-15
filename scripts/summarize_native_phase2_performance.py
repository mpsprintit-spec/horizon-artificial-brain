import json, pathlib, re, statistics, sys
root = pathlib.Path(sys.argv[1])
raw = {}
for path in root.glob('bench-*.txt'):
    rows = {}
    for line in path.read_text().splitlines():
        if not line.startswith('Benchmark'):
            continue
        parts = line.split()
        name = re.sub(r'-[0-9]+$', '', parts[0])
        rows[name] = {'samples': int(parts[1])}
        for i in range(2, len(parts) - 1, 2):
            rows[name][parts[i+1]] = float(parts[i])
    if rows:
        raw[path.stem[6:]] = rows
out = {'raw': raw, 'paired': {}}
for label, before, after in [('controller-and-legacy', ['A1','A2'], ['B1','B2']), ('serial-diagnostic', ['serial-A1','serial-A2'], ['serial-B1','serial-B2']), ('buffer-reuse', ['buffer-A1','buffer-A2'], ['buffer-B1','buffer-B2']), ('isolated-final-legacy', ['isolated-final-A1','isolated-final-A2'], ['isolated-final-B1','isolated-final-B2'])]:
    out['paired'][label] = {}
    for name in raw[before[0]]:
        entry = {}
        for metric in raw[before[0]][name]:
            a = statistics.mean(raw[r][name][metric] for r in before)
            b = statistics.mean(raw[r][name][metric] for r in after)
            entry[metric] = {'before': a, 'after': b, 'before_divided_by_after': a/b if b else None}
        out['paired'][label][name] = entry
out['environment'] = json.loads((root/'benchmark-environment.json').read_text())
(root/'performance-summary.json').write_text(json.dumps(out, indent=2)+'\n')
for group, rows in out['paired'].items():
    for name, metrics in rows.items():
        print(group, name, {k:v for k,v in metrics.items() if k in ['B/op','allocs/op','p95_ms','think_p95_ms','episode_p95_ms']})
