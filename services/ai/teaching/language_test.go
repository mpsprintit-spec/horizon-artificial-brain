package teaching

import (
	"bytes"
	"hash/crc32"
	"math"
	"reflect"
	"testing"
	"time"
)

var base = time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
var qX = Question{Domain: 1, Relation: Sign, A: Ref{0, 0}}
var qY = Question{Domain: 1, Relation: Sign, A: Ref{0, 1}}
var qRelation = Question{Domain: 1, Relation: SameSign, A: Ref{0, 0}, B: Ref{0, 1}}

func askFixture(q Question, x, y float64) Packet {
	return Packet{Version: Version, Kind: Ask, ID: 1, Source: 1, Stream: 1, At: base.Add(24 * time.Hour).UnixNano(), Question: q, Examples: []Example{{Role: Probe, Episode: 1, ObservedAt: base.Add(24 * time.Hour).UnixNano(), Samples: []Sample{{Lag: 0, Present: true, Values: [6]float64{x, y}}}}}}
}
func mustWire(t *testing.T, p Packet) []byte {
	t.Helper()
	b, e := Marshal(p)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestHNLCanonicalReferencesAndTemporalPresence(t *testing.T) {
	p := askFixture(qRelation, .2, -.3)
	p.Examples[0].Samples = append(p.Examples[0].Samples, Sample{Lag: 3, Present: true, Values: [6]float64{.4}}, Sample{Lag: 1, Present: false})
	w := mustWire(t, p)
	p.Question.A, p.Question.B = p.Question.B, p.Question.A
	p.Examples[0].Samples[0], p.Examples[0].Samples[2] = p.Examples[0].Samples[2], p.Examples[0].Samples[0]
	if !bytes.Equal(w, mustWire(t, p)) {
		t.Fatal("commutative reference or row order changed canonical packet")
	}
	d, e := Unmarshal(w)
	if e != nil {
		t.Fatal(e)
	}
	in, e := CompileAsk(d)
	if e != nil {
		t.Fatal(e)
	}
	if len(in.Sequence) != 4 || in.Sequence[0][0] != .4 || in.Sequence[0][6] != 1 || in.Sequence[2][6] != 0 || in.Sequence[3][0] != .2 {
		t.Fatal("temporal/missing coordinates changed")
	}
	zero := askFixture(qX, 0, 0)
	a, _ := CompileAsk(zero)
	zero.Examples[0].Samples[0].Present = false
	b, _ := CompileAsk(zero)
	if reflect.DeepEqual(a, b) {
		t.Fatal("observed zero collapsed into missing")
	}
	aCode, _ := QuestionCode(qX)
	other := qX
	other.A.Lag = 1
	bCode, _ := QuestionCode(other)
	if reflect.DeepEqual(aCode, bCode) {
		t.Fatal("different temporal arguments collapsed")
	}
	other = qRelation
	other.Relation = Greater
	aCode, _ = QuestionCode(other)
	other.A, other.B = other.B, other.A
	bCode, _ = QuestionCode(other)
	if reflect.DeepEqual(aCode, bCode) {
		t.Fatal("ordered comparison incorrectly made commutative")
	}
}

func TestHNLRejectsCorruptionBoundsAndAnswerLeakage(t *testing.T) {
	p := askFixture(qX, .2, -.3)
	w := mustWire(t, p)
	for n := 0; n < len(w); n++ {
		if _, e := Unmarshal(w[:n]); e == nil {
			t.Fatalf("accepted truncation at %d", n)
		}
	}
	for i := range w {
		bad := append([]byte(nil), w...)
		bad[i] ^= 1
		if _, e := Unmarshal(bad); e == nil {
			t.Fatalf("accepted corrupted byte %d", i)
		}
	}
	bad := append([]byte(nil), w...)
	bad[6] = 99
	order.PutUint32(bad[len(bad)-4:], crc32.ChecksumIEEE(bad[:len(bad)-4]))
	if _, e := Unmarshal(bad); e == nil {
		t.Fatal("accepted unknown instruction with valid CRC")
	}
	cases := []func(*Packet){
		func(p *Packet) { p.Examples[0].Outcome = &Evidence{Value: 1, ObservedAt: p.At} },
		func(p *Packet) { p.Examples[0].Samples[0].Values[0] = math.NaN() },
		func(p *Packet) { p.Examples[0].Samples[0].Lag = 64 },
		func(p *Packet) { p.Examples[0].Samples = append(p.Examples[0].Samples, p.Examples[0].Samples[0]) },
		func(p *Packet) { p.Examples[0].Samples[0].Present = false },
		func(p *Packet) { p.Examples[0].ObservedAt = p.At + 1 },
		func(p *Packet) { p.Question.Relation = 99 },
	}
	for i, change := range cases {
		p := askFixture(qX, .2, -.3)
		change(&p)
		if _, e := Marshal(p); e == nil {
			t.Fatalf("accepted invalid example %d", i)
		}
	}
}
