package evolution

import (
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// ComputeSnapshotStats is the value-only counterpart to ComputeStats. It
// inspects a detached neural read snapshot, never *knowledge.Brain or its
// mutable registry. Counts include parallel synapses, matching legacy stats.
func ComputeSnapshotStats(snapshot knowledge.BrainReadSnapshot,target knowledge.NodeID) NodeStats {
	var stats NodeStats
	kinds:=make(map[knowledge.RelationKind]bool)
	fromIsA:=make(map[knowledge.NodeID]bool)
	fromDescriptive:=make(map[knowledge.NodeID]bool)
	var confSum float64
	var confCount int
	for _,s:=range snapshot.Synapses() {
		if s.SourceID==target {
			stats.OutDegree++
			kinds[s.Kind]=true
			confSum+=s.Confidence
			confCount++
			if s.Kind==knowledge.RelationIsA {stats.IsAOutgoing++}
		}
		if s.TargetID==target && s.SourceID!=target {
			stats.InDegree++
			kinds[s.Kind]=true
			confSum+=s.Confidence
			confCount++
			switch s.Kind {
			case knowledge.RelationIsA:
				if !fromIsA[s.SourceID] {stats.IsAIncoming++;fromIsA[s.SourceID]=true}
			case knowledge.RelationHas,knowledge.RelationPartWhole:
				if !fromDescriptive[s.SourceID] {stats.DescriptiveIncoming++;fromDescriptive[s.SourceID]=true}
			}
		}
	}
	stats.DistinctRelationKinds=len(kinds)
	if confCount>0 {stats.AverageConfidence=confSum/float64(confCount)}
	return stats
}

// ClusterSnapshot uses the same threshold, statistics and centroid update as
// legacy ClusterNodes, but consumes detached value-only graph data. Clusters
// remain observations of numerical connectivity, not a second knowledge store.
func ClusterSnapshot(snapshot knowledge.BrainReadSnapshot) []Cluster {
	nodes:=snapshot.NeuralUnits()
	sort.Slice(nodes,func(i,j int)bool{return nodes[i].ID<nodes[j].ID})
	var clusters []Cluster
	for _,node:=range nodes {
		v:=vector(ComputeSnapshotStats(snapshot,node.ID))
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
