package runtime

import (
	"errors"
	"fmt"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// BrainInspection is a detached read-only projection of one committed runtime
// revision. Its graph carries values rather than pointers into the live Brain.
// Mutating returned slices has no effect on Horizon's canonical substrate.
type BrainInspection struct {
	Revision uint64
	Graph knowledge.BrainReadSnapshot
}

// InspectBrain captures the current runtime revision and neural inspection
// view while holding the runtime's serialization lock. It must not be used
// inside a callback that has already acquired r.mu.
func (r *BrainRuntime) InspectBrain() (BrainInspection,error) {
	if r==nil || r.brain==nil {return BrainInspection{},errors.New("brain runtime is not initialized")}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err:=r.ensureWritableLocked();err!=nil{return BrainInspection{},err}
	graph,err:=r.brain.InspectReadOnly()
	if err!=nil{return BrainInspection{},err}
	return BrainInspection{Revision:r.seq,Graph:graph},nil
}

// ResolveNodeLabels copies adapter labels for the indicated node IDs from
// one runtime revision. It never exports Brain/Registry/node pointers.
// This is intentionally cheaper than creating a complete neural snapshot
// on every cognition cycle.
func (r *BrainRuntime) ResolveNodeLabels(ids []knowledge.NodeID,modality string) ([]string,error) {
	if r==nil || r.brain==nil {return nil,errors.New("brain runtime is not initialized")}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err:=r.ensureWritableLocked();err!=nil{return nil,err}
	out:=make([]string,0,len(ids))
	seen:=make(map[string]struct{},len(ids))
	for _,id:=range ids {
		if r.brain.Registry.GetByID(id)==nil {continue}
		label:=r.brain.SurfaceForNode(id,modality)
		if label=="" {label=fmt.Sprintf("node:%d",id)}
		if _,exists:=seen[label];exists {continue}
		seen[label]=struct{}{}
		out=append(out,label)
	}
	return out,nil
}
