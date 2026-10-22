package knowledge

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestBrainReadSnapshotIsDetachedAndCopyOnRead(t *testing.T) {
	brain:=NewBrain()
	a:=brain.Store("alpha")
	b:=brain.Store("beta")
	if a==nil || b==nil {t.Fatal("neural bootstrap failed")}
	brain.ConnectKind(a,b,RelationIsA,0.8,0.9,false)
	brain.Patterns.Learn([]NodeID{a.ID,b.ID},b.ID,0.75,0.8)

	before,err:=brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	snapshot,err:=brain.InspectReadOnly()
	if err!=nil {t.Fatal(err)}
	n,e,p:=snapshot.Counts()
	if n!=2 || e!=1 || p!=1 {t.Fatalf("unexpected snapshot counts nodes=%d synapses=%d patterns=%d",n,e,p)}
	if !bytes.Equal(before,snapshot.CanonicalBytes()) {t.Fatal("snapshot neural state differs from canonical substrate")}
	if got:=snapshot.SurfaceForNode(a.ID,"language");got!="alpha" {t.Fatalf("surface adapter changed label: %q",got)}
	if id,ok:=snapshot.NodeForSurface("beta");!ok || id!=b.ID {t.Fatalf("snapshot surface lookup returned id=%d ok=%v",id,ok)}
	if !snapshot.HasNode(a.ID) || snapshot.HasNode(999) {t.Fatal("snapshot node lookup is inconsistent")}

	units:=snapshot.NeuralUnits()
	units[0].ID=999
	units[0].Representation=append(units[0].Representation,99.0)
	edges:=snapshot.Synapses()
	edges[0].Weight=0
	patterns:=snapshot.Patterns()
	patterns[0].Members[0]=999
	patterns[0].Confidence=0
	blob:=snapshot.CanonicalBytes()
	if len(blob)>0 {blob[0]='x'}

	repeat:=snapshot.NeuralUnits()
	if repeat[0].ID==999 || reflect.DeepEqual(repeat[0].Representation,units[0].Representation) {
		t.Fatal("snapshot neural unit accessor leaked writable storage")
	}
	if snapshot.Synapses()[0].Weight==0 || snapshot.Patterns()[0].Members[0]==999 || snapshot.Patterns()[0].Confidence==0 {
		t.Fatal("snapshot synapse/pattern accessor leaked writable storage")
	}
	if !bytes.Equal(before,snapshot.CanonicalBytes()) {t.Fatal("mutable byte accessor changed stored snapshot")}
	after,err:=brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(before,after) {t.Fatal("inspecting read-only snapshot changed canonical neural substrate")}
}

func TestBrainReadSnapshotSurvivesFurtherCanonicalLearning(t *testing.T) {
	brain:=NewBrain()
	first:=brain.Store("object")
	snapshot,err:=brain.InspectReadOnly()
	if err!=nil {t.Fatal(err)}
	brain.Store("another object")
	if got:=len(snapshot.NeuralUnits());got!=1 {t.Fatalf("historical snapshot changed after learning: %d",got)}
	if n:=len(brain.Registry.Nodes());n!=2 {t.Fatalf("canonical learning failed: %d",n)}
	if snapshot.HasNode(first.ID)==false {t.Fatal("historical node disappeared")}
}

func TestBrainReadSnapshotAnnotationDoesNotMutateSource(t *testing.T) {
	brain:=NewBrain()
	at:=time.Date(2026,10,18,15,0,0,0,time.UTC)
	g,err:=brain.GroundObservationAt("foton","camera","vision",0.90,at)
	if err!=nil {t.Fatal(err)}
	snapshot,err:=brain.InspectReadOnly()
	if err!=nil {t.Fatal(err)}
	if got:=snapshot.SurfaceForNode(g.NodeID,"vision");got!="foton" {t.Fatalf("event-time adapter label lost: %q",got)}
	if id,ok:=snapshot.NodeForSurface("foton");!ok || id!=g.NodeID {t.Fatal("value-only snapshot lost surface anchor")}
	if bytes.Contains(snapshot.CanonicalBytes(),[]byte("foton")) {
		t.Fatal("lexical observation leaked into canonical neural memory")
	}
}
