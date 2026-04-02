# mlog - 企业级 Go 日志库

基于 `zap` + `lumberjack` 的企业级日志库，核心专注于日志功能，框架集成通过独立包提供。

## 设计理念

- **核心纯净** - mlog 核心只依赖 `context.Context`，不依赖任何 Web 框架
- **框架解耦** - Gin 等框架的集成放在独立的 `ginmw` 包中
- **灵活扩展** - 通过 `context.Context` 传递字段，业务可自定义字段映射

## 特性

- ✅ **Context 字段传递** - 通过 context 携带日志字段
- ✅ **请求全链路追踪** - 一个请求的所有日志都带有相同的 trace-id
- ✅ **代码位置记录** - 自动记录日志打印的文件名和行号
- ✅ **动态日志级别** - 运行时动态调整日志级别
- ✅ **子 Logger** - 支持创建带固定字段的子 Logger
- ✅ **结构化日志** - JSON 格式输出，便于日志分析
- ✅ **日志轮转** - 自动按大小和时间轮转日志文件
- ✅ **线程安全** - 支持并发调用和重新初始化
- ✅ **自动堆栈跟踪** - Error 级别及以上自动记录调用栈

## 快速开始

### 1. 初始化日志

```go
package main

import (
    "context"
    "github.com/Ypirate/gobase/mlog"
)

func main() {
    // 初始化日志
    mlog.InitLog(mlog.LogConfig{
        Level:       "info",      // 日志级别: debug/info/warn/error
        Stdout:      true,        // 是否输出到标准输出
        LogDir:      "./logs",    // 日志目录
        LogFileName: "app.log",   // 日志文件名
    })
    defer mlog.CloseLogger()

    // 使用日志
    ctx := context.Background()
    mlog.Infof(ctx, "Application started")
}
```

### 2. 基础日志使用

```go
import (
    "context"
    "github.com/Ypirate/gobase/mlog"
    "go.uber.org/zap"
)

func BusinessLogic(ctx context.Context) {
    // 格式化日志
    mlog.Infof(ctx, "Processing request for user %s", "user123")
    mlog.Debugf(ctx, "Debug info: %v", someData)
    mlog.Warnf(ctx, "Warning: %s", warningMsg)
    mlog.Errorf(ctx, "Error occurred: %v", err)

    // 结构化日志（推荐）
    mlog.Info(ctx, "User login",
        zap.String("user_id", "user123"),
        zap.Int("age", 25),
        zap.Bool("active", true),
    )
}
```

### 3. 添加自定义字段到 Context

```go
func HandleRequest(ctx context.Context) {
    // 添加字段到 context
    ctx = mlog.AddFields(ctx,
        zap.String("trace_id", "abc123"),
        zap.String("user_id", "user456"),
        zap.String("request_id", "req789"),
    )

    // 后续所有日志都会自动包含这些字段
    mlog.Infof(ctx, "Processing request")
    // 输出: {"level":"info","msg":"Processing request","trace_id":"abc123","user_id":"user456","request_id":"req789"}

    // 调用其他函数，传递 context
    DoSomething(ctx)
}

func DoSomething(ctx context.Context) {
    // 自动继承上层 context 的字段
    mlog.Infof(ctx, "Doing something")
    // 输出: {"level":"info","msg":"Doing something","trace_id":"abc123","user_id":"user456","request_id":"req789"}
}
```

### 4. 使用子 Logger

```go
// 创建带固定字段的子 Logger
var (
    userLogger  = mlog.With(zap.String("module", "user"))
    orderLogger = mlog.With(zap.String("module", "order"), zap.String("service", "payment"))
)

func UserService(ctx context.Context) {
    // 子 Logger 的所有日志都会自动带上固定字段
    userLogger.Infof(ctx, "User login attempt")
    // 输出: {"level":"info","msg":"User login attempt","module":"user"}

    // 结合 context 字段
    ctx = mlog.AddFields(ctx, zap.String("user_id", "user123"))
    userLogger.Infof(ctx, "Login successful")
    // 输出: {"level":"info","msg":"Login successful","module":"user","user_id":"user123"}
}
```

### 5. 动态调整日志级别

```go
// 运行时动态调整日志级别
mlog.SetLevel("debug")  // 开启 debug 日志
mlog.SetLevel("error")  // 只记录 error 及以上

// 获取当前日志级别
level := mlog.GetLevel()
fmt.Println("Current level:", level)
```

## Gin 框架集成

使用独立的 `ginmw` 包提供 Gin 中间件：

```go
import (
    "github.com/Ypirate/gobase/mlog"
    "github.com/Ypirate/gobase/mlog/ginmw"
    "github.com/gin-gonic/gin"
)

func main() {
    mlog.InitLog(mlog.LogConfig{Level: "info", Stdout: true})

    r := gin.New()

    // 使用中间件（推荐顺序）
    r.Use(ginmw.TraceID())    // 1. Trace-ID 注入
    r.Use(ginmw.Recovery())   // 2. Panic 恢复
    r.Use(ginmw.AccessLog())  // 3. 请求日志

    r.GET("/user/:id", func(c *gin.Context) {
        // 获取 context（已包含 trace-id）
        ctx := ginmw.GetContext(c)

        // 添加自定义字段
        ginmw.AddString(c, "user_id", c.Param("id"))

        // 使用日志
        mlog.Infof(ctx, "Fetching user")

        c.JSON(200, gin.H{"status": "ok"})
    })

    r.Run(":8080")
}
```

### Gin 中间件说明

#### TraceID 中间件
- 从 HTTP Header `X-Trace-ID` 提取 trace-id
- 如果不存在，自动生成格式为 `timestamp-random8` 的 trace-id
- 注入到 context，后续所有日志自动包含
- 设置响应 Header `X-Trace-ID`

#### AccessLog 中间件
记录每个 HTTP 请求的详细信息：
- `method` - HTTP 方法
- `path` - 请求路径（包含 query）
- `status` - 响应状态码
- `latency` - 请求耗时
- `client_ip` - 客户端 IP
- `user_agent` - User-Agent

#### Recovery 中间件
- 捕获 handler 中的 panic
- 记录 panic 值和完整调用栈
- 返回 500 错误响应
- 保证服务继续运行

### Gin 辅助函数

```go
// 添加字段到 Gin context
ginmw.AddString(c, "user_id", "user123")
ginmw.AddInt(c, "age", 25)
ginmw.AddBool(c, "active", true)
ginmw.AddFields(c, zap.String("key", "value"), zap.Int("count", 10))

// 获取 request context（用于日志）
ctx := ginmw.GetContext(c)
mlog.Infof(ctx, "Processing request")
```

## 日志输出示例

```json
{
  "level": "info",
  "ts": "2025-01-15T10:30:45.123456789+08:00",
  "caller": "handler/user.go:42",
  "msg": "User login successful",
  "trace_id": "1736908245123456789-a1b2c3d4",
  "user_id": "user123",
  "module": "user"
}
```

## API 参考

### 核心日志方法

**格式化日志：**
- `Debugf(ctx, format, args...)` - Debug 级别
- `Infof(ctx, format, args...)` - Info 级别
- `Warnf(ctx, format, args...)` - Warn 级别
- `Errorf(ctx, format, args...)` - Error 级别（自动记录堆栈）
- `Fatalf(ctx, format, args...)` - Fatal 级别（程序退出）
- `Panicf(ctx, format, args...)` - Panic 级别（触发 panic）

**结构化日志（推荐）：**
- `Debug(ctx, msg, fields...)` - Debug 级别
- `Info(ctx, msg, fields...)` - Info 级别
- `Warn(ctx, msg, fields...)` - Warn 级别
- `Error(ctx, msg, fields...)` - Error 级别

### Context 字段管理

- `AddFields(ctx, fields...)` - 添加字段到 context，返回新的 context

### 子 Logger

- `With(fields...)` - 创建带固定字段的子 Logger

### 动态级别

- `SetLevel(level)` - 设置日志级别
- `GetLevel()` - 获取当前日志级别

### 初始化与关闭

- `InitLog(config)` - 初始化日志（可多次调用）
- `CloseLogger()` - 关闭日志（刷新缓冲区）

## 配置说明

```go
type LogConfig struct {
    Level       string  // 日志级别: debug/info/warn/error/dpanic/panic/fatal
    Stdout      bool    // 是否输出到标准输出
    LogDir      string  // 日志目录（默认 ./logs）
    LogFileName string  // 日志文件名（默认 app.log）
}
```

日志轮转配置（内置）：
- `MaxSize`: 100MB - 单个日志文件最大大小
- `MaxBackups`: 5 - 保留的旧日志文件数量
- `MaxAge`: 7 天 - 日志文件保留天数
- `Compress`: true - 是否压缩旧日志文件

## 最佳实践

1. **使用 context.Context 传递字段** - 不要依赖全局变量或框架特定的 context
2. **子 Logger 用于模块** - 为不同模块创建子 Logger，避免重复添加相同字段
3. **结构化日志优先** - 优先使用 `Info/Error` 等方法配合 `zap.Field`
4. **错误日志自动堆栈** - Error 级别及以上自动包含调用栈，无需手动添加
5. **框架集成独立** - 框架相关代码放在独立包中（如 `ginmw`），保持核心纯净

## 扩展其他框架

参考 `ginmw` 包的实现，为其他框架创建类似的中间件包：

```go
// 示例：为其他框架创建中间件
package yourframeworkmw

import (
    "github.com/Ypirate/gobase/mlog"
    "go.uber.org/zap"
)

func TraceIDMiddleware() YourFrameworkHandler {
    return func(ctx YourFrameworkContext) {
        traceID := extractOrGenerateTraceID(ctx)
        newCtx := mlog.AddFields(ctx.Context(), zap.String("trace_id", traceID))
        ctx.SetContext(newCtx)
        ctx.Next()
    }
}
```

## License

MIT
