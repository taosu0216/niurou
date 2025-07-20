package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"niurou/internal/app/llm"
	"niurou/internal/app/tools"
	"niurou/internal/configger"
	"niurou/internal/data/memManager"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var NiurouAgent *Agent

func init() {
	ctx := context.Background()
	llmProvider, err := llm.NewProvider(ctx)
	if err != nil {
		panic(err)
	}
	memManager, err := memManager.InitMemClient()
	if err != nil {
		panic(err)
	}
	memorySearcher, err := tools.NewMemorySearchTool(memManager)
	if err != nil {
		panic(err)
	}
	updateMemoryTool, err := tools.NewUpdateMemoryTool(memManager)
	if err != nil {
		panic(err)
	}
	allDialogTools := []*schema.ToolInfo{
		memorySearcher.ToolInfo,
		updateMemoryTool.ToolInfo,
	}
	conversationalModel, err := llmProvider.GetBaseModel().WithTools(allDialogTools)
	if err != nil {
		panic(err)
	}

	warmUpInfo, err := memManager.WarmUp(ctx)
	if err != nil {
		panic(err)
	}

	NiurouAgent = &Agent{
		agentName:     NiurouAgentName,
		Model:         conversationalModel,
		memManager:    memManager,
		dialogHistory: []*schema.Message{schema.SystemMessage(llm.FormatNeuroConversationPrompt(warmUpInfo))},

		memorySearcher:   memorySearcher,
		updateMemoryTool: updateMemoryTool,
	}

	log.Println("NiurouAgent 初始化成功！")
}

func GetNiurouAgent() *Agent {
	return NiurouAgent
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
	resp, err := a.Model.Generate(ctx, a.dialogHistory, opts...)
	if err != nil {
		return "", fmt.Errorf("agent思考失败: %w", err)
	}

	// 2. 分析LLM的响应
	if len(resp.ToolCalls) > 0 {
		toolCall := resp.ToolCalls[0] // 简化处理，只执行第一个工具调用
		var toolResult string
		var toolErr error
		log.Println("Agent: 决定使用工具: ", toolCall.Function.Name)
		a.dialogHistory = append(a.dialogHistory, resp) // 将LLM的思考过程(工具调用)加入历史

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
