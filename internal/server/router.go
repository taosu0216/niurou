// internal/server/server.go
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"niurou/internal/app/agent/agents"
	"niurou/internal/service"

	"github.com/gin-gonic/gin"
)

// Server HTTP服务器结构
type Server struct {
	httpServer  *http.Server
	chatService *service.ChatService

	agents map[string]*agents.Agent
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

	server.agents = agents.InitAgentPool()

	// 设置路由
	server.setupRoutes(router)

	return server
}

// setupRoutes 设置路由
func (s *Server) setupRoutes(router *gin.Engine) {

	// API路由组
	api := router.Group("/api/v1")
	{
		api.POST("/chat", s.handleChat)
		// api.POST("/learn", s.handleLearn)
		api.DELETE("/clear-all", s.handleClearAll) // 一键清空所有数据
	}

	node := router.Group("/node")
	{
		node.POST("/AddPersonNode", s.handleAddPersonNode)
		node.GET("/AddPersonNodeByAgent", s.handleAddPersonNodeByAgent)
	}

}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("🚀 HTTP服务器启动在端口 %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("🔄 正在关闭HTTP服务器...")
	return s.httpServer.Shutdown(ctx)
}
