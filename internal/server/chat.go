package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"niurou/types"

	"github.com/gin-gonic/gin"
)

// handleChat 处理聊天请求
func (s *Server) handleChat(c *gin.Context) {
	var req types.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ChatResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// 调用聊天服务
	reply, err := s.chatService.Chat(c.Request.Context(), req.Message)
	if err != nil {
		log.Printf("Chat error: %v", err)
		c.JSON(http.StatusInternalServerError, types.ChatResponse{
			Success: false,
			Error:   "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, types.ChatResponse{
		Reply:     reply,
		Timestamp: time.Now().Format(time.RFC3339),
		Success:   true,
	})
}

// // handleLearn 处理学习请求
// func (s *Server) handleLearn(c *gin.Context) {
// 	var req types.ChatRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, types.ChatResponse{
// 			Success: false,
// 			Error:   "Invalid request format",
// 		})
// 		return
// 	}

// 	// 调用学习服务
// 	err := s.chatService.Learn(c.Request.Context(), req.Message)
// 	if err != nil {
// 		log.Printf("Learn error: %v", err)
// 		c.JSON(http.StatusInternalServerError, types.ChatResponse{
// 			Success: false,
// 			Error:   "Failed to learn content",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, types.ChatResponse{
// 		Reply:     "Content learned successfully",
// 		Timestamp: time.Now().Format(time.RFC3339),
// 		Success:   true,
// 	})
// }

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
