package agents

import (
	"context"
	"niurou/internal/app/llm"
	"niurou/internal/app/tools"
	"niurou/internal/data/memManager"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var once sync.Once

type Agent struct {
	agentName     string
	Model         model.ToolCallingChatModel
	memManager    memManager.Manager
	dialogHistory []*schema.Message

	memorySearcher   *tools.MemorySearchTool
	updateMemoryTool *tools.UpdateMemoryTool
}

func (a *Agent) GetMemManager() memManager.Manager {
	return a.memManager
}

func (a *Agent) Close() {
	a.memManager.Close()
}

const (
	AddPersonNodeAgentName = "add_person_node_agent"
	NiurouAgentName        = "niurou_agent"
)

var AgentPool map[string]*Agent

func InitAgentPool() map[string]*Agent {
	once.Do(func() {
		AgentPool = make(map[string]*Agent)
		ctx := context.Background()
		llmProvider, err := llm.NewProvider(ctx)
		if err != nil {
			panic(err)
		}
		toolBoundModel, err := llmProvider.GetBaseModel().WithTools([]*schema.ToolInfo{tools.GetAddPersonTool().ToolInfo})
		if err != nil {
			panic(err)
		}
		addPersonAgent := &Agent{
			agentName: AddPersonNodeAgentName,
			Model:     toolBoundModel,
			dialogHistory: []*schema.Message{
				schema.SystemMessage(llm.PersonExtractionSystemPrompt),
			},
		}

		AgentPool[AddPersonNodeAgentName] = addPersonAgent
	})
	return AgentPool
}

func GetAgent(agentName string) *Agent {
	return AgentPool[agentName]
}
