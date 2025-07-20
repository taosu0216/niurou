// internal/tools/knowledge_extraction.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"niurou/internal/configger"
	"niurou/internal/llm" // 只依赖基础的 llm.Provider 和 llm.Schema

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// KnowledgeExtractorClient 是一个专门用于知识提取的任务客户端。
type KnowledgeExtractorClient struct {
	model             model.ToolCallingChatModel
	knowledgeToolName string
}

// NewKnowledgeExtractorClient 是该客户端的构造函数。
// 它接收一个通用的 llm.Provider，然后在内部构建自己专用的、绑定了工具的模型。
func NewKnowledgeExtractorClient(provider llm.Provider) (*KnowledgeExtractorClient, error) {
	baseModel := provider.GetBaseModel()

	toolName := "save_extracted_knowledge"
	knowledgeSchema := llm.BuildKnowledgeExtractionSchema()
	paramsOneOf := schema.NewParamsOneOfByOpenAPIV3(knowledgeSchema)
	toolInfo := &schema.ToolInfo{
		Name:        toolName,
		Desc:        "从文本中提取结构化的知识图谱实体和关系。",
		ParamsOneOf: paramsOneOf,
	}

	// 为这个特定任务创建一个绑定了工具的模型实例
	toolBoundModel, err := baseModel.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return nil, fmt.Errorf("绑定知识提取工具失败: %w", err)
	}

	return &KnowledgeExtractorClient{
		model:             toolBoundModel,
		knowledgeToolName: toolName,
	}, nil
}

// Extract 是该客户端的核心方法，执行知识提取。
func (c *KnowledgeExtractorClient) Extract(ctx context.Context, text string) (*llm.ExtractedKnowledge, error) {
	inputMessages := []*schema.Message{
		schema.SystemMessage(llm.GraphSystemPrompt),
		schema.UserMessage(text),
	}
	opts := []model.Option{
		model.WithModel(configger.GraphModelName), // 使用强大的模型进行提取
		model.WithToolChoice(schema.ToolChoiceForced),
	}

	resp, err := c.model.Generate(ctx, inputMessages, opts...)
	if err != nil {
		return nil, fmt.Errorf("eino Generate (ExtractKnowledge) 失败: %w", err)
	}

	if len(resp.ToolCalls) == 0 {
		return &llm.ExtractedKnowledge{}, nil
	}
	toolCall := resp.ToolCalls[0]
	if toolCall.Function.Name != c.knowledgeToolName {
		return nil, fmt.Errorf("LLM调用了未知工具: %s", toolCall.Function.Name)
	}

	var knowledge llm.ExtractedKnowledge
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &knowledge); err != nil {
		return nil, fmt.Errorf("反序列化知识提取参数失败: %w。原始参数: %s", err, toolCall.Function.Arguments)
	}
	return &knowledge, nil
}
