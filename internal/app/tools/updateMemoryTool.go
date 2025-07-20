// internal/tools/updateMemoryTool.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"niurou/internal/data/memManager"
	"strings"

	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// UpdateMemoryRequest 更新记忆的请求参数
type UpdateMemoryRequest struct {
	Query      string `json:"query" jsonschema:"required,description=用于搜索要更新的记忆的查询字符串"`
	Action     string `json:"action" jsonschema:"required,description=要执行的更新动作类型：update|append|correct|delete"`
	NewContent string `json:"new_content" jsonschema:"required,description=新的或修正后的信息内容"`
	Reason     string `json:"reason" jsonschema:"description=更新的原因说明"`
}

// UpdateMemoryResponse 更新记忆的响应结果
type UpdateMemoryResponse struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	UpdatedCount  int      `json:"updated_count"`
	FoundMemories []string `json:"found_memories,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// UpdateMemoryTool 封装了记忆更新相关的工具
type UpdateMemoryTool struct {
	memManager memManager.Manager
	ToolInfo   *schema.ToolInfo
}

// NewUpdateMemoryTool 创建新的记忆更新工具
func NewUpdateMemoryTool(memManager memManager.Manager) (*UpdateMemoryTool, error) {
	tool := &UpdateMemoryTool{
		memManager: memManager,
	}

	// 构建工具信息
	paramsOneOf, err := utils.GoStruct2ParamsOneOf[UpdateMemoryRequest]()
	if err != nil {
		log.Printf("从UpdateMemoryRequest结构体推断参数失败: %v", err)
		return nil, fmt.Errorf("构建工具参数失败: %w", err)
	}

	tool.ToolInfo = &schema.ToolInfo{
		Name:        "update_memory",
		Desc:        "更新或修正已存储的记忆信息。当用户要求修正、补充或删除记忆时使用此工具。",
		ParamsOneOf: paramsOneOf,
	}

	return tool, nil
}

// Execute 执行记忆更新操作
func (t *UpdateMemoryTool) Execute(ctx context.Context, params string) (string, error) {
	log.Printf("UpdateMemoryTool: 开始执行记忆更新，参数: %s", params)

	// 1. 解析参数
	var req UpdateMemoryRequest
	if err := json.Unmarshal([]byte(params), &req); err != nil {
		return t.buildErrorResponse("参数解析失败", err)
	}

	// 2. 验证参数
	if err := t.validateRequest(&req); err != nil {
		return t.buildErrorResponse("参数验证失败", err)
	}

	// 3. 搜索相关记忆
	memories, err := t.searchMemories(ctx, req.Query)
	if err != nil {
		return t.buildErrorResponse("搜索记忆失败", err)
	}

	if len(memories) == 0 {
		response := UpdateMemoryResponse{
			Success: false,
			Message: fmt.Sprintf("未找到与查询 '%s' 相关的记忆", req.Query),
		}
		return t.buildResponse(response)
	}

	// 4. 执行更新操作
	updatedCount, err := t.performUpdate(ctx, memories, &req)
	if err != nil {
		return t.buildErrorResponse("执行更新失败", err)
	}

	// 5. 构建成功响应
	response := UpdateMemoryResponse{
		Success:       true,
		Message:       fmt.Sprintf("成功%s了 %d 条记忆", t.getActionDescription(req.Action), updatedCount),
		UpdatedCount:  updatedCount,
		FoundMemories: t.extractMemoryTexts(memories),
	}

	log.Printf("UpdateMemoryTool: 更新完成，影响 %d 条记忆", updatedCount)
	return t.buildResponse(response)
}

// validateRequest 验证请求参数
func (t *UpdateMemoryTool) validateRequest(req *UpdateMemoryRequest) error {
	if strings.TrimSpace(req.Query) == "" {
		return fmt.Errorf("查询字符串不能为空")
	}

	validActions := map[string]bool{
		"update":  true,
		"append":  true,
		"correct": true,
		"delete":  true,
	}

	if !validActions[req.Action] {
		return fmt.Errorf("无效的动作类型: %s，支持的动作: update, append, correct, delete", req.Action)
	}

	if req.Action != "delete" && strings.TrimSpace(req.NewContent) == "" {
		return fmt.Errorf("除delete动作外，new_content不能为空")
	}

	return nil
}

// searchMemories 搜索相关记忆
func (t *UpdateMemoryTool) searchMemories(ctx context.Context, query string) ([]*memManager.KnowledgeFragment, error) {
	log.Printf("UpdateMemoryTool: 搜索记忆，查询: %s", query)

	// 使用HybridSearch搜索相关记忆
	fragments, err := t.memManager.HybridSearch(ctx, query, 5) // 搜索最相关的5条
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 过滤出相关度较高的记忆（相似度>0.7）
	var relevantMemories []*memManager.KnowledgeFragment
	for _, fragment := range fragments {
		if fragment.Certainty > 0.7 {
			relevantMemories = append(relevantMemories, fragment)
		}
	}

	log.Printf("UpdateMemoryTool: 找到 %d 条相关记忆", len(relevantMemories))
	return relevantMemories, nil
}

// performUpdate 执行具体的更新操作
func (t *UpdateMemoryTool) performUpdate(ctx context.Context, memories []*memManager.KnowledgeFragment, req *UpdateMemoryRequest) (int, error) {
	updatedCount := 0

	for _, memory := range memories {
		var newContent string
		var err error

		switch req.Action {
		case "update", "correct":
			// 完全替换内容
			newContent = req.NewContent
		case "append":
			// 追加内容
			newContent = memory.Content + " " + req.NewContent
		case "delete":
			// 删除记忆
			err = t.memManager.DeleteMemory(ctx, t.extractMemoryID(memory))
			if err == nil {
				updatedCount++
			}
			continue
		}

		if newContent != "" {
			// 更新记忆
			memoryID := t.extractMemoryID(memory)
			err = t.memManager.UpdateMemory(ctx, memoryID, newContent)
			if err == nil {
				updatedCount++
			}
		}

		if err != nil {
			log.Printf("UpdateMemoryTool: 更新记忆失败: %v", err)
			// 继续处理其他记忆，不中断整个流程
		}
	}

	return updatedCount, nil
}

// extractMemoryID 从KnowledgeFragment中提取记忆ID
func (t *UpdateMemoryTool) extractMemoryID(fragment *memManager.KnowledgeFragment) string {
	// 现在KnowledgeFragment包含了真正的ID字段
	if fragment.ID != "" {
		return fragment.ID
	}

	// 如果ID为空，生成一个基于内容的临时ID（向后兼容）
	log.Printf("警告: KnowledgeFragment缺少ID，使用内容哈希生成临时ID")
	hash := fmt.Sprintf("%x", fragment.Content)
	if len(hash) > 32 {
		hash = hash[:32]
	}
	return hash
}

// extractMemoryTexts 提取记忆文本内容
func (t *UpdateMemoryTool) extractMemoryTexts(memories []*memManager.KnowledgeFragment) []string {
	var texts []string
	for _, memory := range memories {
		// 截取前100个字符作为预览
		preview := memory.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		texts = append(texts, preview)
	}
	return texts
}

// getActionDescription 获取动作的中文描述
func (t *UpdateMemoryTool) getActionDescription(action string) string {
	switch action {
	case "update":
		return "更新"
	case "append":
		return "补充"
	case "correct":
		return "修正"
	case "delete":
		return "删除"
	default:
		return "处理"
	}
}

// buildResponse 构建成功响应
func (t *UpdateMemoryTool) buildResponse(response UpdateMemoryResponse) (string, error) {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("构建响应失败: %w", err)
	}
	return string(responseBytes), nil
}

// buildErrorResponse 构建错误响应
func (t *UpdateMemoryTool) buildErrorResponse(message string, err error) (string, error) {
	response := UpdateMemoryResponse{
		Success: false,
		Message: message,
		Error:   err.Error(),
	}
	responseBytes, _ := json.Marshal(response)
	return string(responseBytes), nil
}

// GetToolInfo 返回工具信息，符合eino框架的要求
func (t *UpdateMemoryTool) GetToolInfo() *schema.ToolInfo {
	return t.ToolInfo
}
