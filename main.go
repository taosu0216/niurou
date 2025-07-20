package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"niurou/internal/graceful"
	"niurou/internal/server"
	"niurou/internal/service"
)

func main() {
	// 创建主上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建优雅退出管理器
	shutdownManager := graceful.New()
	shutdownManager.Start()

	// 启动应用
	if err := runApplication(ctx, shutdownManager); err != nil {
		log.Fatalf("应用启动失败: %v", err)
	}

	// 等待优雅退出完成
	shutdownManager.Wait()
	log.Println("🏁 应用已安全退出")
}

// runApplication 启动应用程序
func runApplication(ctx context.Context, shutdownManager *graceful.ShutdownManager) error {
	log.Println("🚀 正在启动AI聊天助手...")

	// 1. 初始化聊天服务
	chatService, err := service.New(ctx)
	if err != nil {
		return fmt.Errorf("初始化聊天服务失败: %w", err)
	}

	// 2. 创建HTTP服务器
	httpServer := server.New(chatService, 8080)

	// 3. 注册优雅退出函数
	// shutdownManager.RegisterShutdownFunc(graceful.LogShutdownFunc("对话记忆保存", func(ctx context.Context) error {
	// 	return chatService.SaveConversationToMemory(ctx)
	// }))

	shutdownManager.RegisterShutdownFunc(graceful.LogShutdownFunc("HTTP服务器", func(ctx context.Context) error {
		return httpServer.Shutdown(ctx)
	}))

	shutdownManager.RegisterShutdownFunc(graceful.LogShutdownFunc("聊天服务", func(ctx context.Context) error {
		chatService.Close()
		return nil
	}))

	// 4. 启动HTTP服务器
	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("❗️ HTTP服务器启动失败: %v", err)
		}
	}()

	log.Println("✅ AI聊天助手启动成功!")
	log.Println("📡 HTTP API: http://localhost:8080")
	log.Println("🏥 健康检查: http://localhost:8080/health")
	log.Println("💬 聊天API: POST http://localhost:8080/api/v1/chat")
	log.Println("📚 学习API: POST http://localhost:8080/api/v1/learn")
	log.Println("📊 状态API: GET http://localhost:8080/api/v1/status")
	log.Println("🛑 按 Ctrl+C 优雅退出")

	return nil
}
