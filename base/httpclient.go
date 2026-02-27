package base

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Ypirate/gobase/mlog"
	"github.com/gin-gonic/gin"
)

// Response 全局统一返回体结构  Code 业务状态码 Message 正常异常消息 Data 业务数据
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Config 通用 HTTP 客户端配置
type Config struct {
	Service            string        `yaml:"service"`            // 服务名，用于日志/metrics（可选）
	Domain             string        `yaml:"domain"`             // 基础 URL，如 http://user-center.example.com
	Timeout            time.Duration `yaml:"timeout"`            // 单次请求超时
	MaxIdleConnections int           `yaml:"maxIdleConnections"` // 最大空闲连接数
	IdleConnTimeout    time.Duration `yaml:"idleConnTimeout"`    // 空闲连接超时
	Retry              int           `yaml:"retry"`              // 重试次数（总执行 retry+1 次）
	HTTPStat           bool          `yaml:"httpStat"`           // 是否打印请求耗时（调试用）
}

// Client 是通用 HTTP 客户端
type Client struct {
	httpClient *http.Client
	Config     Config
	baseURL    string
}

// New 创建新客户端
func New(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConnections,
		MaxIdleConnsPerHost: cfg.MaxIdleConnections, // 关键：每个 host 的空闲连接上限
		IdleConnTimeout:     cfg.IdleConnTimeout,
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		Config:  cfg,
		baseURL: cfg.Domain,
	}
}

// Get 发起 GET 请求，返回原始字节
func (c *Client) Get(ctx *gin.Context, path string, headers ...map[string]string) ([]byte, error) {
	var h map[string]string
	if len(headers) > 0 {
		h = headers[0]
	}
	return c.doRequest(ctx, "GET", path, nil, h)
}

// Post 发起 POST 请求，body 为 []byte
func (c *Client) Post(ctx *gin.Context, path string, body []byte, headers ...map[string]string) ([]byte, error) {
	var h map[string]string
	if len(headers) > 0 {
		h = headers[0]
	}
	return c.doRequest(ctx, "POST", path, bytes.NewReader(body), h)
}

// PostJSON 发起 JSON POST 请求，自动序列化 req 并反序列化 resp 到 result
func (c *Client) PostJSON(ctx *gin.Context, path string, req interface{}, result interface{}, headers ...map[string]string) error {
	var body []byte
	var err error
	if req != nil {
		body, err = json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
	}

	var h map[string]string
	if len(headers) > 0 {
		h = headers[0]
	}
	respBytes, err := c.doRequest(ctx, "POST", path, bytes.NewReader(body), h)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(respBytes, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// doRequest 执行底层请求，带重试和统计
func (c *Client) doRequest(ctx *gin.Context, method, path string, body io.Reader, headers map[string]string) ([]byte, error) {
	url := c.baseURL + path
	var lastErr error

	for attempt := 0; attempt <= c.Config.Retry; attempt++ {
		start := time.Now()

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			lastErr = err
			return nil, err
		}

		// 设置通用 Header
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		// 设置自定义 Header
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		// 从 context 获取 trace_id 并透传到下游
		if traceID := getTraceID(ctx); traceID != "" {
			req.Header.Set("Trace-Id", traceID)
		}

		if c.Config.HTTPStat {
			mlog.Infof(ctx, "[HTTPClient][%s] %s %s (attempt=%d)\n", c.Config.Service, method, url, attempt+1)
		}

		resp, err := c.httpClient.Do(req)
		duration := time.Since(start)

		if c.Config.HTTPStat {
			status := "ERR"
			if err == nil {
				status = fmt.Sprintf("%d", resp.StatusCode)
			}
			mlog.Infof(ctx, "[HTTPClient][%s] Done in %v, status=%s\n", c.Config.Service, duration, status)
		}

		// 网络错误或 5xx 错误才重试（可按需调整）
		if err != nil || (resp != nil && resp.StatusCode >= 500) {
			lastErr = err
			if resp != nil {
				resp.Body.Close()
			}
			if attempt < c.Config.Retry {
				time.Sleep(time.Millisecond * 100 * time.Duration(1<<uint(attempt))) // 指数退避
			}
			continue
		}

		if resp == nil {
			return nil, fmt.Errorf("[%s] %s %s: empty response", c.Config.Service, method, url)
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			resp.Body.Close()
			continue
		}

		// 非 2xx 视为业务错误，不重试
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		resp.Body.Close()
		return respBody, nil
	}

	return nil, fmt.Errorf("[%s] %s %s failed after %d retries: %w",
		c.Config.Service, method, url, c.Config.Retry, lastErr)
}

func getTraceID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID := ctx.GetString("trace_id"); traceID != "" {
		return traceID
	}
	if traceID := ctx.GetString("Trace-Id"); traceID != "" {
		return traceID
	}
	if ctx.Request != nil && ctx.Request.Context() != nil {
		if traceID, ok := ctx.Request.Context().Value("Trace-Id").(string); ok {
			return traceID
		}
		if traceID, ok := ctx.Request.Context().Value("trace_id").(string); ok {
			return traceID
		}
	}
	return ""
}
