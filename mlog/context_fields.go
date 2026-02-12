package mlog

import (
	"context"

	"github.com/gin-gonic/gin"
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

// --- Gin Helpers ---

// GinAddFields adds one or more zap.Field to gin.Context's underlying request context.
func GinAddFields(c *gin.Context, fields ...zap.Field) {
	ctx := c.Request.Context()
	ctx = AddFields(ctx, fields...)
	c.Request = c.Request.WithContext(ctx)
}

// GinAddString is a shortcut for adding a string field.
func GinAddString(c *gin.Context, key, value string) {
	GinAddFields(c, zap.String(key, value))
}

// GinAddInt is a shortcut for adding an int field.
func GinAddInt(c *gin.Context, key string, value int) {
	GinAddFields(c, zap.Int(key, value))
}

// GinAddBool is a shortcut for adding a bool field.
func GinAddBool(c *gin.Context, key string, value bool) {
	GinAddFields(c, zap.Bool(key, value))
}
