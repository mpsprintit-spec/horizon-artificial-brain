package core

import (
	"reflect"
	"testing"
)

func TestFacadeDoesNotPublishMutableNeuralAliases(t *testing.T) {
	typeOf := reflect.TypeOf(HorizonEngine{})
	for _, name := range []string{"Runtime", "Knowledge", "Learning", "Thinking", "Language", "Orchestrator", "Supervisor"} {
		if field, exists := typeOf.FieldByName(name); exists && field.IsExported() {
			t.Errorf("facade still publishes replaceable neural owner/alias %s", name)
		}
	}
}
