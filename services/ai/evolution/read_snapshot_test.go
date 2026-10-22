package evolution

import (
	"math"
	"reflect"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestSnapshotStatsMatchLegacyGraphInspection(t *testing.T) {
	brain:=knowledge.NewBrain()
	a:=brain.Store("a")
	b:=brain.Store("b")
	c:=brain.Store("c")
	d:=brain.Store("d")
	brain.ConnectKind(a,b,knowledge.RelationIsA,0.9,0.8,false)
	brain.ConnectKind(c,b,knowledge.RelationHas,0.7,0.6,false)
	brain.ConnectKind(a,c,knowledge.RelationPartWhole,0.6,0.75,false)
	brain.ConnectKind(d,b,knowledge.RelationIsA,0.4,0.7,false)
	brain.ConnectKind(b,c,knowledge.RelationAssociation,0.5,0.9,false)
	snapshot,err:=brain.InspectReadOnly()
	if err!=nil {t.Fatal(err)}
	for _,node:=range brain.Registry.Nodes() {
		legacy:=ComputeStats(brain,node)
		got:=ComputeSnapshotStats(snapshot,node.ID)
		if legacy.IsAIncoming!=got.IsAIncoming ||
			legacy.IsAOutgoing!=got.IsAOutgoing ||
			legacy.DescriptiveIncoming!=got.DescriptiveIncoming ||
			legacy.DistinctRelationKinds!=got.DistinctRelationKinds ||
			legacy.OutDegree!=got.OutDegree || legacy.InDegree!=got.InDegree ||
			math.Abs(legacy.AverageConfidence-got.AverageConfidence)>1e-12 {
			t.Fatalf("node %d stats differ: old=%+v snapshot=%+v",node.ID,legacy,got)
		}
	}
	legacyClusters:=ClusterNodes(brain)
	snapshotClusters:=ClusterSnapshot(snapshot)
	if !reflect.DeepEqual(legacyClusters,snapshotClusters) {
		t.Fatalf("read-only clustering diverged: old=%+v detached=%+v",legacyClusters,snapshotClusters)
	}
}

func TestSnapshotClusterResultsAreDetachedFromBrain(t *testing.T) {
	brain:=knowledge.NewBrain()
	for _,token:=range []string{"one","two","three","four"} {brain.Store(token)}
	snapshot,err:=brain.InspectReadOnly()
	if err!=nil {t.Fatal(err)}
	clusters:=ClusterSnapshot(snapshot)
	if len(clusters)==0 {t.Fatal("snapshot failed to form numerical clusters")}
	clusters[0].Members[0]=999
	clusters[0].Centroid[0]=999
	again:=ClusterSnapshot(snapshot)
	if !reflect.DeepEqual(again,ClusterNodes(brain)) {
		t.Fatal("changing snapshot cluster display altered stable neural clustering")
	}
}
