package knowledge

// NativeAdmissionState is consistency metadata, never a second neural state or
// a policy readout. It avoids copying an episode, activation or learned weights.
type NativeAdmissionState string

const (
	NativeUninitialized   NativeAdmissionState = "uninitialized"
	NativeIdle            NativeAdmissionState = "idle"
	NativeEpisodeActive   NativeAdmissionState = "episode-active"
	NativeAwaitingOutcome NativeAdmissionState = "awaiting-outcome"
	NativeContrastPending NativeAdmissionState = "contrast-pending"
)

func (k *KnowledgeBase) NativeAdmission() NativeAdmissionState {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.NativeControl
	if s == nil {
		return NativeUninitialized
	}
	if s.PendingContrast != nil {
		return NativeContrastPending
	}
	if s.Episode != nil {
		if s.Episode.Stopped {
			return NativeAwaitingOutcome
		}
		return NativeEpisodeActive
	}
	return NativeIdle
}

// NativeReservedNodesLocked returns a transient ADDRESS fence for the existing
// policy circuit in N. Call under the Brain lock. World learning must not apply
// its incompatible confidence/weight law to these signed policy parameters or
// recruit an effector as a sensory relay. Only native observed-credit updates
// own those parameters. There are no copied neural values or persistent caches.
// Nil preserves the original production hot path when native is not enabled.
func (k *KnowledgeBase) NativeReservedNodesLocked() map[NodeID]bool {
	if s := k.BrainState.NativeControl; s != nil {
		return s.ReservedNodes()
	}
	return nil
}

// NativeOwnership is a transient address-set accelerator, not neural state.
// Bounds avoid hashing every world edge when the policy cells lie outside it;
// the canonical role addresses remain authoritative for sparse/noncontiguous imported IDs.
type NativeOwnership struct {
	state       *NativeControlState
	First, Last NodeID
}

func (o NativeOwnership) Enabled() bool { return o.state != nil }
func (o NativeOwnership) Has(id NodeID) bool {
	if o.state == nil || id < o.First || id > o.Last {
		return false
	}
	if id == o.state.Bias {
		return true
	}
	for _, v := range o.state.Features {
		if id == v {
			return true
		}
	}
	for _, group := range o.state.Outputs {
		for _, v := range group {
			if id == v {
				return true
			}
		}
	}
	return false
}

// Caller holds the Brain lock throughout use. Only canonical role addresses
// are read; no map, weights, episode or second neural representation is copied.
func (k *KnowledgeBase) NativeOwnershipLocked() NativeOwnership {
	o := NativeOwnership{state: k.BrainState.NativeControl}
	if o.state == nil {
		return o
	}
	o.First, o.Last = o.state.Bias, o.state.Bias
	add := func(id NodeID) {
		if id < o.First {
			o.First = id
		}
		if id > o.Last {
			o.Last = id
		}
	}
	for _, id := range o.state.Features {
		add(id)
	}
	for _, group := range o.state.Outputs {
		for _, id := range group {
			add(id)
		}
	}
	return o
}
