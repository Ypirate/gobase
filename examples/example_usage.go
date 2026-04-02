package main

import (
	"context"

	"github.com/Ypirate/gobase/mlog"
	"github.com/Ypirate/gobase/mlog/ginmw"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 为不同模块创建子 Logger
var (
	userLogger  = mlog.With(zap.String("module", "user"))
	orderLogger = mlog.With(zap.String("module", "order"))
)

func main() {
	// 初始化日志
	mlog.InitLog(mlog.LogConfig{
		Level:       "info",
		Stdout:      true,
		LogDir:      "./logs",
		LogFileName: "app.log",
	})
	defer mlog.CloseLogger()

	// 创建 Gin 路由
	r := gin.New()

	// 使用中间件（推荐顺序）
	r.Use(ginmw.TraceID())   // 1. Trace-ID 注入
	r.Use(ginmw.Recovery())  // 2. Panic 恢复
	r.Use(ginmw.AccessLog()) // 3. 请求日志

	// 注册路由
	r.GET("/hello", HelloHandler)
	r.GET("/user/:id", GetUserHandler)
	r.POST("/order", CreateOrderHandler)
	r.GET("/panic", PanicHandler)

	mlog.Infof(context.Background(), "Server starting on :8080")
	r.Run(":8080")
}

// HelloHandler 基础示例
func HelloHandler(c *gin.Context) {
	ctx := ginmw.GetContext(c)
	mlog.Infof(ctx, "Processing hello request")
	c.JSON(200, gin.H{"message": "hello"})
}

// GetUserHandler 使用子 Logger 和自定义字段
func GetUserHandler(c *gin.Context) {
	userID := c.Param("id")

	// 添加自定义字段
	ginmw.AddString(c, "user_id", userID)

	ctx := ginmw.GetContext(c)

	// 使用子 Logger
	userLogger.Infof(ctx, "Fetching user details")

	// 模拟业务逻辑
	user := map[string]interface{}{
		"id":   userID,
		"name": "John Doe",
		"age":  30,
	}

	// 使用结构化日志
	mlog.Info(ctx, "User fetched successfully",
		zap.String("user_id", userID),
		zap.String("name", user["name"].(string)),
	)

	c.JSON(200, user)
}

// CreateOrderHandler 使用子 Logger 记录订单创建
func CreateOrderHandler(c *gin.Context) {
	var req struct {
		ProductID string  `json:"product_id"`
		Quantity  int     `json:"quantity"`
		Amount    float64 `json:"amount"`
	}

	ctx := ginmw.GetContext(c)

	if err := c.ShouldBindJSON(&req); err != nil {
		mlog.Errorf(ctx, "Invalid request: %v", err)
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	// 添加订单相关字段
	ginmw.AddString(c, "product_id", req.ProductID)
	ginmw.AddInt(c, "quantity", req.Quantity)

	// 使用子 Logger
	orderLogger.Infof(ctx, "Creating order")

	// 模拟订单创建
	orderID := "order-12345"

	// 使用结构化日志记录订单创建成功
	mlog.Info(ctx, "Order created successfully",
		zap.String("order_id", orderID),
		zap.String("product_id", req.ProductID),
		zap.Int("quantity", req.Quantity),
		zap.Float64("amount", req.Amount),
	)

	c.JSON(200, gin.H{
		"order_id": orderID,
		"status":   "created",
	})
}

// PanicHandler 演示 panic 捕获
func PanicHandler(c *gin.Context) {
	ctx := ginmw.GetContext(c)
	mlog.Warnf(ctx, "About to panic")

	// 触发 panic（会被 Recovery 中间件捕获）
	panic("something went wrong!")
}
