// internal/server/server.go
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"niurou/internal/service"

	"github.com/gin-gonic/gin"
)

// Server HTTP服务器结构
type Server struct {
	httpServer  *http.Server
	chatService *service.ChatService
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Reply     string `json:"reply"`
	Timestamp string `json:"timestamp"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// New 创建新的HTTP服务器
func New(chatService *service.ChatService, port int) *Server {
	gin.SetMode(gin.ReleaseMode) // 设置为生产模式，减少日志输出

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	server := &Server{
		chatService: chatService,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: router,
		},
	}

	// 设置路由
	server.setupRoutes(router)

	return server
}

// setupRoutes 设置路由
func (s *Server) setupRoutes(router *gin.Engine) {
	// 健康检查
	router.GET("/health", s.healthCheck)

	// API路由组
	api := router.Group("/api/v1")
	{
		api.POST("/chat", s.handleChat)
		api.GET("/status", s.getStatus)
		api.POST("/learn", s.handleLearn)
		api.DELETE("/clear-all", s.handleClearAll) // 一键清空所有数据
	}

	// 静态文件服务（如果需要前端界面）
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")
	router.GET("/", s.indexPage)
}

// healthCheck 健康检查端点
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleChat 处理聊天请求
func (s *Server) handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// 调用聊天服务
	reply, err := s.chatService.Chat(c.Request.Context(), req.Message)
	if err != nil {
		log.Printf("Chat error: %v", err)
		c.JSON(http.StatusInternalServerError, ChatResponse{
			Success: false,
			Error:   "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, ChatResponse{
		Reply:     reply,
		Timestamp: time.Now().Format(time.RFC3339),
		Success:   true,
	})
}

// handleLearn 处理学习请求
func (s *Server) handleLearn(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ChatResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// 调用学习服务
	err := s.chatService.Learn(c.Request.Context(), req.Message)
	if err != nil {
		log.Printf("Learn error: %v", err)
		c.JSON(http.StatusInternalServerError, ChatResponse{
			Success: false,
			Error:   "Failed to learn content",
		})
		return
	}

	c.JSON(http.StatusOK, ChatResponse{
		Reply:     "Content learned successfully",
		Timestamp: time.Now().Format(time.RFC3339),
		Success:   true,
	})
}

// getStatus 获取服务状态
func (s *Server) getStatus(c *gin.Context) {
	status := s.chatService.GetStatus()
	c.JSON(http.StatusOK, status)
}

// indexPage 首页
func (s *Server) indexPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "AI Chat Assistant",
	})
}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("🚀 HTTP服务器启动在端口 %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// handleClearAll 处理清空所有数据的请求
func (s *Server) handleClearAll(c *gin.Context) {
	log.Println("🗑️ 收到清空所有数据的请求")

	// 调用ChatService的清空方法
	err := s.chatService.ClearAllData(c.Request.Context())
	if err != nil {
		log.Printf("❗️ 清空数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("清空数据失败: %v", err),
		})
		return
	}

	log.Println("✅ 所有数据已成功清空")
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "所有记忆数据已成功清空",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("🔄 正在关闭HTTP服务器...")
	return s.httpServer.Shutdown(ctx)
}
