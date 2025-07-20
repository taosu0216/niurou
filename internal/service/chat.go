// internal/service/chat.go
package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"niurou/internal/app/agent"
	"niurou/internal/app/agent/agents"
	"niurou/internal/app/tools"
	"niurou/internal/data/memManager"
)

// ChatService 聊天服务
type ChatService struct {
	agent           *agents.Agent
	memManager      memManager.Manager
	conversationLog []ConversationEntry
	mu              sync.RWMutex
	startTime       time.Time
	messageCount    int
}

// ConversationEntry 对话记录条目
type ConversationEntry struct {
	Timestamp time.Time `json:"timestamp"`
	UserInput string    `json:"user_input"`
	AIReply   string    `json:"ai_reply"`
	Duration  string    `json:"duration"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Status           string    `json:"status"`
	StartTime        time.Time `json:"start_time"`
	Uptime           string    `json:"uptime"`
	MessageCount     int       `json:"message_count"`
	ConversationSize int       `json:"conversation_size"`
	LastActivity     time.Time `json:"last_activity,omitempty"`
}

// New 创建新的聊天服务
func New(ctx context.Context) (*ChatService, error) {
	log.Println("🤖 正在初始化聊天服务...")

	// 初始化Agent
	agentInstance := agents.GetNiurouAgent()

	// 复用Agent的MemManager，避免重复初始化ONNX Runtime
	memManagerInstance := agentInstance.GetMemManager()

	service := &ChatService{
		agent:           agentInstance,
		memManager:      memManagerInstance,
		conversationLog: make([]ConversationEntry, 0),
		startTime:       time.Now(),
		messageCount:    0,
	}

	log.Println("✅ 聊天服务初始化成功！")
	return service, nil
}

// Chat 处理聊天请求
func (s *ChatService) Chat(ctx context.Context, userInput string) (string, error) {
	startTime := time.Now()

	log.Printf("💬 收到用户消息: %s", userInput)

	// 调用Agent进行对话
	reply, err := s.agent.Respond(ctx, userInput)
	if err != nil {
		return "", fmt.Errorf("Agent响应失败: %w", err)
	}

	duration := time.Since(startTime)

	// 记录对话
	s.mu.Lock()
	s.conversationLog = append(s.conversationLog, ConversationEntry{
		Timestamp: startTime,
		UserInput: userInput,
		AIReply:   reply,
		Duration:  duration.String(),
	})
	s.messageCount++
	s.mu.Unlock()

	log.Printf("✅ AI回复完成，耗时: %v", duration)
	return reply, nil
}

// // Learn 处理学习请求
// func (s *ChatService) Learn(ctx context.Context, content string) error {
// 	log.Printf("📚 开始学习内容: %s", content)

// 	_, err := s.agent.IngestAndLearn(ctx, content)
// 	if err != nil {
// 		return fmt.Errorf("学习内容失败: %w", err)
// 	}

// 	log.Println("✅ 内容学习完成")
// 	return nil
// }

// GetConversationLog 获取对话记录
func (s *ChatService) GetConversationLog() []ConversationEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本以避免并发问题
	log := make([]ConversationEntry, len(s.conversationLog))
	copy(log, s.conversationLog)
	return log
}

// // SaveConversationToMemory 将对话记录保存到记忆库
// // 使用智能记忆回收Agent进行价值判断和选择性保存
// func (s *ChatService) SaveConversationToMemory(ctx context.Context) error {
// 	s.mu.RLock()
// 	conversationLog := s.GetConversationLog()
// 	s.mu.RUnlock()

// 	if len(conversationLog) == 0 {
// 		log.Println("📝 没有对话记录需要保存")
// 		return nil
// 	}

// 	log.Printf("🧠 启动智能记忆回收，分析 %d 条对话记录...", len(conversationLog))

// 	// 1. 创建记忆回收Agent
// 	memoryAgent, err := agent.NewMemoryRecoveryAgent(ctx)
// 	if err != nil {
// 		log.Printf("⚠️ 记忆回收Agent创建失败，回退到传统保存方式: %v", err)
// 		return s.fallbackSaveConversation(ctx, conversationLog)
// 	}
// 	defer memoryAgent.Close()

// 	// 2. 转换对话格式（从service.ConversationEntry到tools.ConversationEntry）
// 	toolsConversations := s.convertConversationEntries(conversationLog)

// 	// 3. 使用记忆回收Agent处理对话
// 	report, err := memoryAgent.ProcessConversationMemory(ctx, toolsConversations)
// 	if err != nil {
// 		log.Printf("⚠️ 智能记忆回收失败，回退到传统保存方式: %v", err)
// 		return s.fallbackSaveConversation(ctx, conversationLog)
// 	}

// 	// 4. 输出处理报告
// 	s.logMemoryRecoveryReport(report)

// 	return nil
// }

// convertConversationEntries 转换对话记录格式
func (s *ChatService) convertConversationEntries(conversations []ConversationEntry) []tools.ConversationEntry {
	toolsConversations := make([]tools.ConversationEntry, len(conversations))
	for i, conv := range conversations {
		toolsConversations[i] = tools.ConversationEntry{
			Timestamp: conv.Timestamp,
			UserInput: conv.UserInput,
			AIReply:   conv.AIReply,
			Duration:  conv.Duration,
		}
	}
	return toolsConversations
}

// // fallbackSaveConversation 传统的对话保存方式（作为备用方案）
// func (s *ChatService) fallbackSaveConversation(ctx context.Context, conversationLog []ConversationEntry) error {
// 	log.Println("📝 使用传统方式保存对话记录...")

// 	// 构建对话摘要
// 	summary := s.buildConversationSummary(conversationLog)

// 	// 保存到记忆库
// 	_, err := s.agent.IngestAndLearn(ctx, summary)
// 	if err != nil {
// 		return fmt.Errorf("保存对话记录失败: %w", err)
// 	}

// 	log.Println("✅ 对话记录已保存到记忆库（传统方式）")
// 	return nil
// }

// logMemoryRecoveryReport 输出记忆回收处理报告
func (s *ChatService) logMemoryRecoveryReport(report *agent.MemoryRecoveryReport) {
	if !report.ShouldSave {
		log.Printf("📝 智能分析结果：%s", report.SkippedReason)
		return
	}

	log.Printf("🎯 智能记忆回收完成:")
	log.Printf("   📊 整体价值评分: %d/10", report.AnalysisResult.ConversationAnalysis.OverallValue)
	log.Printf("   📈 总片段数: %d", report.TotalSegments)
	log.Printf("   💎 高价值片段: %d", report.HighValueSegments)
	log.Printf("   💾 成功保存: %d", report.SavedSegments)

	if len(report.ProcessingErrors) > 0 {
		log.Printf("   ⚠️ 处理错误: %d 个", len(report.ProcessingErrors))
		for i, err := range report.ProcessingErrors {
			log.Printf("      %d. %s", i+1, err)
		}
	}

	// 输出对话主题
	themes := report.AnalysisResult.ConversationAnalysis.ConversationThemes
	if len(themes) > 0 {
		log.Printf("   🏷️ 对话主题: %v", themes)
	}

	log.Printf("   📝 对话摘要: %s", report.AnalysisResult.ConversationAnalysis.Summary)
}

// buildConversationSummary 构建对话摘要
func (s *ChatService) buildConversationSummary(conversations []ConversationEntry) string {
	if len(conversations) == 0 {
		return ""
	}

	startTime := conversations[0].Timestamp
	endTime := conversations[len(conversations)-1].Timestamp

	summary := fmt.Sprintf("对话会话记录 - 时间: %s 到 %s, 共 %d 条消息\n\n",
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"),
		len(conversations))

	// 只保存最重要的对话内容，避免过长
	maxEntries := 10
	if len(conversations) > maxEntries {
		summary += fmt.Sprintf("（显示最近 %d 条对话）\n\n", maxEntries)
		conversations = conversations[len(conversations)-maxEntries:]
	}

	for i, entry := range conversations {
		summary += fmt.Sprintf("%d. 用户: %s\n", i+1, entry.UserInput)
		summary += fmt.Sprintf("   AI: %s\n\n", entry.AIReply)
	}

	return summary
}

// ClearAllData 一键清空所有记忆数据
func (s *ChatService) ClearAllData(ctx context.Context) error {
	log.Println("🗑️ ChatService: 开始清空所有记忆数据...")

	if s.memManager == nil {
		return fmt.Errorf("memManager未初始化")
	}

	err := s.memManager.ClearAllData(ctx)
	if err != nil {
		log.Printf("❗️ ChatService: 清空数据失败: %v", err)
		return fmt.Errorf("清空数据失败: %w", err)
	}

	log.Println("✅ ChatService: 所有记忆数据已清空")
	return nil
}

// Close 关闭服务
func (s *ChatService) Close() {
	log.Println("🔄 正在关闭聊天服务...")
	if s.agent != nil {
		s.agent.Close()
	}
	log.Println("✅ 聊天服务已关闭")
}
