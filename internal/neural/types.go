package neural

import (
	"errors"
	"math"
	"sort"
	"sync"
)

var (
	ErrInvalidConfig = errors.New("invalid neural configuration")
	ErrInvalidID     = errors.New("invalid neural id")
)

// Config controls the substrate. Values are deliberately bounded so a bad
// configuration cannot destabilize the network numerically.
type Config struct {
	Leak             float64
	Threshold        float64
	Reset            float64
	LearningRate     float64
	HomeostasisRate  float64
	TraceDecay       float64
	SynapseDecay     float64
	StructuralRate   float64
	MinWeight        float64
	MaxWeight        float64
	MaxNeurons       int
	MaxSynapses      int
}

func DefaultConfig() Config {
	return Config{Leak: 0.15, Threshold: 1, Reset: 0, LearningRate: 0.02,
		HomeostasisRate: 0.002, TraceDecay: 0.92, SynapseDecay: 0.0002,
		StructuralRate: 0.001, MinWeight: -2, MaxWeight: 2,
		MaxNeurons: 100_000, MaxSynapses: 1_000_000}
}

func (c Config) Validate() error {
	if c.Leak < 0 || c.Leak > 1 || c.Threshold <= 0 || c.LearningRate < 0 ||
		c.HomeostasisRate < 0 || c.TraceDecay < 0 || c.TraceDecay > 1 ||
		c.SynapseDecay < 0 || c.StructuralRate < 0 || c.MinWeight >= c.MaxWeight ||
		c.MaxNeurons <= 0 || c.MaxSynapses <= 0 {
		return ErrInvalidConfig
	}
	return nil
}

type NeuronID uint64
type SynapseID uint64

type Neuron struct {
	ID          NeuronID
	Potential   float64
	Activity    float64
	Homeostatic float64
	Trace       float64
	Age         uint64
	Fired       bool
}

type Synapse struct {
	ID          SynapseID
	Pre         NeuronID
	Post        NeuronID
	Weight      float64
	Eligibility float64
	Trace       float64
	Usage       float64
	Age         uint64
}

type Network struct {
	mu       sync.RWMutex
	cfg      Config
	step     uint64
	nextN    NeuronID
	nextS    SynapseID
	neurons  map[NeuronID]*Neuron
	synapses map[SynapseID]*Synapse
	out      map[NeuronID][]SynapseID
	in       map[NeuronID][]SynapseID
}

func New(cfg Config) (*Network, error) {
	if err := cfg.Validate(); err != nil { return nil, err }
	return &Network{cfg: cfg, neurons: make(map[NeuronID]*Neuron), synapses: make(map[SynapseID]*Synapse), out: make(map[NeuronID][]SynapseID), in: make(map[NeuronID][]SynapseID)}, nil
}

func (n *Network) Config() Config { n.mu.RLock(); defer n.mu.RUnlock(); return n.cfg }
func (n *Network) StepCount() uint64 { n.mu.RLock(); defer n.mu.RUnlock(); return n.step }
func (n *Network) NeuronCount() int { n.mu.RLock(); defer n.mu.RUnlock(); return len(n.neurons) }
func (n *Network) SynapseCount() int { n.mu.RLock(); defer n.mu.RUnlock(); return len(n.synapses) }

func (n *Network) AddNeuron() (NeuronID, error) {
	n.mu.Lock(); defer n.mu.Unlock()
	if len(n.neurons) >= n.cfg.MaxNeurons { return 0, ErrInvalidConfig }
	n.nextN++
	id := n.nextN
	n.neurons[id] = &Neuron{ID:id, Homeostatic:n.cfg.Threshold}
	return id, nil
}

func (n *Network) Connect(pre, post NeuronID, weight float64) (SynapseID, error) {
	n.mu.Lock(); defer n.mu.Unlock()
	if _, ok := n.neurons[pre]; !ok { return 0, ErrInvalidID }
	if _, ok := n.neurons[post]; !ok { return 0, ErrInvalidID }
	if len(n.synapses) >= n.cfg.MaxSynapses { return 0, ErrInvalidConfig }
	if math.IsNaN(weight) || math.IsInf(weight, 0) { return 0, ErrInvalidConfig }
	weight = clamp(weight, n.cfg.MinWeight, n.cfg.MaxWeight)
	n.nextS++
	s := &Synapse{ID:n.nextS, Pre:pre, Post:post, Weight:weight}
	n.synapses[s.ID] = s
	n.out[pre] = append(n.out[pre], s.ID)
	n.in[post] = append(n.in[post], s.ID)
	return s.ID, nil
}

func (n *Network) Neuron(id NeuronID) (Neuron, bool) {
	n.mu.RLock(); defer n.mu.RUnlock(); x, ok := n.neurons[id]; if !ok { return Neuron{}, false }; return *x, true
}

func (n *Network) Synapse(id SynapseID) (Synapse, bool) {
	n.mu.RLock(); defer n.mu.RUnlock(); x, ok := n.synapses[id]; if !ok { return Synapse{}, false }; return *x, true
}

func (n *Network) SynapsesFrom(id NeuronID) []Synapse {
	n.mu.RLock(); defer n.mu.RUnlock()
	ids := append([]SynapseID(nil), n.out[id]...)
	sort.Slice(ids, func(i,j int) bool { return ids[i] < ids[j] })
	out := make([]Synapse,0,len(ids)); for _, sid := range ids { if s,ok:=n.synapses[sid]; ok { out=append(out,*s) } }; return out
}

func clamp(v, lo, hi float64) float64 { if v < lo { return lo }; if v > hi { return hi }; return v }
