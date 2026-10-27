package teaching

import (
	"errors"
	"fmt"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	brainruntime "github.com/project-horizon/horizon-core/services/ai/runtime"
)

// QuestionCode encodes the AST injectively in the supported bounds. Opcodes
// distinguish questions; they do not execute Sign, Greater or SameSign.
func QuestionCode(q Question) ([]float64, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	q = q.Canonical()
	v := make([]float64, 16)
	v[0] = -.75
	v[1] = float64(q.Domain) / 65536
	v[2] = float64(q.Relation) / 8
	v[3] = float64(q.A.Lag) / 64
	v[4] = float64(q.A.Channel) / 8
	v[5] = float64(q.B.Lag) / 64
	v[6] = float64(q.B.Channel) / 8
	v[7] = float64(q.C.Lag) / 64
	v[8] = float64(q.C.Channel) / 8
	v[9] = float64(q.D.Lag) / 64
	v[10] = float64(q.D.Channel) / 8
	return v, nil
}

func scaffoldScene(at time.Time) knowledge.CompositionScene {
	// Legacy runtime envelope only. Its constants carry no teaching target.
	return knowledge.CompositionScene{ObservedAt: at, Point: []float64{0, 0, 1}, Patches: []knowledge.CompositionPatch{{Values: []float64{0, 0, 0, -1, 0}, XY: []float64{0, 0}}}}
}

// compileExample never reads Outcome. Lag and channel references remain visible
// in the query; raw measured coordinates retain their native temporal order.
func compileExample(q Question, e Example) (knowledge.CompositionInput, error) {
	v, err := QuestionCode(q)
	if err != nil {
		return knowledge.CompositionInput{}, err
	}
	maxLag := uint8(0)
	for _, s := range e.Samples {
		if s.Lag > maxLag {
			maxLag = s.Lag
		}
	}
	seq := make([][]float64, int(maxLag)+1)
	for i := range seq {
		seq[i] = make([]float64, 7)
	}
	for _, s := range e.Samples {
		row := seq[int(maxLag-s.Lag)]
		copy(row, s.Values[:])
		if s.Present {
			row[6] = 1
		}
	}
	in := knowledge.CompositionInput{Words: [][]float64{v}, Scene: scaffoldScene(time.Unix(0, e.ObservedAt).UTC()), Sequence: seq}
	return in, in.Validate()
}

func CompileAsk(p Packet) (knowledge.CompositionInput, error) {
	if err := p.Validate(); err != nil {
		return knowledge.CompositionInput{}, err
	}
	if p.Kind != Ask {
		return knowledge.CompositionInput{}, errors.New("HNL request is not Ask")
	}
	return compileExample(p.Question, p.Examples[0])
}

func CompileLesson(p Packet) (knowledge.CompositionExposure, error) {
	if err := p.Validate(); err != nil {
		return knowledge.CompositionExposure{}, err
	}
	if p.Kind != Teach {
		return knowledge.CompositionExposure{}, errors.New("HNL request is not Teach")
	}
	batch := &knowledge.NResolutionLesson{}
	for _, e := range p.Examples {
		in, err := compileExample(p.Question, e)
		if err != nil {
			return knowledge.CompositionExposure{}, err
		}
		o := knowledge.NObservation{Input: in, Outcome: e.Outcome.Value, ObservedAt: time.Unix(0, e.Outcome.ObservedAt).UTC(), EpisodeID: fmt.Sprintf("hnl:%d:%d:%d", p.Source, p.Stream, e.Episode)}
		if e.Role == Fit {
			batch.Fit = append(batch.Fit, o)
		} else {
			batch.Check = append(batch.Check, o)
		}
	}
	// V5 consumes the explicit signed numerical outcomes. The legacy contact
	// fields below are a no-op and never express a trading or motor action.
	first := batch.Fit[0].Input
	exp := knowledge.CompositionExposure{ID: fmt.Sprintf("hnl:%d:%d:%d", p.Source, p.Stream, p.ID), Source: fmt.Sprintf("hnl:%d", p.Source), EpisodeID: fmt.Sprintf("%d", p.ID), IndependenceGroup: fmt.Sprintf("hnl-stream:%d", p.Stream), CausalGroup: "teacher-labelled-numeric-observations", Reliability: 1, Input: first, Motor: []float64{0, 0, 0, 0}, After: scaffoldScene(time.Unix(0, p.At).UTC()), Resolution: batch}
	return exp, exp.Validate()
}

type Bridge struct {
	Runtime *brainruntime.BrainRuntime
	// A trusted simulator clock may be supplied for accelerated replay. A
	// packet cannot advance this clock merely by claiming a later timestamp.
	Clock brainruntime.Clock
}

func causeCode(s string) uint8 {
	switch s {
	case "insufficient-predictive-evidence":
		return 1
	case "readout":
		return 2
	case "resolution":
		return 3
	case "missing-direction":
		return 4
	case "missing-boundary":
		return 5
	case "missing-relation":
		return 6
	case "conflicting-observations":
		return 7
	}
	return 0
}

// Exchange is bidirectional: validated binary lesson/question in, detached
// binary diagnosis/prediction out. No parsing path computes a target value.
func (b Bridge) Exchange(data []byte) ([]byte, error) {
	p, err := Unmarshal(data)
	if err != nil {
		return nil, err
	}
	if b.Runtime == nil {
		return nil, errors.New("HNL runtime unavailable")
	}
	now := time.Now()
	if b.Clock != nil {
		now = b.Clock.Now()
	}
	if p.At > now.UnixNano() {
		return nil, errors.New("HNL packet is ahead of the trusted observation clock")
	}
	if p.Kind == Answer {
		return nil, errors.New("HNL answers cannot be submitted as lessons")
	}
	cfg, err := b.Runtime.CompositionStatus()
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.Config.Version != knowledge.NeuralCompositionResolutionV5 {
		return nil, errors.New("HNL/1 requires the explicitly enabled numerical V5 research contract")
	}
	r := &Response{}
	if p.Kind == Teach {
		e, err := CompileLesson(p)
		if err != nil {
			return nil, err
		}
		seq, receipt, err := b.Runtime.LearnCompositionWithDiagnosis(e, time.Unix(0, p.At).UTC())
		if err != nil {
			return nil, err
		}
		if receipt == nil {
			return nil, errors.New("HNL numerical teaching receipt unavailable after commit")
		}
		d := receipt.Diagnosis
		r.Sequence = seq
		r.Status = 4
		r.LearnedN = uint16(receipt.LearnedNUnits)
		r.Cause = causeCode(d.Cause)
		r.Committed = d.Committed
		r.BeforeError = d.BeforeError
		r.AfterError = d.AfterError
		r.Grown = uint64(d.Grown)
		r.ResolutionBefore = d.ResolutionBefore
		r.ResolutionAfter = d.ResolutionAfter
		for _, v := range d.Trials {
			r.Trials = append(r.Trials, Trial{Kind: causeCode(v.Kind), Node: uint64(v.Feature), Error: v.CheckError, Regressions: uint16(v.RetentionRegressions), Eligible: v.Eligible})
		}
	} else {
		in, err := CompileAsk(p)
		if err != nil {
			return nil, err
		}
		out, err := b.Runtime.ReadComposition(in, time.Unix(0, p.At).UTC())
		if err != nil {
			return nil, err
		}
		r.Sequence = out.Sequence
		r.Status = 1
		if out.Status == "uncertain-numerical-state" {
			r.Status = 2
		}
		if out.Accepted {
			r.Status = 3
		}
		if n := out.NResolution; n != nil {
			r.Value = n.Value
			r.CheckRMSE = n.ObservedCheckRMSE
			for _, u := range n.Units {
				r.Units = append(r.Units, Unit{Node: uint64(u.Node), Code: u.Code, Resolution: u.Resolution, Contribution: u.Contribution})
			}
		}
	}
	return Marshal(Packet{Version: p.Version, Kind: Answer, ID: p.ID, Source: p.Source, Stream: p.Stream, At: p.At, Question: p.Question, Reply: r})
}
