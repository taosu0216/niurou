// internal/agent/memoryRecoveryAgent.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"niurou/internal/graphDB"
	"niurou/internal/llm"
	"niurou/internal/memManager"
	"niurou/internal/tools"
)

// MemoryRecoveryAgent 专门负责记忆回收的Agent
// 它分析对话记录，智能判断哪些内容值得保存到长期记忆库中
type MemoryRecoveryAgent struct {
	recoveryClient     *tools.MemoryRecoveryClient
	knowledgeExtractor *tools.KnowledgeExtractorClient
	memManager         memManager.Manager
}

// NewMemoryRecoveryAgent 创建新的记忆回收Agent
func NewMemoryRecoveryAgent(ctx context.Context) (*MemoryRecoveryAgent, error) {
	log.Println("🧠 正在初始化记忆回收Agent...")

	// 1. 初始化LLM Provider
	llmProvider, err := llm.NewProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("初始化LLM Provider失败: %w", err)
	}

	// 2. 创建记忆回收客户端
	recoveryClient, err := tools.NewMemoryRecoveryClient(llmProvider)
	if err != nil {
		return nil, fmt.Errorf("创建记忆回收客户端失败: %w", err)
	}

	// 3. 创建知识提取客户端（用于处理高价值片段）
	knowledgeExtractor, err := tools.NewKnowledgeExtractorClient(llmProvider)
	if err != nil {
		return nil, fmt.Errorf("创建知识提取客户端失败: %w", err)
	}

	// 4. 初始化记忆管理器
	memManager, err := memManager.New()
	if err != nil {
		return nil, fmt.Errorf("初始化记忆管理器失败: %w", err)
	}

	log.Println("✅ 记忆回收Agent初始化成功！")
	return &MemoryRecoveryAgent{
		recoveryClient:     recoveryClient,
		knowledgeExtractor: knowledgeExtractor,
		memManager:         memManager,
	}, nil
}

// ProcessConversationMemory 处理对话记录的记忆回收
// 这是记忆回收Agent的核心方法
func (a *MemoryRecoveryAgent) ProcessConversationMemory(ctx context.Context, conversations []tools.ConversationEntry) (*MemoryRecoveryReport, error) {
	log.Printf("🔍 开始分析 %d 条对话记录...", len(conversations))

	// 1. 使用LLM分析对话价值
	analysisResult, err := a.recoveryClient.AnalyzeConversation(ctx, conversations)
	if err != nil {
		return nil, fmt.Errorf("对话分析失败: %w", err)
	}

	log.Printf("📊 对话分析完成 - 整体价值: %d/10, 发现 %d 个值得保存的片段",
		analysisResult.ConversationAnalysis.OverallValue,
		len(analysisResult.WorthySegments))

	// 2. 判断是否值得保存
	shouldSave := a.recoveryClient.ShouldSaveConversation(analysisResult)
	if !shouldSave {
		log.Println("📝 对话价值较低，跳过保存")
		return &MemoryRecoveryReport{
			AnalysisResult: analysisResult,
			ShouldSave:     false,
			SavedSegments:  0,
			SkippedReason:  "对话整体价值较低，未达到保存阈值",
		}, nil
	}

	// 3. 提取值得保存的片段
	worthySegments := a.recoveryClient.FilterWorthySegments(conversations, analysisResult)
	highValueSegments := a.recoveryClient.GetHighValueSegments(worthySegments)

	log.Printf("💎 发现 %d 个高价值片段（评分≥7）", len(highValueSegments))

	// 4. 处理高价值片段
	savedCount := 0
	var processingErrors []string

	for i, segment := range highValueSegments {
		log.Printf("📚 正在处理第 %d 个高价值片段（评分: %d）...", i+1, segment.ValueScore)

		err := a.processHighValueSegment(ctx, segment)
		if err != nil {
			errorMsg := fmt.Sprintf("处理片段 %d 失败: %v", i+1, err)
			log.Printf("❗️ %s", errorMsg)
			processingErrors = append(processingErrors, errorMsg)
			continue
		}

		savedCount++
		log.Printf("✅ 片段 %d 保存成功", i+1)
	}

	// 5. 生成处理报告
	report := &MemoryRecoveryReport{
		AnalysisResult:    analysisResult,
		ShouldSave:        true,
		SavedSegments:     savedCount,
		TotalSegments:     len(worthySegments),
		HighValueSegments: len(highValueSegments),
		ProcessingErrors:  processingErrors,
	}

	log.Printf("🎯 记忆回收完成 - 成功保存 %d/%d 个高价值片段", savedCount, len(highValueSegments))
	return report, nil
}

// processHighValueSegment 处理单个高价值片段
func (a *MemoryRecoveryAgent) processHighValueSegment(ctx context.Context, segment tools.WorthyConversationSegment) error {
	// 1. 如果片段已经包含提取的知识，直接使用
	var extractedKnowledge *llm.ExtractedKnowledge
	if segment.ExtractedKnowledge != nil {
		extractedKnowledge = segment.ExtractedKnowledge
	} else {
		// 2. 否则，对提取的文本进行知识提取
		log.Println("🔬 正在进行知识提取...")
		knowledge, err := a.knowledgeExtractor.Extract(ctx, segment.ExtractedText)
		if err != nil {
			return fmt.Errorf("知识提取失败: %w", err)
		}
		extractedKnowledge = knowledge
	}

	// 3. 转换为GraphDB格式
	processedGraph, err := a.processExtractedKnowledge(extractedKnowledge)
	if err != nil {
		return fmt.Errorf("知识处理失败: %w", err)
	}

	// 4. 保存到记忆库
	memoryID, err := a.memManager.AddMemory(ctx, processedGraph, segment.ExtractedText)
	if err != nil {
		return fmt.Errorf("保存到记忆库失败: %w", err)
	}

	log.Printf("💾 片段已保存到记忆库，ID: %s", memoryID)
	return nil
}

// processExtractedKnowledge 将LLM提取的知识转换为GraphDB格式
// 这个方法复用了主Agent中的逻辑
func (a *MemoryRecoveryAgent) processExtractedKnowledge(knowledge *llm.ExtractedKnowledge) (*graphDB.KnowledgeGraph, error) {
	if knowledge == nil {
		return &graphDB.KnowledgeGraph{}, nil
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

	return &graphDB.KnowledgeGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// Close 关闭记忆回收Agent
func (a *MemoryRecoveryAgent) Close() {
	log.Println("🔄 正在关闭记忆回收Agent...")
	if a.memManager != nil {
		a.memManager.Close()
	}
	log.Println("✅ 记忆回收Agent已关闭")
}

// MemoryRecoveryReport 记忆回收处理报告
type MemoryRecoveryReport struct {
	AnalysisResult    *llm.MemoryRecoveryResult `json:"analysis_result"`
	ShouldSave        bool                      `json:"should_save"`
	SavedSegments     int                       `json:"saved_segments"`
	TotalSegments     int                       `json:"total_segments"`
	HighValueSegments int                       `json:"high_value_segments"`
	ProcessingErrors  []string                  `json:"processing_errors,omitempty"`
	SkippedReason     string                    `json:"skipped_reason,omitempty"`
}
