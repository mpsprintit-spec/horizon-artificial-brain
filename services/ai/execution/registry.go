package execution

import "github.com/project-horizon/horizon-core/services/ai/plugin"


func (e *ExecutionCore) RegisterPlugin(actionName string, p plugin.Plugin) {
	e.Mu.Lock()
	defer e.Mu.Unlock()
	e.Plugins[actionName] = append(e.Plugins[actionName], p)
}
