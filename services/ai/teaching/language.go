// Package teaching implements HNL/1, an explicit numerical teaching language.
// Syntax, references and provenance are engineering scaffolding. A relation
// opcode never computes an answer; its meaning must be learned by the Brain.
package teaching

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

const Version uint16 = 1
const ExtendedVersion uint16 = 2
const MaxPacketBytes = 1 << 20

type Kind uint8

const (
	Teach  Kind = 1
	Ask    Kind = 2
	Answer Kind = 3
)

type Relation uint8

const (
	Sign       Relation = 1
	Greater    Relation = 2
	SameSign   Relation = 3
	SumGreater Relation = 4 // A+B > C+D, supplied as a question, never evaluated here.
	SameChange Relation = 5 // Same nonzero sign for A-B and C-D.
	Equal      Relation = 6 // Equality at the learned measurement resolution.
	SumEqual   Relation = 7 // Equality of A+B and C+D at learned resolution.
)

// Ref addresses a measured coordinate, not a pre-existing knowledge neuron.
// Lag zero is the latest observation; channel numbering is zero based.
type Ref struct{ Lag, Channel uint8 }
type Question struct {
	Domain     uint16
	Relation   Relation
	A, B, C, D Ref
}

func refLess(a, b Ref) bool { return a.Lag < b.Lag || a.Lag == b.Lag && a.Channel < b.Channel }
func (q Question) Canonical() Question {
	if (q.Relation == SameSign || q.Relation == Equal) && refLess(q.B, q.A) {
		q.A, q.B = q.B, q.A
	}
	if q.Relation == SumGreater || q.Relation == SumEqual {
		if refLess(q.B, q.A) {
			q.A, q.B = q.B, q.A
		}
		if refLess(q.D, q.C) {
			q.C, q.D = q.D, q.C
		}
		if q.Relation == SumEqual && (refLess(q.C, q.A) || q.C == q.A && refLess(q.D, q.B)) {
			q.A, q.B, q.C, q.D = q.C, q.D, q.A, q.B
		}
	}
	if q.Relation == SameChange && (refLess(q.C, q.A) || q.C == q.A && refLess(q.D, q.B)) {
		q.A, q.B, q.C, q.D = q.C, q.D, q.A, q.B
	}
	return q
}
func (q Question) Validate() error {
	if q.Domain == 0 || q.Relation < Sign || q.Relation > SumEqual {
		return errors.New("unsupported HNL question or reference")
	}
	for _, r := range []Ref{q.A, q.B, q.C, q.D} {
		if r.Lag >= 64 || r.Channel >= 6 {
			return errors.New("unsupported HNL reference")
		}
	}
	if q.Relation == Sign && q.B != (Ref{}) {
		return errors.New("unary HNL question has a second argument")
	}
	if (q.Relation <= SameSign || q.Relation == Equal) && (q.C != (Ref{}) || q.D != (Ref{})) {
		return errors.New("HNL binary question has extra arguments")
	}
	return nil
}

type Role uint8

const (
	Fit   Role = 1
	Check Role = 2
	Probe Role = 3
)

type Sample struct {
	Lag     uint8
	Present bool
	Values  [6]float64
}
type Evidence struct {
	Value      float64
	ObservedAt int64
}
type Example struct {
	Role       Role
	Episode    uint64
	ObservedAt int64 // Unix nanoseconds: time of the latest input observation.
	Samples    []Sample
	Outcome    *Evidence // Never permitted on an Ask packet.
}
type Unit struct {
	Node                           uint64
	Code, Resolution, Contribution float64
}
type Trial struct {
	Kind        uint8
	Node        uint64
	Error       float64
	Regressions uint16
	Eligible    bool
}
type Response struct {
	Sequence         uint64
	Status           uint8 // 1 untrained, 2 uncertain, 3 numeric prediction, 4 teaching receipt.
	Value, CheckRMSE float64
	Units            []Unit
	LearnedN         uint16
	Cause            uint8 // 0 none, 1 evidence, 2 readout, 3 resolution, 4 direction,
	// 5 boundary, 6 relation, 7 conflicting observations.
	Committed                         bool
	BeforeError, AfterError           float64
	Grown                             uint64
	ResolutionBefore, ResolutionAfter float64
	Trials                            []Trial
}
type Packet struct {
	Version            uint16
	Kind               Kind
	ID, Source, Stream uint64 // provenance only; never neural input features.
	At                 int64
	Question           Question
	Examples           []Example
	Reply              *Response
}

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
func (p Packet) Validate() error {
	if p.Version != Version && p.Version != ExtendedVersion || p.Kind < Teach || p.Kind > Answer || p.ID == 0 || p.Source == 0 || p.Stream == 0 || p.At <= 0 {
		return errors.New("invalid HNL envelope")
	}
	if err := p.Question.Validate(); err != nil {
		return err
	}
	if p.Version == Version && p.Question.Relation > SameSign {
		return errors.New("four-reference questions require HNL/2")
	}
	if p.Kind == Answer {
		if len(p.Examples) != 0 || p.Reply == nil {
			return errors.New("invalid HNL answer payload")
		}
		r := p.Reply
		if r.Status < 1 || r.Status > 4 || !finite(r.Value) || math.Abs(r.Value) > 1 || !finite(r.CheckRMSE) || r.CheckRMSE < 0 || r.LearnedN > 32 || r.Cause > 7 || len(r.Units) > 32 || len(r.Trials) > 64 {
			return errors.New("invalid HNL numeric response")
		}
		for _, v := range []float64{r.BeforeError, r.AfterError, r.ResolutionBefore, r.ResolutionAfter} {
			if !finite(v) || v < 0 {
				return errors.New("invalid HNL diagnosis")
			}
		}
		seen := map[uint64]bool{}
		for _, u := range r.Units {
			if u.Node == 0 || seen[u.Node] || !finite(u.Code) || !finite(u.Resolution) || u.Resolution <= 0 || !finite(u.Contribution) {
				return errors.New("invalid HNL N contribution")
			}
			seen[u.Node] = true
		}
		for _, t := range r.Trials {
			if t.Kind < 2 || t.Kind > 6 || !finite(t.Error) || t.Error < 0 {
				return errors.New("invalid HNL trial")
			}
		}
		return nil
	}
	if p.Reply != nil || len(p.Examples) == 0 || len(p.Examples) > 128 {
		return errors.New("invalid HNL observation count")
	}
	if p.Kind == Ask && len(p.Examples) != 1 {
		return errors.New("HNL Ask needs exactly one unlabelled example")
	}
	counts := map[Role]int{}
	episodes := map[uint64]Role{}
	for _, e := range p.Examples {
		if e.Episode == 0 || e.ObservedAt <= 0 || e.ObservedAt > p.At || len(e.Samples) == 0 || len(e.Samples) > 64 {
			return errors.New("invalid HNL episode or observation time")
		}
		if _, ok := episodes[e.Episode]; ok {
			return errors.New("HNL episodes must be unique, including across fit/check")
		}
		episodes[e.Episode] = e.Role
		if p.Kind == Ask {
			if e.Role != Probe || e.Outcome != nil {
				return errors.New("HNL Ask cannot contain an answer or feedback")
			}
		} else if (e.Role != Fit && e.Role != Check) || e.Outcome == nil || math.Abs(e.Outcome.Value) != 1 || e.Outcome.ObservedAt < e.ObservedAt || e.Outcome.ObservedAt > p.At {
			return errors.New("HNL teaching requires an already observed signed outcome")
		}
		counts[e.Role]++
		seen := map[uint8]bool{}
		for _, s := range e.Samples {
			if s.Lag >= 64 || seen[s.Lag] {
				return errors.New("HNL has duplicate or excessive lags")
			}
			seen[s.Lag] = true
			for _, v := range s.Values {
				if !finite(v) || math.Abs(v) > 1 || !s.Present && v != 0 {
					return errors.New("HNL measurements must be finite bounded values; missing rows carry no value")
				}
			}
		}
	}
	if p.Kind == Teach && (counts[Fit] < 4 || counts[Fit] > 64 || counts[Check] < 4 || counts[Check] > 64) {
		return fmt.Errorf("HNL lesson requires 4..64 fit and check episodes, got %d/%d", counts[Fit], counts[Check])
	}
	return nil
}

// canonicalPacket copies before ordering; callers retain ownership of all data.
func canonicalPacket(p Packet) Packet {
	p.Question = p.Question.Canonical()
	p.Examples = append([]Example(nil), p.Examples...)
	for i := range p.Examples {
		p.Examples[i].Samples = append([]Sample(nil), p.Examples[i].Samples...)
		for j := range p.Examples[i].Samples {
			for k, v := range p.Examples[i].Samples[j].Values {
				if v == 0 {
					p.Examples[i].Samples[j].Values[k] = 0
				}
			}
		}
		sort.Slice(p.Examples[i].Samples, func(a, b int) bool { return p.Examples[i].Samples[a].Lag < p.Examples[i].Samples[b].Lag })
	}
	return p
}
