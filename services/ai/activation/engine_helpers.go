package activation

import (
	"sort"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func temporalPenalty(now, last time.Time) float64 {
	if last.IsZero() { return 0.7 }
	days := now.Sub(last).Hours() / 24
	if days <= 1 { return 1 }
	return clamp(1-(days*0.01), 0.35, 1)
}

func normalize(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	for k, v := range in { in[k] = squash(v) }
	return in
}

func squash(v float64) float64 {
	if v <= 1 { return clamp(v, 0, 1) }
	return 1 + (v-1)/v
}

func clamp(v, low, high float64) float64 {
	if v < low { return low }
	if v > high { return high }
	return v
}

func cloneState(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	out := make(map[knowledge.NodeID]float64, len(in))
	for id, value := range in { out[id] = value }
	return out
}

func sortedNodeIDs(in map[knowledge.NodeID]float64) []knowledge.NodeID {
	ids := make([]knowledge.NodeID, 0, len(in))
	for id := range in { ids = append(ids, id) }
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func stateDifference(a, b map[knowledge.NodeID]float64) float64 {
	keys := make(map[knowledge.NodeID]struct{}, len(a)+len(b))
	for id := range a { keys[id] = struct{}{} }
	for id := range b { keys[id] = struct{}{} }
	if len(keys) == 0 { return 0 }
	ids := make([]knowledge.NodeID, 0, len(keys))
	for id := range keys { ids = append(ids, id) }
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var total float64
	for _, id := range ids { total += abs(a[id] - b[id]) }
	return clamp01(total / float64(len(ids)))
}

func abs(v float64) float64 {
	if v < 0 { return -v }
	return v
}

func clamp01(v float64) float64 { return clamp(v, 0, 1) }