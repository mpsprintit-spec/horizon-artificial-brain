package knowledge

import "time"

// NativeRuntimeView contains only command preflight metadata. It is neither a
// checkpoint nor a learned-parameter store. Large neural decision maps stay in N;
// their existing digest is computed under the same read lock only when needed.
// A distinct type prevents accidentally serializing an incomplete checkpoint.
type NativeRuntimeView struct {
	Config          NativeControlConfig
	Bias            NodeID
	Features        []NodeID
	Outputs         [][]NodeID
	Episode         *NativeEpisodeInfo
	PendingContrast *NativePendingInfo
}
type NativeEpisodeInfo struct {
	ID                                         string
	Training, Intervention, Stopped, Truncated bool
	Started, LastDecisionAt                    time.Time
	DecisionCount                              int
	TraceDigest                                string
}
type NativePendingInfo struct {
	ID                            string
	Mode, MatchKey, InitialDigest string
	Episode                       *NativeEpisodeInfo
}

func nativeEpisodeInfo(ep *NativeEpisode, digest bool) *NativeEpisodeInfo {
	if ep == nil {
		return nil
	}
	out := &NativeEpisodeInfo{ID: ep.ID, Training: ep.Training, Intervention: ep.Intervention, Stopped: ep.Stopped, Truncated: ep.Truncated, Started: ep.Started, DecisionCount: len(ep.Decisions)}
	if len(ep.Decisions) > 0 {
		out.LastDecisionAt = ep.Decisions[len(ep.Decisions)-1].At
	}
	if digest {
		out.TraceDigest = NativeDigest(ep)
	}
	return out
}
func (k *KnowledgeBase) NativeControlRuntimeView(digest bool) *NativeRuntimeView {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.NativeControl
	if s == nil {
		return nil
	}
	out := &NativeRuntimeView{Config: s.Config, Bias: s.Bias, Features: append([]NodeID(nil), s.Features...), Outputs: make([][]NodeID, len(s.Outputs)), Episode: nativeEpisodeInfo(s.Episode, digest)}
	out.Config.Sources = append([]NodeID(nil), s.Config.Sources...)
	out.Config.Operations = append([]NativeOperation(nil), s.Config.Operations...)
	for i, g := range s.Outputs {
		out.Outputs[i] = append([]NodeID(nil), g...)
	}
	if s.PendingContrast != nil {
		out.PendingContrast = &NativePendingInfo{ID: s.PendingContrast.ID, Episode: nativeEpisodeInfo(s.PendingContrast.Episode, false)}
		out.PendingContrast.Mode = s.PendingContrast.Mode
		if ep := s.PendingContrast.Episode; ep != nil {
			out.PendingContrast.MatchKey = ep.MatchKey
			if len(ep.Decisions) > 0 {
				out.PendingContrast.InitialDigest = ep.Decisions[0].Before
			}
		}
	}
	return out
}
