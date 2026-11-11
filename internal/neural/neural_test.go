package neural

import (
	"path/filepath"
	"testing"
)

func TestNetworkLearnsWithoutSymbolicSemantics(t *testing.T) {
	n, err := New(DefaultConfig()); if err != nil { t.Fatal(err) }
	a,_:=n.AddNeuron(); b,_:=n.AddNeuron(); if _,err=n.Connect(a,b,0.2);err!=nil{t.Fatal(err)}
	for i:=0;i<20;i++ { if err:=n.Inject(a,2);err!=nil{t.Fatal(err)}; n.Tick(); n.Learn(1) }
	var max float64
	for _,s:=range n.SynapsesFrom(a){ if s.Weight>max{max=s.Weight} }
	if max <= 0.2 { t.Fatalf("expected learned weight > initial, got %v",max) }
}

func TestTemporalEligibilityPersistsAcrossSteps(t *testing.T) {
	n,_:=New(DefaultConfig()); a,_:=n.AddNeuron(); b,_:=n.AddNeuron(); sid,_:=n.Connect(a,b,0.1)
	_ = n.SetActivity(a,1); n.Tick(); n.SetActivity(a,0); n.SetActivity(b,1); n.Tick()
	s,_:=n.Synapse(sid); if s.Eligibility <= 0 { t.Fatalf("expected eligibility trace, got %v",s.Eligibility) }
}

func TestPersistenceRoundTrip(t *testing.T) {
	n,_:=New(DefaultConfig()); a,_:=n.AddNeuron(); b,_:=n.AddNeuron(); n.Connect(a,b,0.4); n.Inject(a,2); n.Tick();
	p:=filepath.Join(t.TempDir(),"brain.json"); if err:=n.Save(p);err!=nil{t.Fatal(err)}
	m,err:=Load(p);if err!=nil{t.Fatal(err)}
	if m.NeuronCount()!=n.NeuronCount() || m.SynapseCount()!=n.SynapseCount() || m.StepCount()!=n.StepCount(){t.Fatalf("state mismatch after restore")}
}

func TestInvalidConnectionsRejected(t *testing.T) {
	n,_:=New(DefaultConfig()); a,_:=n.AddNeuron(); if _,err:=n.Connect(a,99,1);err!=ErrInvalidID{t.Fatalf("expected ErrInvalidID, got %v",err)}
}
