package knowledge

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

// NeuralUnitRead is a detached VALUE projection of a unit. It contains no
// *NeuralUnit, synapse pointer, or write-capable Brain/Registry reference.
type NeuralUnitRead struct {
	ID NodeID
	Representation []float64
	Activation float64
	Frequency int64
	Plasticity float64
	Importance float64
}

type SynapseRead struct {
	SourceID NodeID
	TargetID NodeID
	Kind RelationKind
	Weight float64
	Confidence float64
	Inhibitory bool
}

type PatternRead struct {
	ID PatternID
	Members []NodeID
	Result NodeID
	Weight float64
	Confidence float64
	Frequency int64
}

// BrainReadSnapshot is an ephemeral inspection result, NOT another neural
// substrate. All collections are private; public methods return fresh copies.
// No method can mutate canonical Brain state, even after a runtime handover.
type BrainReadSnapshot struct {
	canonical []byte
	units []NeuralUnitRead
	synapses []SynapseRead
	patterns []PatternRead
	annotations []SurfaceAnnotation
}

// CanonicalBytes returns a detached numerical substrate encoding. Callers
// may modify these returned bytes without affecting the snapshot or Brain.
func (s BrainReadSnapshot) CanonicalBytes() []byte {
	return append([]byte(nil),s.canonical...)
}
func (s BrainReadSnapshot) NeuralUnits() []NeuralUnitRead {
	out:=make([]NeuralUnitRead,len(s.units))
	for i,unit:=range s.units {
		out[i]=unit
		out[i].Representation=append([]float64(nil),unit.Representation...)
	}
	return out
}
func (s BrainReadSnapshot) Synapses() []SynapseRead {
	return append([]SynapseRead(nil),s.synapses...)
}
func (s BrainReadSnapshot) Patterns() []PatternRead {
	out:=make([]PatternRead,len(s.patterns))
	for i,pattern:=range s.patterns {
		out[i]=pattern
		out[i].Members=append([]NodeID(nil),pattern.Members...)
	}
	return out
}
func (s BrainReadSnapshot) Counts() (units,synapses,patterns int) {
	return len(s.units),len(s.synapses),len(s.patterns)
}
func (s BrainReadSnapshot) HasNode(id NodeID) bool {
	i:=sort.Search(len(s.units),func(i int)bool{return s.units[i].ID>=id})
	return i<len(s.units)&&s.units[i].ID==id
}
func (s BrainReadSnapshot) SurfaceForNode(id NodeID,modality string) string {
	best:=""
	var seen time.Time
	for _,a:=range s.annotations {
		if modality!="" && CanonicalSurfaceModality(modality)!=CanonicalSurfaceModality(a.Modality) {continue}
		if a.NodeID!=id {
			found:=false
			for _,member:=range a.Population {
				if member==id {found=true;break}
			}
			if !found {continue}
		}
		if a.Surface!="" && (best=="" || a.LastSeen.After(seen)) {
			best=a.Surface
			seen=a.LastSeen
		}
	}
	return best
}
func (s BrainReadSnapshot) NodeForSurface(surface string) (NodeID,bool) {
	surface=strings.TrimSpace(strings.ToLower(surface))
	var id NodeID
	var seen time.Time
	bestLanguage:=false
	for _,a:=range s.annotations {
		if strings.ToLower(strings.TrimSpace(a.Surface))!=surface || a.NodeID==0 || !s.HasNode(a.NodeID) {continue}
		isLanguage:=a.Modality=="language"
		if id==0 || (isLanguage && !bestLanguage) || (isLanguage==bestLanguage && a.LastSeen.After(seen)) {
			id=a.NodeID
			seen=a.LastSeen
			bestLanguage=isLanguage
		}
	}
	return id,id!=0
}

// InspectReadOnly constructs detached, typed copies from the already
// canonicalized numerical substrate and separately copied adapter annotations.
// Callers that require a consistent revision must invoke this while holding
// the owning BrainRuntime lock; the knowledge layer cannot revoke arbitrary
// direct mutation aliases that were published by legacy code.
func (k *KnowledgeBase) InspectReadOnly() (BrainReadSnapshot,error) {
	if k==nil {return BrainReadSnapshot{},errors.New("canonical brain is nil")}
	data,err:=k.CanonicalJSON()
	if err!=nil {return BrainReadSnapshot{},err}
	var decoded struct {
		NeuralUnits []canonicalNeuralUnit `json:"neural_units"`
		Synapses []canonicalSynapse `json:"synapses"`
		TemporalPatterns []*PatternSynapse `json:"temporal_patterns"`
	}
	if err:=json.Unmarshal(data,&decoded);err!=nil{return BrainReadSnapshot{},err}
	out:=BrainReadSnapshot{
		canonical: data,
		units:make([]NeuralUnitRead,0,len(decoded.NeuralUnits)),
		synapses:make([]SynapseRead,0,len(decoded.Synapses)),
		patterns:make([]PatternRead,0,len(decoded.TemporalPatterns)),
		annotations:k.SurfaceAnnotationsSnapshot(),
	}
	for _,unit:=range decoded.NeuralUnits {
		out.units=append(out.units,NeuralUnitRead{
			ID:unit.ID,Representation:append([]float64(nil),unit.Representation...),
			Activation:unit.Activation,Frequency:unit.Frequency,Plasticity:unit.Plasticity,Importance:unit.Importance,
		})
	}
	sort.Slice(out.units,func(i,j int)bool{return out.units[i].ID<out.units[j].ID})
	for _,edge:=range decoded.Synapses {
		out.synapses=append(out.synapses,SynapseRead{
			SourceID:edge.SourceID,TargetID:edge.TargetID,Kind:edge.Kind,
			Weight:edge.Weight,Confidence:edge.Confidence,Inhibitory:edge.Inhibitory,
		})
	}
	for _,p:=range decoded.TemporalPatterns {
		if p==nil {continue}
		out.patterns=append(out.patterns,PatternRead{
			ID:p.ID,Members:append([]NodeID(nil),p.Members...),Result:p.Result,
			Weight:p.Weight,Confidence:p.Confidence,Frequency:p.Frequency,
		})
	}
	return out,nil
}
