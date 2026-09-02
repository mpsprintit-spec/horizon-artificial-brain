package utils

import (
	"context"
	"testing"
)

func TestRequestIDContext(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "req-1")
	if RequestIDFromContext(ctx) != "req-1" {
		t.Fatal("missing request id")
	}
}
func TestNewID(t *testing.T) {
	if NewID("req") == NewID("req") {
		t.Fatal("ids should be unique")
	}
}
