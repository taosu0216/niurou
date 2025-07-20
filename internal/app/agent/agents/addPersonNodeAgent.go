package agents

import (
	"context"
	"errors"
	"fmt"
	"log"
	"niurou/internal/app/tools"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var AddPersonNodeAgent *Agent

func GetAddPersonNodeAgent() *Agent {
	if AddPersonNodeAgent == nil {
		log.Println("add person node agent 第一次初始化")
		AddPersonNodeAgent = GetAgent(AddPersonNodeAgentName)
		if AddPersonNodeAgent == nil {
			panic("add person node agent not found")
		}
	}
	return AddPersonNodeAgent
}

type AddPersonNodeInput struct {
	Person struct {
		Name        string   `json:"name"`
		Aliases     []string `json:"aliases,omitempty"`
		Roles       []string `json:"roles,omitempty"`
		Status      string   `json:"status,omitempty"`
		ContactInfo []string `json:"contact_info,omitempty"`
		Notes       string   `json:"notes,omitempty"`
	}
	Labels []string
}

func (a *Agent) AddPersonNodeFn(ctx context.Context, userInput string) (string, error) {

	// 将用户的当前输入添加到对话历史中
	if userInput != "" {
		a.dialogHistory = append(a.dialogHistory, schema.UserMessage(userInput))
	}

	// 1. 调用LLM，让其自主决策
	log.Println("Agent: 正在思考...")
	opts := []model.Option{
		model.WithToolChoice(schema.ToolChoiceForced), // 强制调用工具
	}
	resp, err := a.Model.Generate(ctx, a.dialogHistory, opts...)
	if err != nil {
		log.Printf("❗️ Agent Generate 失败: %v", err)
		return "", fmt.Errorf("agent思考失败: %w", err)
	}
	log.Println("here ")

	// 2. 分析LLM的响应
	if len(resp.ToolCalls) > 0 {
		log.Println("Agent: 决定使用工具。")
		a.dialogHistory = append(a.dialogHistory, resp) // 将LLM的思考过程(工具调用)加入历史

		toolCall := resp.ToolCalls[0] // 简化处理，只执行第一个工具调用
		var toolResult string
		var toolErr error

		// 3. 执行工具
		switch toolCall.Function.Name {
		case tools.AddPersonToolName:
			toolErr = tools.GetAddPersonTool().Execute(ctx, toolCall.Function.Arguments)
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
		return a.AddPersonNodeFn(ctx, "")

	} else {
		// --- LLM决定直接回答 ---
		log.Println("Agent: 决定直接回答。")
		log.Println(resp)
		finalAnswer := resp.Content
		a.dialogHistory = append(a.dialogHistory, resp) // 将最终答案也加入历史
		return finalAnswer, nil
	}
}
