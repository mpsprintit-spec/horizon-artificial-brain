package teaching

import (
	"bytes"
	"reflect"
	"testing"
)

func TestHNL2FourReferencesAndLegacyCompatibility(t *testing.T) {
	p := askFixture(qX, .2, .3)
	before := mustWire(t, p)
	decoded, e := Unmarshal(before)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(before, mustWire(t, decoded)) {
		t.Fatal("HNL/1 changed binary representation")
	}
	a, _ := QuestionCode(p.Question)
	p.Version = ExtendedVersion
	b, _ := QuestionCode(p.Question)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("existing query identity changed in HNL/2")
	}
	p.Question = Question{Domain: 3, Relation: SumGreater, A: Ref{0, 1}, B: Ref{0, 2}, C: Ref{0, 0}, D: Ref{0, 3}}
	w := mustWire(t, p)
	p.Question.A, p.Question.B = p.Question.B, p.Question.A
	p.Question.C, p.Question.D = p.Question.D, p.Question.C
	if !bytes.Equal(w, mustWire(t, p)) {
		t.Fatal("sum operand order did not canonicalize")
	}
	d, e := Unmarshal(w)
	if e != nil || d.Question != p.Question.Canonical() {
		t.Fatal("four arguments did not round trip")
	}
	p.Version = Version
	if _, e = Marshal(p); e == nil {
		t.Fatal("four-argument opcode accepted in legacy packet")
	}
	p.Version = ExtendedVersion
	p.Question.Relation = SameChange
	p.Question.A, p.Question.B, p.Question.C, p.Question.D = Ref{0, 3}, Ref{3, 3}, Ref{3, 3}, Ref{6, 3}
	w = mustWire(t, p)
	p.Question.A, p.Question.B, p.Question.C, p.Question.D = p.Question.C, p.Question.D, p.Question.A, p.Question.B
	if !bytes.Equal(w, mustWire(t, p)) {
		t.Fatal("change-pair exchange did not preserve identity")
	}
	h := newHarness(t)
	out := exchange(t, h.bridge, p)
	if out.Version != ExtendedVersion || out.Reply.Status == 3 {
		t.Fatal("untrained four-argument relation was evaluated by the bridge")
	}
	for _, q := range []Question{
		{Domain: 3, Relation: Equal, A: Ref{0, 0}, B: Ref{0, 3}},
		{Domain: 3, Relation: SumEqual, A: Ref{0, 1}, B: Ref{0, 2}, C: Ref{0, 0}, D: Ref{0, 3}},
	} {
		p.Question = q
		p.Version = ExtendedVersion
		w := mustWire(t, p)
		decoded, err := Unmarshal(w)
		if err != nil || decoded.Question != q.Canonical() {
			t.Fatal("equality AST failed round trip")
		}
		if exchange(t, h.bridge, p).Reply.Status == 3 {
			t.Fatal("equality was calculated by the bridge")
		}
		p.Version = Version
		if _, err = Marshal(p); err == nil {
			t.Fatal("new equality opcode accepted by legacy wire version")
		}
	}
}
