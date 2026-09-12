package knowledge

import "time"

// LearnVectorTransition binds two experienced population states through the
// existing dynamic synapses. It does not introduce semantic relation types;
// temporal order is learned from repeated co-activation.
func (k *KnowledgeBase) LearnVectorTransition(previous, current NeuralVector, threshold float64, populationSize int, now time.Time) error {
	if k == nil {
		return ErrNilBrain
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	before, err := k.ProjectVectorPopulation(previous, threshold, populationSize)
	if err != nil {
		return err
	}
	after, err := k.ProjectVectorPopulation(current, threshold, populationSize)
	if err != nil {
		return err
	}
	for _, source := range before.Units {
		from := k.Registry.GetByID(source.NodeID)
		if from == nil {
			continue
		}
		for _, target := range after.Units {
			to := k.Registry.GetByID(target.NodeID)
			if to == nil || from.ID == to.ID {
				continue
			}
			strength := clamp01(source.Activation * target.Activation)
			if strength <= 0 {
				continue
			}
			k.Connect(from, to, strength, strength, false)
		}
	}
	return nil
}
