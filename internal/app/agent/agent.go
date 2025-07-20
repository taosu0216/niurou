package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"niurou/internal/app/llm"
	"niurou/internal/app/tools"
	"niurou/internal/configger"
	"niurou/internal/data/graphDB"
	"niurou/internal/data/memManager"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Agent 是我们系统的顶层协调器。
type Agent struct {
	// Agent 声明它需要的能力客户端
	knowledgeExtractor *tools.KnowledgeExtractorClient
	memorySearcher     *tools.MemorySearchTool
	updateMemoryTool   *tools.UpdateMemoryTool

	// Agent 持有一个绑定了【对话工具】的通用对话模型
	conversationalModel model.ToolCallingChatModel
	memManager          memManager.Manager
	dialogHistory       []*schema.Message
}

// New 是 Agent 的构造函数，负责组装所有依赖。
func New(ctx context.Context) (*Agent, error) {
	// 1. 初始化最底层的依赖
	llmProvider, err := llm.NewProvider(ctx)
	if err != nil {
		return nil, err
	}
	memManager, err := memManager.InitMemClient()
	if err != nil {
		return nil, err
	}

	// 2. 初始化所有需要的“能力客户端”和“工具”，并进行依赖注入
	knowledgeExtractor, err := tools.NewKnowledgeExtractorClient(llmProvider)
	if err != nil {
		return nil, err
	}
	memorySearcher, err := tools.NewMemorySearchTool(memManager)
	if err != nil {
		return nil, err
	}
	updateMemoryTool, err := tools.NewUpdateMemoryTool(memManager)
	if err != nil {
		return nil, err
	}

	// 3. 为Agent的对话循环，创建一个绑定了所有【可对话】工具的模型
	// 注意：知识提取工具不是对话工具，所以不在这里绑定
	allDialogTools := []*schema.ToolInfo{
		memorySearcher.ToolInfo,
		updateMemoryTool.ToolInfo,
	}
	conversationalModel, err := llmProvider.GetBaseModel().WithTools(allDialogTools)
	if err != nil {
		return nil, fmt.Errorf("为Agent绑定对话工具失败: %w", err)
	}

	// 4. 添加系统提示词来指导Agent使用工具
	dialogHistory := []*schema.Message{schema.SystemMessage(llm.AgentSystemPrompt)}

	log.Println("✅ Agent 初始化成功！")
	return &Agent{
		knowledgeExtractor:  knowledgeExtractor,
		memorySearcher:      memorySearcher,
		updateMemoryTool:    updateMemoryTool,
		conversationalModel: conversationalModel,
		memManager:          memManager,
		dialogHistory:       dialogHistory,
	}, nil
}

// IngestAndLearn 是新的知识入口点。
func (a *Agent) IngestAndLearn(ctx context.Context, text string) (string, error) {
	log.Println("Agent: 正在调用知识提取器...")
	extractedKnowledge, err := a.knowledgeExtractor.Extract(ctx, text)
	if err != nil {
		return "", fmt.Errorf("Agent知识提取失败: %w", err)
	}

	log.Println("Agent: 正在处理提取的知识...")
	processedGraph, err := a.processExtractedKnowledge(extractedKnowledge)
	if err != nil {
		return "", fmt.Errorf("Agent知识处理失败: %w", err)
	}

	log.Println("Agent: 正在调用记忆管理器存储知识...")
	return a.memManager.AddMemory(ctx, processedGraph, text)
}

// Respond 是核心的、支持工具调用的多轮对话方法。
func (a *Agent) Respond(ctx context.Context, userInput string) (string, error) {
	// 将用户的当前输入添加到对话历史中
	if userInput != "" {
		a.dialogHistory = append(a.dialogHistory, schema.UserMessage(userInput))
	}

	// 1. 调用LLM，让其自主决策
	log.Println("Agent: 正在思考...")
	opts := []model.Option{model.WithModel(configger.ChatModelName)}
	resp, err := a.conversationalModel.Generate(ctx, a.dialogHistory, opts...)
	if err != nil {
		return "", fmt.Errorf("agent思考失败: %w", err)
	}

	// 2. 分析LLM的响应
	if len(resp.ToolCalls) > 0 {
		log.Println("Agent: 决定使用工具。")
		a.dialogHistory = append(a.dialogHistory, resp) // 将LLM的思考过程(工具调用)加入历史

		toolCall := resp.ToolCalls[0] // 简化处理，只执行第一个工具调用
		var toolResult string
		var toolErr error

		// 3. 执行工具
		switch toolCall.Function.Name {
		case a.memorySearcher.ToolInfo.Name:
			var args tools.MemorySearchInput
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("解析记忆搜索工具参数失败: %w", err)
			}
			toolResult, toolErr = a.memorySearcher.GoFunc(ctx, &args)
		case a.updateMemoryTool.ToolInfo.Name:
			log.Printf("Agent: 执行记忆更新工具，参数: %s", toolCall.Function.Arguments)
			toolResult, toolErr = a.updateMemoryTool.Execute(ctx, toolCall.Function.Arguments)
		default:
			toolResult = fmt.Sprintf("未知的工具: %s", toolCall.Function.Name)
			toolErr = errors.New("unknown tool")
		}

		if toolErr != nil {
			log.Printf("❗️ Agent: 工具执行出错: %v", toolErr)
		}

		// 4. 将工具执行结果作为新的信息，再次喂给LLM进行总结
		log.Println("Agent: 已获得工具结果，正在进行总结...")
		a.dialogHistory = append(a.dialogHistory, schema.ToolMessage(toolResult, toolCall.ID, schema.WithToolName(toolCall.Function.Name)))

		// 再次调用，但不接收新的用户输入，让LLM基于工具结果进行回应
		return a.Respond(ctx, "")

	} else {
		// --- LLM决定直接回答 ---
		log.Println("Agent: 决定直接回答。")
		finalAnswer := resp.Content
		a.dialogHistory = append(a.dialogHistory, resp) // 将最终答案也加入历史
		return finalAnswer, nil
	}
}

// processExtractedKnowledge 将llm DTO转换为graphDB DTO
func (a *Agent) processExtractedKnowledge(knowledge *llm.ExtractedKnowledge) (*graphDB.KnowledgeGraph, error) {
	if knowledge == nil {
		return nil, fmt.Errorf("knowledge is nil")
	}

	// 1. 转换实体
	nodes := make([]graphDB.Node, 0, len(knowledge.Entities))
	for _, ent := range knowledge.Entities {
		var props map[string]interface{}
		if len(ent.Properties) > 0 {
			if err := json.Unmarshal(ent.Properties, &props); err != nil {
				return nil, fmt.Errorf("实体 '%s' 的属性反序列化失败: %w", ent.Name, err)
			}
		} else {
			props = make(map[string]interface{})
		}
		node := graphDB.Node{
			Name:       ent.Name,
			Labels:     ent.Labels,
			Properties: props,
		}
		nodes = append(nodes, node)
	}

	// 2. 转换关系
	edges := make([]graphDB.Edge, 0, len(knowledge.Relations))
	for _, rel := range knowledge.Relations {
		var props map[string]interface{}
		if len(rel.Properties) > 0 {
			if err := json.Unmarshal(rel.Properties, &props); err != nil {
				return nil, fmt.Errorf("关系 '%s-%s-%s' 的属性反序列化失败: %w", rel.Subject, rel.Predicate, rel.Object, err)
			}
		} else {
			props = make(map[string]interface{})
		}
		edge := graphDB.Edge{
			FromNodeName: rel.Subject,
			ToNodeName:   rel.Object,
			Type:         rel.Predicate,
			Properties:   props,
		}
		edges = append(edges, edge)
	}

	kg := &graphDB.KnowledgeGraph{
		Nodes: nodes,
		Edges: edges,
	}
	return kg, nil
}

// GetMemManager 获取Agent的MemManager实例（用于复用，避免重复初始化ONNX Runtime）
func (a *Agent) GetMemManager() memManager.Manager {
	return a.memManager
}

func (a *Agent) Close() {
	a.memManager.Close()
}
