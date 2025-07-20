// internal/tools/memoryRecovery.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"niurou/internal/configger"
	"niurou/internal/llm"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ConversationEntry 对话记录条目（避免循环导入，在tools包中重新定义）
type ConversationEntry struct {
	Timestamp time.Time `json:"timestamp"`
	UserInput string    `json:"user_input"`
	AIReply   string    `json:"ai_reply"`
	Duration  string    `json:"duration"`
}

// MemoryRecoveryClient 是专门用于记忆回收分析的任务客户端
type MemoryRecoveryClient struct {
	model            model.ToolCallingChatModel
	recoveryToolName string
}

// NewMemoryRecoveryClient 创建记忆回收客户端
func NewMemoryRecoveryClient(provider llm.Provider) (*MemoryRecoveryClient, error) {
	baseModel := provider.GetBaseModel()

	toolName := "analyze_conversation_memory"
	recoverySchema := llm.BuildMemoryRecoverySchema()
	paramsOneOf := schema.NewParamsOneOfByOpenAPIV3(recoverySchema)
	toolInfo := &schema.ToolInfo{
		Name:        toolName,
		Desc:        "分析对话记录，智能判断哪些内容值得保存到长期记忆库中。",
		ParamsOneOf: paramsOneOf,
	}

	// 为记忆回收任务创建绑定了工具的模型实例
	toolBoundModel, err := baseModel.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return nil, fmt.Errorf("绑定记忆回收工具失败: %w", err)
	}

	return &MemoryRecoveryClient{
		model:            toolBoundModel,
		recoveryToolName: toolName,
	}, nil
}

// AnalyzeConversation 分析对话记录并返回记忆回收结果
func (c *MemoryRecoveryClient) AnalyzeConversation(ctx context.Context, conversations []ConversationEntry) (*llm.MemoryRecoveryResult, error) {
	// 构建对话文本
	conversationText := c.buildConversationText(conversations)

	inputMessages := []*schema.Message{
		schema.SystemMessage(llm.MemoryRecoverySystemPrompt),
		schema.UserMessage(conversationText),
	}

	opts := []model.Option{
		model.WithModel(configger.GraphModelName), // 使用强大的模型进行分析
		model.WithToolChoice(schema.ToolChoiceForced),
	}

	resp, err := c.model.Generate(ctx, inputMessages, opts...)
	if err != nil {
		return nil, fmt.Errorf("eino Generate (MemoryRecovery) 失败: %w", err)
	}

	if len(resp.ToolCalls) == 0 {
		return &llm.MemoryRecoveryResult{}, nil
	}

	toolCall := resp.ToolCalls[0]
	if toolCall.Function.Name != c.recoveryToolName {
		return nil, fmt.Errorf("LLM调用了未知工具: %s", toolCall.Function.Name)
	}

	var result llm.MemoryRecoveryResult
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &result); err != nil {
		return nil, fmt.Errorf("反序列化记忆回收参数失败: %w。原始参数: %s", err, toolCall.Function.Arguments)
	}

	return &result, nil
}

// buildConversationText 将对话记录构建为分析用的文本格式
func (c *MemoryRecoveryClient) buildConversationText(conversations []ConversationEntry) string {
	if len(conversations) == 0 {
		return "没有对话记录。"
	}

	text := fmt.Sprintf("以下是需要分析的对话记录，共 %d 条消息：\n\n", len(conversations))

	for i, entry := range conversations {
		text += fmt.Sprintf("=== 对话 %d ===\n", i+1)
		text += fmt.Sprintf("时间: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
		text += fmt.Sprintf("用户: %s\n", entry.UserInput)
		text += fmt.Sprintf("AI: %s\n", entry.AIReply)
		text += fmt.Sprintf("耗时: %s\n\n", entry.Duration)
	}

	text += "请分析以上对话记录，判断哪些内容值得保存到长期记忆库中。"
	return text
}

// WorthyConversationSegment 值得保存的对话片段
type WorthyConversationSegment struct {
	Conversations      []ConversationEntry     `json:"conversations"`
	ValueScore         int                     `json:"value_score"`
	ValueReason        string                  `json:"value_reason"`
	Categories         []string                `json:"categories"`
	ExtractedText      string                  `json:"extracted_text"`
	ExtractedKnowledge *llm.ExtractedKnowledge `json:"extracted_knowledge,omitempty"`
}

// FilterWorthySegments 根据分析结果过滤出值得保存的对话片段
func (c *MemoryRecoveryClient) FilterWorthySegments(conversations []ConversationEntry, result *llm.MemoryRecoveryResult) []WorthyConversationSegment {
	var segments []WorthyConversationSegment

	for _, segment := range result.WorthySegments {
		// 验证索引范围
		if len(segment.SegmentIndex) != 2 {
			continue
		}

		start := segment.SegmentIndex[0]
		end := segment.SegmentIndex[1]

		if start < 0 || end >= len(conversations) || start > end {
			continue
		}

		// 提取对应的对话片段
		segmentConversations := conversations[start : end+1]

		segments = append(segments, WorthyConversationSegment{
			Conversations:      segmentConversations,
			ValueScore:         segment.ValueScore,
			ValueReason:        segment.ValueReason,
			Categories:         segment.Categories,
			ExtractedText:      segment.ExtractedText,
			ExtractedKnowledge: segment.ExtractedKnowledge,
		})
	}

	return segments
}

// GetHighValueSegments 获取高价值片段（评分>=7）
func (c *MemoryRecoveryClient) GetHighValueSegments(segments []WorthyConversationSegment) []WorthyConversationSegment {
	var highValueSegments []WorthyConversationSegment

	for _, segment := range segments {
		if segment.ValueScore >= 7 {
			highValueSegments = append(highValueSegments, segment)
		}
	}

	return highValueSegments
}

// ShouldSaveConversation 判断整个对话是否值得保存
func (c *MemoryRecoveryClient) ShouldSaveConversation(result *llm.MemoryRecoveryResult) bool {
	// 如果整体价值评分>=5，或者有任何高价值片段，就值得保存
	if result.ConversationAnalysis.OverallValue >= 5 {
		return true
	}

	for _, segment := range result.WorthySegments {
		if segment.ValueScore >= 7 {
			return true
		}
	}

	return false
}
