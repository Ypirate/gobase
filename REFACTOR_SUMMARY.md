# mlog 企业级日志库重构总结

## 重构目标

将 mlog 打造成企业级 Go 日志库，支持：
- 请求全链路追踪（trace-id）
- 自动记录代码位置
- Panic 捕获和堆栈记录
- 动态日志级别调整
- 子 Logger 支持
- 框架解耦设计

## 核心设计理念

### 1. 核心纯净，框架解耦

**问题：** 原设计中 mlog 核心依赖 Gin 框架，不够通用

**解决方案：**
- mlog 核心只依赖 `context.Context`
- 所有日志方法签名统一为 `func(ctx context.Context, ...)`
- Gin 相关功能移到独立的 `mlog/ginmw` 包

**优势：**
- 可用于任何 Go 项目（CLI、gRPC、HTTP 等）
- 易于扩展到其他框架（Echo、Fiber 等）
- 测试更简单，不需要 mock 框架

### 2. Context 驱动的字段传递

**核心机制：**
```go
// 通过 context 携带日志字段
ctx = mlog.AddFields(ctx, 
    zap.String("trace_id", "abc123"),
    zap.String("user_id", "user456"),
)

// 所有使用该 context 的日志都自动包含这些字段
mlog.Infof(ctx, "Processing request")
```

**优势：**
- 符合 Go 标准实践
- 字段自动传播到调用链
- 不依赖全局变量或框架特定 context

## 重构内容

### Phase 1: 重构核心初始化 (base.go)

**修改：**
- 移除 `sync.Once`，改用 `atomic.Bool` + `sync.RWMutex`
- 支持重新初始化（测试友好）
- 添加 `zap.AtomicLevel` 支持动态日志级别
- 添加 `getLogger()` 线程安全访问
- Error 级别自动记录堆栈（`zap.AddStacktrace(zapcore.ErrorLevel)`）

**新增 API：**
```go
func SetLevel(level string)  // 动态调整日志级别
func GetLevel() string        // 获取当前日志级别
```

### Phase 2: 统一日志 API (logger.go)

**修改：**
- 统一所有日志方法签名为 `ctx context.Context`（原 `Panicf` 不一致）
- 使用 `getLogger()` 替代直接访问全局 `logger`
- 移除 Gin 依赖

**新增结构化日志方法：**
```go
func Info(ctx context.Context, msg string, fields ...zap.Field)
func Error(ctx context.Context, msg string, fields ...zap.Field)
func Warn(ctx context.Context, msg string, fields ...zap.Field)
func Debug(ctx context.Context, msg string, fields ...zap.Field)
```

### Phase 3: 框架集成分离 (ginmw/)

**新建包：** `mlog/ginmw`

**提供中间件：**
- `TraceID()` - 自动提取或生成 trace-id
- `AccessLog()` - 记录 HTTP 请求详情
- `Recovery()` - 捕获 panic 并记录堆栈

**辅助函数：**
```go
func AddString(c *gin.Context, key, value string)
func AddInt(c *gin.Context, key string, value int)
func AddBool(c *gin.Context, key string, value bool)
func AddFields(c *gin.Context, fields ...zap.Field)
func GetContext(c *gin.Context) context.Context
```

### Phase 4: 子 Logger 支持 (child_logger.go)

**新增：**
```go
type Logger struct {
    fields []zap.Field
}

func With(fields ...zap.Field) *Logger
```

**使用示例：**
```go
var userLogger = mlog.With(zap.String("module", "user"))
userLogger.Infof(ctx, "User login")
// 输出: {"module":"user","msg":"User login",...}
```

### Phase 5: 测试重构

**新建：** `test_helper.go` - 消除测试代码重复

**测试覆盖：**
- 核心日志功能：10 个测试
- Gin 中间件：6 个测试
- 测试覆盖率：81.2%

## 文件结构

```
mlog/
├── base.go              # 核心初始化、配置
├── logger.go            # 日志 API（Infof/Errorf 等）
├── context_fields.go    # Context 字段管理
├── child_logger.go      # 子 Logger 实现
├── test_helper.go       # 测试辅助函数
├── logger_test.go       # 核心功能测试
├── README.md            # 文档
└── ginmw/               # Gin 框架集成（独立包）
    ├── middleware.go    # Gin 中间件
    └── middleware_test.go

examples/
└── example_usage.go     # 完整使用示例
```

## 使用对比

### 重构前（框架耦合）

```go
// 必须使用 Gin Context
func Handler(c *gin.Context) {
    mlog.GinAddString(c, "user_id", "user123")
    mlog.Infof(c, "Processing")  // 接受 any 类型
}
```

### 重构后（框架解耦）

```go
// 核心：使用标准 context.Context
func BusinessLogic(ctx context.Context) {
    ctx = mlog.AddFields(ctx, zap.String("user_id", "user123"))
    mlog.Infof(ctx, "Processing")
}

// Gin 集成：使用独立包
func Handler(c *gin.Context) {
    ginmw.AddString(c, "user_id", "user123")
    ctx := ginmw.GetContext(c)
    mlog.Infof(ctx, "Processing")
}
```

## 关键改进

### 1. 可重新初始化

**问题：** `sync.Once` 导致测试困难

**解决：**
```go
// 测试中可多次调用
mlog.InitLog(mlog.LogConfig{Level: "debug"})
// ... 测试
mlog.InitLog(mlog.LogConfig{Level: "info"})  // 重新初始化
```

### 2. 动态日志级别

**新增功能：**
```go
mlog.SetLevel("debug")  // 运行时调整
level := mlog.GetLevel()
```

### 3. 自动堆栈跟踪

**配置：**
```go
logger = zap.New(core, 
    zap.AddCaller(),
    zap.AddCallerSkip(2),
    zap.AddStacktrace(zapcore.ErrorLevel),  // Error+ 自动堆栈
)
```

### 4. 框架扩展性

**示例：为其他框架创建中间件**
```go
package echomw

import "github.com/Ypirate/gobase/mlog"

func TraceID() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            traceID := extractOrGenerate(c)
            ctx := mlog.AddFields(c.Request().Context(), 
                zap.String("trace_id", traceID))
            c.SetRequest(c.Request().WithContext(ctx))
            return next(c)
        }
    }
}
```

## 测试结果

```bash
$ go test ./mlog/... -v
=== RUN   TestLogLevels
--- PASS: TestLogLevels (0.00s)
=== RUN   TestContextFields
--- PASS: TestContextFields (0.00s)
=== RUN   TestGinContextFields
--- PASS: TestGinContextFields (0.00s)
=== RUN   TestNilLoggerFallback
--- PASS: TestNilLoggerFallback (0.00s)
=== RUN   TestParseLevel
--- PASS: TestParseLevel (0.00s)
=== RUN   TestTruncateUTF8
--- PASS: TestTruncateUTF8 (0.00s)
=== RUN   TestFatalfWithContext
--- PASS: TestFatalfWithContext (0.00s)
=== RUN   TestDynamicLogLevel
--- PASS: TestDynamicLogLevel (0.00s)
=== RUN   TestChildLogger
--- PASS: TestChildLogger (0.00s)
=== RUN   TestChildLoggerAllLevels
--- PASS: TestChildLoggerAllLevels (0.00s)
PASS
ok      github.com/Ypirate/gobase/mlog  0.866s

=== RUN   TestTraceIDMiddleware
--- PASS: TestTraceIDMiddleware (0.00s)
=== RUN   TestAccessLogMiddleware
--- PASS: TestAccessLogMiddleware (0.00s)
=== RUN   TestRecoveryMiddleware
--- PASS: TestRecoveryMiddleware (0.00s)
=== RUN   TestMiddlewareChain
--- PASS: TestMiddlewareChain (0.00s)
=== RUN   TestAddFieldsHelpers
--- PASS: TestAddFieldsHelpers (0.00s)
=== RUN   TestGenerateTraceID
--- PASS: TestGenerateTraceID (0.00s)
PASS
ok      github.com/Ypirate/gobase/mlog/ginmw    0.385s

测试覆盖率: 81.2%
```

## 向后兼容性

**破坏性变更：**
1. 所有日志方法签名从 `ctx any` 改为 `ctx context.Context`
2. Gin 相关函数移到 `ginmw` 包

**迁移指南：**
```go
// 旧代码
import "github.com/Ypirate/gobase/mlog"
mlog.GinAddString(c, "key", "value")
mlog.Infof(c, "message")

// 新代码
import (
    "github.com/Ypirate/gobase/mlog"
    "github.com/Ypirate/gobase/mlog/ginmw"
)
ginmw.AddString(c, "key", "value")
mlog.Infof(ginmw.GetContext(c), "message")
```

## 总结

### 达成目标

✅ **核心纯净** - 不依赖任何 Web 框架  
✅ **请求追踪** - trace-id 自动传播  
✅ **代码位置** - 自动记录文件:行号  
✅ **Panic 捕获** - 完整堆栈记录  
✅ **动态级别** - 运行时调整  
✅ **子 Logger** - 固定字段支持  
✅ **框架扩展** - 易于集成其他框架  
✅ **测试友好** - 可重新初始化  

### 代码质量

- 测试覆盖率：81.2%
- 所有测试通过
- 无 Gin 依赖（核心包）
- 线程安全
- 文档完善

### 适用场景

- ✅ HTTP 服务（Gin、Echo、Fiber 等）
- ✅ gRPC 服务
- ✅ CLI 工具
- ✅ 后台任务
- ✅ 微服务架构
- ✅ 任何需要结构化日志的 Go 项目
