package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

func TestGateGFacadeThinkingCannotFallBackToLegacyEngines(t *testing.T) {
	h := NewHorizonEngine()
	h.Decision = nil
	out, e := h.Pulse("pengalaman neural")
	if e != nil || out.Interpretation.Source != "neural" {
		t.Fatal("production Pulse relied on legacy decision", e)
	}
	thought, e := h.ThinkActive(2)
	if e != nil || thought.BrainIdentity != out.State.BrainIdentity {
		t.Fatal("recurrent thought relied on legacy decision", e)
	}
	active := map[string]bool{"ThinkActive": true, "Pulse": true, "PulseSignals": true, "pulseSignalsWith": true}
	for _, path := range []string{"engine.go", "managed_engine.go"} {
		data, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		f, e := parser.ParseFile(token.NewFileSet(), path, data, 0)
		if e != nil {
			t.Fatal(e)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !active[fn.Name.Name] {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				s, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if field, ok := s.X.(*ast.SelectorExpr); ok && (field.Sel.Name == "Decision" || field.Sel.Name == "Thinking" || field.Sel.Name == "Understanding") {
					t.Fatalf("%s reintroduced retired cognitive call", fn.Name.Name)
				}
				if pkg, ok := s.X.(*ast.Ident); ok && (pkg.Name == "thinking" || pkg.Name == "understanding" || pkg.Name == "decision") {
					t.Fatalf("%s calls retired engine", fn.Name.Name)
				}
				return true
			})
		}
	}
}
