// Package utils contains small shared helpers that do not belong to a domain package.
package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func NewID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return prefix + "-unknown"
	}
	p := strings.TrimSuffix(prefix, "-")
	if p == "" {
		return hex.EncodeToString(b)
	}
	return p + "-" + hex.EncodeToString(b)
}
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
