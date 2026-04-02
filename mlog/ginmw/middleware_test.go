package ginmw

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ypirate/gobase/mlog"
	"github.com/gin-gonic/gin"
)

func TestTraceIDMiddleware(t *testing.T) {
	mlog.InitLog(mlog.LogConfig{Level: "debug", Stdout: false})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceID())
	router.GET("/test", func(c *gin.Context) {
		mlog.Infof(c.Request.Context(), "test request")
		c.String(200, "ok")
	})

	// Test with existing trace-id
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "existing-trace-123")
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	if wr.Header().Get("X-Trace-ID") != "existing-trace-123" {
		t.Error("expected trace-id in response header")
	}

	// Test with generated trace-id
	req2 := httptest.NewRequest("GET", "/test", nil)
	wr2 := httptest.NewRecorder()
	router.ServeHTTP(wr2, req2)

	responseTraceID := wr2.Header().Get("X-Trace-ID")
	if responseTraceID == "" {
		t.Error("expected generated trace-id in response header")
	}
}

func TestAccessLogMiddleware(t *testing.T) {
	mlog.InitLog(mlog.LogConfig{Level: "debug", Stdout: false})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AccessLog())
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	if wr.Code != 200 {
		t.Errorf("expected status 200, got %d", wr.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	mlog.InitLog(mlog.LogConfig{Level: "debug", Stdout: false})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	if wr.Code != 500 {
		t.Errorf("expected status 500, got %d", wr.Code)
	}
}

func TestMiddlewareChain(t *testing.T) {
	mlog.InitLog(mlog.LogConfig{Level: "debug", Stdout: false})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceID())
	router.Use(Recovery())
	router.Use(AccessLog())
	router.GET("/test", func(c *gin.Context) {
		mlog.Infof(c.Request.Context(), "handler executed")
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	if wr.Code != 200 {
		t.Errorf("expected status 200, got %d", wr.Code)
	}
}

func TestAddFieldsHelpers(t *testing.T) {
	mlog.InitLog(mlog.LogConfig{Level: "debug", Stdout: false})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		AddString(c, "user_id", "user123")
		AddInt(c, "age", 25)
		AddBool(c, "active", true)

		ctx := GetContext(c)
		mlog.Infof(ctx, "test with fields")
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)

	if wr.Code != 200 {
		t.Errorf("expected status 200, got %d", wr.Code)
	}
}

func TestGenerateTraceID(t *testing.T) {
	id1 := generateTraceID()
	id2 := generateTraceID()

	if id1 == id2 {
		t.Error("expected different trace IDs")
	}

	if !strings.Contains(id1, "-") {
		t.Error("expected trace ID to contain hyphen")
	}
}
