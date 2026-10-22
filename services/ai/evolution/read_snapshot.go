package evolution

import (
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type snapshotStatsAccum struct {
	stats NodeStats
	kinds map[knowledge.RelationKind]bool
	fromIsA map[knowledge.NodeID]bool
	fromDescriptive map[knowledge.NodeID]bool
	sum float64
	count int
}

// buildSnapshotStats scans synapses once, O(nodes + edges), rather than
// recomputing graph-wide incoming connectivity for each neural unit.
func buildSnapshotStats(snapshot knowledge.BrainReadSnapshot) map[knowledge.NodeID]NodeStats {
	acc:=make(map[knowledge.NodeID]*snapshotStatsAccum)
	ensure:=func(id knowledge.NodeID)*snapshotStatsAccum {
		if a:=acc[id];a!=nil {return a}
		a:=&snapshotStatsAccum{
			kinds:make(map[knowledge.RelationKind]bool),
			fromIsA:make(map[knowledge.NodeID]bool),
			fromDescriptive:make(map[knowledge.NodeID]bool),
		}
		acc[id]=a
		return a
	}
	for _,edge:=range snapshot.Synapses() {
		out:=ensure(edge.SourceID)
		out.stats.OutDegree++
		out.kinds[edge.Kind]=true
		out.sum+=edge.Confidence
		out.count++
		if edge.Kind==knowledge.RelationIsA {out.stats.IsAOutgoing++}
		if edge.SourceID==edge.TargetID {continue}
		in:=ensure(edge.TargetID)
		in.stats.InDegree++
		in.kinds[edge.Kind]=true
		in.sum+=edge.Confidence
		in.count++
		switch edge.Kind {
		case knowledge.RelationIsA:
			if !in.fromIsA[edge.SourceID] {in.fromIsA[edge.SourceID]=true;in.stats.IsAIncoming++}
		case knowledge.RelationHas,knowledge.RelationPartWhole:
			if !in.fromDescriptive[edge.SourceID] {in.fromDescriptive[edge.SourceID]=true;in.stats.DescriptiveIncoming++}
		}
	}
	result:=make(map[knowledge.NodeID]NodeStats,len(acc))
	for id,a:=range acc {
		a.stats.DistinctRelationKinds=len(a.kinds)
		if a.count>0 {a.stats.AverageConfidence=a.sum/float64(a.count)}
		result[id]=a.stats
	}
	return result
}

// ComputeSnapshotStats inspects detached values rather than live registry
// pointers. Legacy ComputeStats is retained for external compatibility.
func ComputeSnapshotStats(snapshot knowledge.BrainReadSnapshot,target knowledge.NodeID) NodeStats {
	return buildSnapshotStats(snapshot)[target]
}

// ClusterSnapshot is the snapshot-only version of ClusterNodes. Its sorted
// node order, thresholds and centroid averaging match the old algorithm.
func ClusterSnapshot(snapshot knowledge.BrainReadSnapshot) []Cluster {
	nodes:=snapshot.NeuralUnits()
	stats:=buildSnapshotStats(snapshot)
	sort.Slice(nodes,func(i,j int)bool{return nodes[i].ID<nodes[j].ID})
	var clusters []Cluster
	for _,node:=range nodes {
		v:=vector(stats[node.ID])
		bestIdx:=-1
		bestDist:=ClusterThreshold
		for i,cl:=range clusters {
			d:=distance(v,cl.Centroid)
			if d<bestDist {bestDist=d;bestIdx=i}
		}
		if bestIdx<0 {
			clusters=append(clusters,Cluster{ID:len(clusters)+1,Members:[]knowledge.NodeID{node.ID},Centroid:v})
			continue
		}
		cl:=&clusters[bestIdx]
		n:=float64(len(cl.Members))
		for i:=range cl.Centroid {
			cl.Centroid[i]=(cl.Centroid[i]*n+v[i])/(n+1)
		}
		cl.Members=append(cl.Members,node.ID)
	}
	return clusters
}
