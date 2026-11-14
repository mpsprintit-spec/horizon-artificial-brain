package jsonstream

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"testing"
)

func TestCompactArrayMatchesMarshal(t *testing.T) {
	values := []any{nil, true, -1, math.Copysign(0, -1), .12345678901234567,
		"<script>\n\u2028\"\\", map[string]any{"b": 2, "a": "&"},
		json.RawMessage(` { "escaped" : "<>&", "x": 1 } `), []int{}, []int(nil)}
	for _, count := range []int{0, 1, len(values)} {
		var out bytes.Buffer
		if err := Array(&out, count, false, false, "", func(i int) any { return values[i] }); err != nil {
			t.Fatal(err)
		}
		want, err := json.Marshal(values[:count])
		if err != nil || !bytes.Equal(out.Bytes(), want) {
			t.Fatalf("compact bytes changed: %s != %s (%v)", out.Bytes(), want, err)
		}
	}
	for _, value := range []any{math.NaN(), math.Inf(1), make(chan int), json.RawMessage(`{"bad":`)} {
		if err := Array(io.Discard, 1, false, false, "", func(int) any { return value }); err == nil {
			t.Fatalf("unserializable value accepted: %T", value)
		}
	}
	if err := Array(shortWriter{}, 1, false, false, "", func(int) any { return "value" }); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write swallowed: %v", err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
