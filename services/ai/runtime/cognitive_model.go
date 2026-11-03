package runtime

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"strings"
	"time"
)

const coordinatedJournalSchemaVersion = 13
const coordinatedCheckpointSchemaVersion = 19

const EventTypeCognitiveModel = "cognitive_model_command"
const cognitiveModelJournalSchemaVersion = 5
const cognitiveModelCheckpointSchemaVersion = 15
const neuralMetaCheckpointSchemaVersion = 16
const neuralMetaJournalSchemaVersion = 6
const terminalReadoutJournalSchemaVersion = 7
const componentLearningJournalSchemaVersion = 8
const phase2JournalSchemaVersion = 9
const phase2CheckpointSchemaVersion = 17
const phase3JournalSchemaVersion = 10
const phase4JournalSchemaVersion = 11
const terminalExposureJournalSchemaVersion = 12
const phase4CheckpointSchemaVersion = 18

type CognitiveModelCommand struct {
	Coordinated    *knowledge.CoordinatedBatchV5     `json:"coordinated,omitempty"`
	Phase4Init     *knowledge.Phase4SpecV5           `json:"phase4_init,omitempty"`
	Phase4Training *knowledge.Phase4BatchV5          `json:"phase4_training,omitempty"`
	Phase3Training *knowledge.CognitiveTrainingBatch `json:"phase3_training,omitempty"`
	Phase2Init     *knowledge.Phase2SpecV5           `json:"phase2_init,omitempty"`
	Phase2Training *knowledge.Phase2BatchV5          `json:"phase2_training,omitempty"`
	Calibration    *knowledge.CognitiveTrainingBatch `json:"calibration,omitempty"`
	ID             string                            `json:"id"`
	Initialize     *knowledge.CognitiveModelConfig   `json:"initialize,omitempty"`
	Input          *knowledge.CognitiveInput         `json:"input,omitempty"`
	Training       *knowledge.CognitiveTrainingBatch `json:"training,omitempty"`
	Search         []knowledge.BMPSEvaluation        `json:"search,omitempty"`
	Evidence       *knowledge.CognitiveEvidence      `json:"evidence,omitempty"`
}

func (c CognitiveModelCommand) Validate() error {
	if c.ID == "" {
		return errors.New("cognitive command requires ID")
	}
	n := 0
	if c.Coordinated != nil {
		n++
	}
	if c.Phase4Init != nil {
		n++
		if e := c.Phase4Init.Validate(); e != nil {
			return e
		}
	}
	if c.Phase4Training != nil {
		n++
		if p := c.Phase4Training.Protocol; p != "" && p != knowledge.Phase4TerminalProtocolV5 {
			return errors.New("unknown Phase IV training protocol")
		}
		for _, ep := range c.Phase4Training.Episodes {
			if ep.Acquisition != nil && c.Phase4Training.Protocol == "" {
				return errors.New("terminal acquisition cannot use a historical payload")
			}
		}
	}
	if c.Phase3Training != nil {
		n++
		b := c.Phase3Training
		if !b.TerminalReadoutOnly || b.LearnPolicy || b.ComponentV4 != "" || b.EntropyWeightV4 != nil {
			return errors.New("phase3 supports verified terminal readout adaptation only")
		}
	}
	if c.Phase2Init != nil {
		n++
		if err := c.Phase2Init.Validate(); err != nil {
			return err
		}
	}
	if c.Phase2Training != nil {
		n++
	}
	if c.Calibration != nil {
		n++
	}
	if c.Initialize != nil {
		n++
		if err := c.Initialize.Validate(); err != nil {
			return err
		}
	}
	if c.Input != nil {
		n++
	}
	if c.Training != nil {
		n++
	}
	if len(c.Search) > 0 {
		n++
		if c.Evidence == nil {
			return errors.New("policy search requires verified return evidence")
		}
		if err := c.Evidence.Validate(); err != nil {
			return err
		}
		if c.Evidence.Digest != knowledge.CognitiveDigest(c.Search) {
			return errors.New("search digest mismatch")
		}
		for _, e := range c.Search {
			if err := e.Validate(); err != nil {
				return err
			}
		}
	} else if c.Evidence != nil {
		return errors.New("foreign cognitive evidence payload")
	}
	if n != 1 {
		return errors.New("cognitive command needs exactly one operation")
	}
	return nil
}

type CognitiveModelResult struct {
	Coordinated    *knowledge.CoordinatedLearningV5   `json:"coordinated,omitempty"`
	Sequence       uint64                             `json:"sequence"`
	Readout        *knowledge.CognitiveReadout        `json:"readout,omitempty"`
	Learning       *knowledge.CognitiveLearningResult `json:"learning,omitempty"`
	Phase4Learning *knowledge.Phase4LearningV5        `json:"phase4_learning,omitempty"`
	Duplicate      bool                               `json:"duplicate"`
}

func (r *BrainRuntime) ApplyCognitiveModel(command CognitiveModelCommand, at time.Time) (CognitiveModelResult, error) {
	out := CognitiveModelResult{}
	if r == nil || r.brain == nil {
		return out, errors.New("runtime unavailable")
	}
	c, err := freezeComposition(command)
	if err != nil {
		return out, err
	}
	if err = c.Validate(); err != nil {
		return out, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err = r.ensureWritableLocked(); err != nil {
		return out, err
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	at = at.UTC()
	prefix := "cognitive/" + c.ID + "/"
	key := prefix + knowledge.CognitiveDigest(c)
	if r.cognitiveCommandIndex == nil {
		r.cognitiveCommandIndex = map[string]string{}
		for id := range r.processedEvents {
			if strings.HasPrefix(id, "cognitive/") {
				i := strings.LastIndex(id, "/")
				r.cognitiveCommandIndex[id[:i+1]] = id
			}
		}
	}
	if id, ok := r.cognitiveCommandIndex[prefix]; ok {
		if id != key {
			return out, errors.New("cognitive ID reused for a different command")
		}
		return CognitiveModelResult{Sequence: r.processedEvents[id], Duplicate: true}, nil
	}
	if c.Initialize != nil && r.brain.HasCognitiveModel() {
		return out, errors.New("cognitive model already initialized")
	}
	if c.Input != nil {
		p, e := r.brain.CognitivePreview(*c.Input)
		if e != nil {
			return out, e
		}
		out.Readout = &p
	}
	if c.Training != nil {
		if err = r.brain.ValidateCognitiveBatch(*c.Training); err != nil {
			return out, err
		}
	}
	if c.Calibration != nil {
		if err = r.brain.ValidateCognitiveBatch(*c.Calibration); err != nil {
			return out, err
		}
	}
	if c.Phase2Init != nil {
		if err = r.brain.ValidatePhase2InitV5(*c.Phase2Init); err != nil {
			return out, err
		}
	}
	if c.Phase2Training != nil {
		if err = r.brain.ValidatePhase2BatchV5(*c.Phase2Training); err != nil {
			return out, err
		}
	}
	if r.brain.HasPhase4V5() && c.Phase4Training == nil && c.Coordinated == nil {
		return out, errors.New("Phase IV frozen substrate requires explicit metapolicy update")
	}
	if c.Phase4Init != nil {
		if err = r.brain.ValidatePhase4InitV5(*c.Phase4Init); err != nil {
			return out, err
		}
	}
	if c.Phase4Training != nil {
		if err = r.brain.ValidatePhase4BatchV5(*c.Phase4Training); err != nil {
			return out, err
		}
	}
	if c.Phase3Training != nil {
		if !r.brain.HasPhase2V5() {
			return out, errors.New("phase3 requires an experimental Phase II model")
		}
		if err = r.brain.ValidateCognitiveBatch(*c.Phase3Training); err != nil {
			return out, err
		}
	}
	if r.brain.HasPhase2V5() && (c.Training != nil || c.Calibration != nil) {
		return out, errors.New("phase2 experiment requires explicit versioned world-only updates")
	}
	if c.Coordinated != nil {
		if err = r.brain.ValidateCoordinatedBatchV5(*c.Coordinated); err != nil {
			return out, err
		}
	}
	if len(c.Search) > 0 {
		cfg, e := r.brain.CognitiveConfig()
		if e != nil {
			return out, e
		}
		for i, evaluation := range c.Search {
			point, e := knowledge.ProposeBMPS(c.Search[:i], cfg.Seed)
			if e != nil {
				return out, e
			}
			if point != evaluation.Point {
				return out, errors.New("search history differs from GP/EI proposals")
			}
		}
	}
	err = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeCognitiveModel, Timestamp: at, CognitiveModel: &c}, func() error {
		if c.Phase4Init != nil {
			if e := r.brain.InitializePhase4V5(*c.Phase4Init, at); e != nil {
				return e
			}
		}
		if c.Coordinated != nil {
			st, e := r.brain.LearnCoordinatedV5(*c.Coordinated, at)
			if e != nil {
				return e
			}
			out.Coordinated = &st
		}
		if c.Phase4Training != nil {
			st, e := r.brain.LearnPhase4V5(*c.Phase4Training, at)
			if e != nil {
				return e
			}
			out.Phase4Learning = &st
		}

		if c.Phase3Training != nil {
			p, e := r.brain.LearnCognitiveBatch(*c.Phase3Training, at)
			if e != nil {
				return e
			}
			out.Learning = &p
		}
		if c.Phase2Init != nil {
			if e := r.brain.InitializePhase2V5(*c.Phase2Init, at); e != nil {
				return e
			}
		}
		if c.Phase2Training != nil {
			p, e := r.brain.LearnPhase2V5(*c.Phase2Training, at)
			if e != nil {
				return e
			}
			out.Learning = &p
		}
		if c.Initialize != nil {
			if e := r.brain.InitializeCognitiveModel(*c.Initialize, at); e != nil {
				return e
			}
		}
		if c.Input != nil {
			p, e := r.brain.AdvanceCognitiveModel(*c.Input, at)
			if e != nil {
				return e
			}
			out.Readout = &p
		}
		if c.Training != nil {
			p, e := r.brain.LearnCognitiveBatch(*c.Training, at)
			if e != nil {
				return e
			}
			out.Learning = &p
		}
		if len(c.Search) > 0 {
			if e := r.brain.FitCognitiveBMPS(c.Search, *c.Evidence, at); e != nil {
				return e
			}
		}
		if c.Calibration != nil {
			if e := r.brain.CalibrateNeuralMeta(*c.Calibration, at); e != nil {
				return e
			}
		}
		r.seq++
		r.processedEvents[key] = r.seq
		r.cognitiveCommandIndex[prefix] = key
		return nil
	})
	out.Sequence = r.seq
	return out, err
}
func (r *BrainRuntime) PreviewCognitiveModel(input knowledge.CognitiveInput) (knowledge.CognitiveReadout, error) {
	if r == nil {
		return knowledge.CognitiveReadout{}, errors.New("runtime unavailable")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return knowledge.CognitiveReadout{}, err
	}
	return r.brain.CognitivePreview(input)
}
func (r *BrainRuntime) CognitiveModelParameters() ([]float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return nil, err
	}
	return r.brain.CognitiveParameters()
}
func (r *BrainRuntime) CognitiveBMPSWeights() ([4]float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return [4]float64{}, err
	}
	return r.brain.CognitiveBMPSWeights()
}
