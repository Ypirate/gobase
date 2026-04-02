package mlog

import (
	"context"

	"go.uber.org/zap"
)

// contextFieldsKey is the key type for storing dynamic zap fields in context.
type contextFieldsKey struct{}

// AddFields appends multiple zap.Field to the context.
// Returns a new context containing all existing and new fields.
func AddFields(ctx context.Context, fields ...zap.Field) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	existing := getFieldsFromContext(ctx)
	newFields := append(existing, fields...)
	return context.WithValue(ctx, contextFieldsKey{}, newFields)
}

// getFieldsFromContext safely retrieves the dynamic fields from context.
func getFieldsFromContext(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}
	if val := ctx.Value(contextFieldsKey{}); val != nil {
		if fields, ok := val.([]zap.Field); ok {
			return fields
		}
	}
	return nil
}
