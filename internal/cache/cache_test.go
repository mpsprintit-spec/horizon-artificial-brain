package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryCacheSetGetDelete(t *testing.T) {
	c := NewMemory()
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v" {
		t.Fatalf("got %q", got)
	}
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(ctx, "k")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
