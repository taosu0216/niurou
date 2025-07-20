// internal/tools/memory_tool.go
package tools

import (
	"context"
	"fmt"
	"log"
	"niurou/internal/data/memManager" // <-- 只依赖 memManager
	"strings"

	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// MemorySearchTool 封装了与记忆搜索相关的工具。
// 它将一个Go函数的功能，通过eino框架暴露给LLM。
type MemorySearchTool struct {
	memManager memManager.Manager
	// ToolInfo 是符合 eino 框架规范的工具定义，可以被 Agent 注册。
	ToolInfo *schema.ToolInfo
	// GoFunc 保存了要被 Agent 调度的 Go 函数的引用，用于实际执行。
	GoFunc func(ctx context.Context, input *MemorySearchInput) (string, error)
}

// MemorySearchInput 是 search_long_term_memory 工具的输入参数结构体。
// eino 会自动根据这个结构体和它的tags来推断工具的参数Schema。
type MemorySearchInput struct {
	Query string `json:"query" jsonschema:"required,description=需要搜索的自然语言问题或关键词。"`
}

// NewMemorySearchTool 是 MemorySearchTool 的构造函数。
// 它接收一个 memManager 实例，并构建出 LLM 可以调用的工具。
func NewMemorySearchTool(mm memManager.Manager) (*MemorySearchTool, error) {
	toolName := "search_long_term_memory"
	toolDesc := "当需要回答关于用户过去经历、已知事实或历史对话的问题时，调用此工具来搜索用户的长期记忆库。"

	// 1. 创建工具的实例
	t := &MemorySearchTool{
		memManager: mm,
	}
	// 将工具的 Go 函数实现绑定到实例上
	t.GoFunc = t.search

	// 2. 【核心】使用 eino 的 utils.InferTool 自动从 Go 函数推断出 ToolInfo
	// 根据您的 go doc，`InferTool` 是将一个 `InvokeFunc` 转换为 `InvokableTool` 的正确方法。
	// 但我们这里只需要它的 Schema 推断能力，所以我们先用更直接的 GoStruct2ToolInfo。
	// 如果需要完整的 InvokableTool 对象，则应使用 InferTool。
	// 对于仅需要 ToolInfo 的场景，GoStruct2ToolInfo 更直接。
	// 注意：GoFunc2ToolInfo 并不存在，正确的是 GoStruct2ToolInfo 作用于输入结构体，或者 InferTool 作用于整个函数。
	// 我们这里采用另一种更直接的方式：从输入结构体推断参数，然后手动组装 ToolInfo。

	// 从 MemorySearchInput 结构体推断参数的 Schema
	paramsOneOf, err := utils.GoStruct2ParamsOneOf[MemorySearchInput]()
	if err != nil {
		return nil, fmt.Errorf("从MemorySearchInput结构体推断参数失败: %w", err)
	}

	t.ToolInfo = &schema.ToolInfo{
		Name:        toolName,
		Desc:        toolDesc,
		ParamsOneOf: paramsOneOf,
	}

	return t, nil
}

// search 是将被 LLM 调用的实际 Go 函数。
func (t *MemorySearchTool) search(ctx context.Context, input *MemorySearchInput) (string, error) {
	log.Printf("🤖 [Tool Executing] search_long_term_memory, Query: '%s'", input.Query)

	// 调用 memManager 的 HybridSearch，它只返回知识片段
	fragments, err := t.memManager.HybridSearch(ctx, input.Query, 3) // topK=3
	if err != nil {
		log.Printf("❗️ [Tool Error] 记忆搜索失败: %v", err)
		return "记忆搜索时发生内部错误。", err
	}
	if len(fragments) == 0 {
		log.Println("✅ [Tool Result] 在长期记忆中没有找到相关信息。")
		return "在长期记忆中没有找到相关信息。", nil
	}

	// 将找到的知识片段格式化成一个简洁的字符串，供LLM后续处理
	var sb strings.Builder
	sb.WriteString("从长期记忆中找到以下相关信息：\n")
	for _, frag := range fragments {
		sb.WriteString(fmt.Sprintf("- [%s] %s (置信度: %.2f)\n", frag.Source, frag.Content, frag.Certainty))
	}

	resultString := sb.String()
	log.Printf("✅ [Tool Result] 返回了 %d 条知识片段。", len(fragments))
	return resultString, nil
}
