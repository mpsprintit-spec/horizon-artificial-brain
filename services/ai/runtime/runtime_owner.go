package runtime

import (
	"errors"
	"fmt"
	"sync"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// RuntimeOwner is the transport-neutral owner of Horizon's single active
// intelligence. Every adapter using this owner must execute against its active
// BrainRuntime through WithRuntime; callbacks MUST NOT retain that pointer.
//
// The RWMutex admits concurrent routed requests and prevents a recovery
// handover until every in-flight routed request has finished. A replacement
// runtime is constructed and verified in isolation while the owner holds the
// exclusive lock. The previous runtime is then retired and the new one becomes
// active in a single critical section.
//
// RuntimeOwner does not make direct external access to a leaked *BrainRuntime
// safe. All service adapters must route their access through this boundary.
type RuntimeOwner struct {
	mu         sync.RWMutex
	current    *BrainRuntime
	journal    *EventLog
	generation uint64
	changed    chan struct{}
}

// NewRuntimeOwner binds one existing runtime to the owner. When journal is
// nonnil, it must already be attached to the supplied runtime. This prevents
// silently switching the active brain to a different event stream.
func NewRuntimeOwner(initial *BrainRuntime, journal *EventLog) (*RuntimeOwner, error) {
	if initial == nil || initial.brain == nil {
		return nil, errors.New("active brain runtime is not initialized")
	}
	initial.mu.Lock()
	defer initial.mu.Unlock()
	if initial.retired {
		return nil, ErrRuntimeRetired
	}
	if initial.eventLog != journal {
		return nil, errors.New("runtime event log does not match owner journal")
	}
	return &RuntimeOwner{
		current: initial,
		journal: journal,
		generation: 1,
		changed: make(chan struct{}),
	}, nil
}

// WithRuntime holds a read lease until fn returns, preventing handover while
// a routed operation is running. Never call a recovery method inside fn.
func (o *RuntimeOwner) WithRuntime(fn func(*BrainRuntime) error) error {
	if o == nil { return errors.New("runtime owner is not initialized") }
	if fn == nil { return errors.New("runtime callback is nil") }
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.current == nil { return errors.New("active runtime is unavailable") }
	return fn(o.current)
}

// WithRuntimeGeneration additionally supplies a signal that closes when the
// active runtime is replaced. Useful for transport streams that must stop
// subscribing to the retired runtime after the callback releases its lease.
func (o *RuntimeOwner) WithRuntimeGeneration(fn func(*BrainRuntime, <-chan struct{}) error) error {
	if o == nil { return errors.New("runtime owner is not initialized") }
	if fn == nil { return errors.New("runtime callback is nil") }
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.current == nil { return errors.New("active runtime is unavailable") }
	return fn(o.current, o.changed)
}

func (o *RuntimeOwner) Generation() uint64 {
	if o == nil { return 0 }
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.generation
}

// RecoverFromCheckpoint pauses all owner-routed operations, verifies the
// anchored checkpoint against the full committed journal and replays the
// suffix into a fresh runtime. On ANY error the active pointer is unchanged.
func (o *RuntimeOwner) RecoverFromCheckpoint(checkpointPath string) error {
	if o == nil { return errors.New("runtime owner is not initialized") }
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.journal == nil { return errors.New("runtime owner has no event journal") }
	fresh, err := RebuildRuntimeFromCheckpointAndJournal(o.journal, checkpointPath)
	if err != nil { return fmt.Errorf("rebuild runtime before handover: %w", err) }
	return o.handoverLocked(fresh)
}

// RecoverTrailingIntentFromCheckpoint additionally repairs a single verified
// orphan sensor intent, preserving a durable backup, before rebuilding the
// runtime. If reconstruction fails, the old runtime remains active (and,
// if already poisoned, remains blocked). The backup path is returned even
// when reconstruction fails, so an operator can inspect the prior journal.
func (o *RuntimeOwner) RecoverTrailingIntentFromCheckpoint(checkpointPath string) (string, error) {
	if o == nil { return "", errors.New("runtime owner is not initialized") }
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.journal == nil { return "", errors.New("runtime owner has no event journal") }
	backup, err := o.journal.RecoverTrailingSensorIntent()
	if err != nil { return backup, fmt.Errorf("repair trailing sensor intent: %w", err) }
	fresh, err := RebuildRuntimeFromCheckpointAndJournal(o.journal, checkpointPath)
	if err != nil { return backup, fmt.Errorf("rebuild after repaired journal: %w", err) }
	return backup, o.handoverLocked(fresh)
}

// RecoverFromGenesis is the fallback when no checkpoint is available. This
// is isolated from the serving runtime and fully replays the committed log.
func (o *RuntimeOwner) RecoverFromGenesis(factory func() *knowledge.Brain) error {
	if o == nil { return errors.New("runtime owner is not initialized") }
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.journal == nil { return errors.New("runtime owner has no event journal") }
	fresh, err := RebuildRuntimeFromCommittedJournal(o.journal, factory)
	if err != nil { return fmt.Errorf("rebuild genesis before handover: %w", err) }
	return o.handoverLocked(fresh)
}

// handoverLocked never invokes fallible I/O after retiring the old runtime.
// Both runtime mutexes prevent racing direct core mutations that honor the
// runtime write boundary. The owner mutex excludes owner-routed calls.
func (o *RuntimeOwner) handoverLocked(fresh *BrainRuntime) error {
	if o.current == nil || fresh == nil || fresh == o.current {
		return errors.New("handover requires a distinct initialized runtime")
	}
	old := o.current
	old.mu.Lock()
	defer old.mu.Unlock()
	fresh.mu.Lock()
	defer fresh.mu.Unlock()
	if old.retired { return ErrRuntimeRetired }
	if old.eventLog != o.journal {
		return errors.New("current runtime journal changed outside the owner")
	}
	if fresh.eventLog != nil || fresh.retired || fresh.recoveryRequired || fresh.brain == nil {
		return errors.New("replacement runtime is not an isolated healthy brain")
	}
	old.retired = true
	old.eventLog = nil
	fresh.eventLog = o.journal
	o.current = fresh
	o.generation++
	close(o.changed)
	o.changed = make(chan struct{})
	return nil
}
