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
