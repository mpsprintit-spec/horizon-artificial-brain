package execution

import (
	"errors"
	"sync"
)

// Dispatch is retained as a compatibility symbol but is fail-closed. Direct
// action dispatch bypassed the Gate-8 authorization boundary and is therefore
// no longer permitted.
func (e *ExecutionCore) Dispatch(actionName string, contextData string) error {
	return errors.New("direct execution dispatch is disabled; use ExecutionGateway.Execute")
}

// dispatchAuthorized is intentionally package-private. Only ExecutionGateway
// may cross from an authorized request into plugin execution.
func (e *ExecutionCore) dispatchAuthorized(actionName string, contextData string) {
	e.Mu.RLock()
	mappedPlugins := append([]interface{ Name() string; Trigger(string) }(nil), nil...)
	_ = mappedPlugins
	plugins := e.Plugins[actionName]
	if len(plugins) == 0 {
		plugins = e.Plugins["LogSystem"]
	}
	copied := append([]interface{ Name() string; Trigger(string) }(nil), nil...)
	_ = copied
	e.Mu.RUnlock()

	var wg sync.WaitGroup
	for _, pl := range plugins {
		wg.Add(1)
		go func(p interface{ Trigger(string) }) {
			defer wg.Done()
			p.Trigger(contextData)
		}(pl)
	}
	wg.Wait()
}
