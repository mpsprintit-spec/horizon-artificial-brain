package execution

import (
	"github.com/project-horizon/horizon-core/services/ai/plugin"
	"sync"
)


type ExecutionCore struct {
	Mu      sync.RWMutex
	Plugins map[string][]plugin.Plugin
}

func NewExecutionCore() *ExecutionCore {
	return &ExecutionCore{Plugins: make(map[string][]plugin.Plugin)}
}
