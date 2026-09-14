package knowledge

import (
	"errors"
	"strings"
	"time"
)

// GetOrCreateAt is the event-time variant of GetOrCreate. New neural units
// receive their initial history from the supplied event timestamp, preventing
// wall-clock creation time from entering replayed neural state.
func (r *TokenRegistry) GetOrCreateAt(token string, now time.Time) (*ConceptNode, bool, error) {
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil, false, errors.New("token is empty")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byToken[canonical]; ok {
		n := r.byID[id]
		n.Frequency++
		n.LastActivation = now
		return n, false, nil
	}

	n := newConceptNode(r.nextID, canonical)
	n.UsageHistory = []time.Time{now}
	n.LastActivation = now
	n.Frequency = 1
	r.nextID++
	r.byToken[canonical] = n.ID
	r.byID[n.ID] = n
	return n, true, nil
}

// StoreAt exposes event-time token creation through the same Brain substrate;
// it does not create another memory representation.
func (k *KnowledgeBase) StoreAt(token string, now time.Time) *ConceptNode {
	if k == nil || k.Registry == nil {
		return nil
	}
	n, _, err := k.Registry.GetOrCreateAt(strings.TrimSpace(token), now)
	if err != nil {
		return nil
	}
	return n
}
