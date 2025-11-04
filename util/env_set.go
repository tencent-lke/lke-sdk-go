package util

import "context"

type contextKey string

const envSetContextKey contextKey = "EnvSet"

// WithEnvSet stores envSet in ctx for downstream runner calls.
func WithEnvSet(ctx context.Context, envSet string) context.Context {
	if ctx == nil || envSet == "" {
		return ctx
	}
	return context.WithValue(ctx, envSetContextKey, envSet)
}

// GetEnvSetFromContext extracts envSet from ctx if present.
func GetEnvSetFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if envSet, ok := ctx.Value(envSetContextKey).(string); ok {
		return envSet
	}
	return ""
}
