package execution

import (
	"github.com/project-horizon/horizon-core/services/ai/plugin"
	"sync"
)


func (e *ExecutionCore) Dispatch(actionName string, contextData string) {
	e.Mu.RLock()
	mappedPlugins, exists := e.Plugins[actionName]
	e.Mu.RUnlock()

	if !exists || len(mappedPlugins) == 0 {
		e.Mu.RLock()
		fallbackPlugins := e.Plugins["LogSystem"]
		e.Mu.RUnlock()
		for _, p := range fallbackPlugins {
			p.Trigger(actionName)
		}
		return
	}

	var wg sync.WaitGroup
	for _, pl := range mappedPlugins {
		wg.Add(1)
		go func(p plugin.Plugin) {
			defer wg.Done()
			p.Trigger(contextData)
		}(pl)
	}
	wg.Wait()
}
