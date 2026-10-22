package runtime

import (
	"os"
	"strings"
	"testing"
)

// Regression guard for production access patterns. The legacy mutable
// WithBrainView/WithActiveBrain APIs are deliberately still exported during
// migration, but production consumers MUST use the detached inspection
// boundary rather than reintroducing live neural pointers.
func TestProductionReadConsumersDoNotAcquireMutableBrainPointers(t *testing.T) {
	sources:=[]string{
		"../main.go",
		"../startup.go",
		"../cmd/horizon-cli/main.go",
	}
	for _,path:=range sources {
		t.Run(path,func(t *testing.T){
			data,err:=os.ReadFile(path)
			if err!=nil {t.Fatal(err)}
			code:=string(data)
			for _,forbidden:=range []string{
				".WithActiveBrain(",
				".WithBrainView(",
				".Knowledge.Registry",
				".Runtime.brain",
				".Orchestrator.Runtime",
			} {
				if strings.Contains(code,forbidden) {
					t.Fatalf("production source %s bypasses immutable Brain ownership via %q",path,forbidden)
				}
			}
		})
	}
}
