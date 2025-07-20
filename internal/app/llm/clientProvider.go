// internal/llm/provider.go
package llm

import (
	"context"
	"fmt"
	"niurou/internal/configger"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// Provider 是一个LLM服务提供者的接口，只负责提供基础模型。
type Provider interface {
	GetBaseModel() model.ToolCallingChatModel
}

type einoProvider struct {
	baseModel model.ToolCallingChatModel
}

// NewProvider 是 einoProvider 的构造函数。
func NewProvider(ctx context.Context) (Provider, error) {
	config := &openai.ChatModelConfig{
		APIKey:  configger.APIKey,
		Model:   configger.ChatModelName,
		BaseURL: configger.APIBaseURL,
	}
	baseModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("创建 eino openai chat model 失败: %w", err)
	}
	return &einoProvider{baseModel: baseModel}, nil
}

func (p *einoProvider) GetBaseModel() model.ToolCallingChatModel {
	return p.baseModel
}
