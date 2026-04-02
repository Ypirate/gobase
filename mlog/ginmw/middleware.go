package ginmw

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/Ypirate/gobase/mlog"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	TraceIDHeader = "X-Trace-ID"
	TraceIDKey    = "trace_id"
)

// TraceID middleware extracts or generates trace-id and injects it into context.
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			traceID = generateTraceID()
		}

		ctx := mlog.AddFields(c.Request.Context(), zap.String(TraceIDKey, traceID))
		c.Request = c.Request.WithContext(ctx)
		c.Header(TraceIDHeader, traceID)
		c.Next()
	}
}

// generateTraceID creates a unique trace ID using timestamp and random string.
func generateTraceID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// AccessLog middleware logs HTTP request details.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		mlog.Info(c.Request.Context(), "HTTP Request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

// Recovery middleware recovers from panics and logs the full stack trace.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := make([]byte, 4096)
				length := runtime.Stack(stack, false)

				mlog.Errorf(c.Request.Context(), "Panic recovered: %v\nStack:\n%s",
					err,
					string(stack[:length]),
				)

				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal Server Error",
				})
			}
		}()
		c.Next()
	}
}

// AddString adds a string field to gin context.
func AddString(c *gin.Context, key, value string) {
	ctx := mlog.AddFields(c.Request.Context(), zap.String(key, value))
	c.Request = c.Request.WithContext(ctx)
}

// AddInt adds an int field to gin context.
func AddInt(c *gin.Context, key string, value int) {
	ctx := mlog.AddFields(c.Request.Context(), zap.Int(key, value))
	c.Request = c.Request.WithContext(ctx)
}

// AddBool adds a bool field to gin context.
func AddBool(c *gin.Context, key string, value bool) {
	ctx := mlog.AddFields(c.Request.Context(), zap.Bool(key, value))
	c.Request = c.Request.WithContext(ctx)
}

// AddFields adds multiple fields to gin context.
func AddFields(c *gin.Context, fields ...zap.Field) {
	ctx := mlog.AddFields(c.Request.Context(), fields...)
	c.Request = c.Request.WithContext(ctx)
}

// GetContext returns the request context (for logging).
func GetContext(c *gin.Context) context.Context {
	return c.Request.Context()
}
