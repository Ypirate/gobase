package base

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ypirate/gobase/mlog"
	"github.com/gin-gonic/gin"
)

func init() {
	mlog.InitLog(mlog.LogConfig{
		Level:  "debug",
		Stdout: true,
	})
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := New(Config{
		Domain:  server.URL,
		Timeout: 5 * time.Second,
	})

	ctx := &gin.Context{}
	resp, err := client.Get(ctx, "/")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if user.ID != 1 || user.Name != "test" {
		t.Errorf("unexpected response: %+v", user)
	}

	t.Logf("Get success: %+v", user)
}

func TestPostJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type: application/json, got %s", contentType)
		}

		var req User
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request failed: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Code:    0,
			Message: "success",
			Data:    req,
		})
	}))
	defer server.Close()

	client := New(Config{
		Domain:  server.URL,
		Timeout: 5 * time.Second,
	})

	req := User{ID: 100, Name: "post_test"}
	var resp Response
	ctx := &gin.Context{}
	err := client.PostJSON(ctx, "/", req, &resp)
	if err != nil {
		t.Fatalf("PostJSON failed: %v", err)
	}

	if resp.Code != 0 || resp.Message != "success" {
		t.Errorf("unexpected response: %+v", resp)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok || data["id"].(float64) != 100 || data["name"] != "post_test" {
		t.Errorf("unexpected data: %+v", resp.Data)
	}

	t.Logf("PostJSON success: %+v", resp)
}

func TestRetry(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		t.Logf("Attempt %d", attemptCount)
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"success_after_retry"}`))
	}))
	defer server.Close()

	client := New(Config{
		Domain:  server.URL,
		Timeout: 5 * time.Second,
		Retry:   3,
	})

	ctx := &gin.Context{}
	resp, err := client.Get(ctx, "/")
	if err != nil {
		t.Fatalf("Get with retry failed: %v", err)
	}

	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}

	if user.ID != 1 || user.Name != "success_after_retry" {
		t.Errorf("unexpected response: %+v", user)
	}

	t.Logf("Retry success after %d attempts: %+v", attemptCount, user)
}

func TestTraceID(t *testing.T) {
	expectedTraceID := "test-trace-abc-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("Trace-Id")
		if traceID != expectedTraceID {
			t.Errorf("expected trace_id %s, got %s", expectedTraceID, traceID)
		}
		t.Logf("Received trace_id: %s", traceID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"trace_test"}`))
	}))
	defer server.Close()

	client := New(Config{
		Domain:  server.URL,
		Timeout: 5 * time.Second,
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("trace_id", expectedTraceID)
	resp, err := client.Get(ctx, "/")
	if err != nil {
		t.Fatalf("Get with trace_id failed: %v", err)
	}

	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	t.Logf("TraceID test passed: %+v", user)
}

func TestCustomHeaders(t *testing.T) {
	customKey := "X-Custom-Header"
	customValue := "custom-value-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		custom := r.Header.Get(customKey)
		if custom != customValue {
			t.Errorf("expected %s=%s, got %s", customKey, customValue, custom)
		}
		t.Logf("Received custom header: %s", custom)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"header_test"}`))
	}))
	defer server.Close()

	client := New(Config{
		Domain:  server.URL,
		Timeout: 5 * time.Second,
	})

	ctx := &gin.Context{}
	headers := map[string]string{
		customKey: customValue,
	}
	resp, err := client.Get(ctx, "/", headers)
	if err != nil {
		t.Fatalf("Get with custom headers failed: %v", err)
	}

	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	t.Logf("Custom headers test passed: %+v", user)
}
