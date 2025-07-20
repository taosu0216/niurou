package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"niurou/internal/app/llm"
	"niurou/internal/data/graphDB" // 导入我们定义好的schema

	"github.com/cloudwego/eino/schema"
)

const (
	AddPersonToolName = "add_person"
)

type AddPersonTool struct {
	ToolInfo *schema.ToolInfo
	GraphDB  graphDB.Service
}

func GetAddPersonTool() *AddPersonTool {
	toolName := AddPersonToolName
	toolSchema := llm.BuildAddPersonToolSchema()
	paramsOneOf := schema.NewParamsOneOfByOpenAPIV3(toolSchema)
	toolInfo := &schema.ToolInfo{
		Name:        toolName,
		Desc:        "添加一个Person节点。",
		ParamsOneOf: paramsOneOf,
	}

	graphDBService, err := graphDB.InitGraphDbService()
	if err != nil {
		panic(err)
	}

	return &AddPersonTool{
		ToolInfo: toolInfo,
		GraphDB:  graphDBService,
	}
}

func (t *AddPersonTool) Execute(ctx context.Context, input string) error {
	var args struct {
		Person graphDB.Person
		Labels []string
	}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return fmt.Errorf("解析记忆搜索工具参数失败: %w", err)
	}
	return t.GraphDB.AddPersonNode(ctx, &args.Person, args.Labels)
}
