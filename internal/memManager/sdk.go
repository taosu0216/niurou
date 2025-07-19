// iinternal/memManager/manager.go

package memManager

import (
	"context"
	"fmt"
	"log"
	"niurou/internal/graphDB"
	"niurou/internal/vecX"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Manager interface {
	AddMemory(ctx context.Context, memoryText string) (string, error)
	UpdateMemory(ctx context.Context, id, newMemoryText string) error // <-- 新增
	DeleteMemory(ctx context.Context, id string) error                // <-- 新增
	HybridSearch(ctx context.Context, queryText string, topK uint64) ([]string, error)
	Close()
}

// managerImpl 实现了管理器
type managerImpl struct {
	vecService   vecX.Service
	graphService graphDB.Service
}

// New 是管理器的构造函数
func New() (Manager, error) {
	log.Println("--- Memory Manager 初始化开始 ---")

	vecService, err := vecX.New()
	if err != nil {
		return nil, fmt.Errorf("初始化 vecX 服务失败: %w", err)
	}

	graphService, err := graphDB.New()
	if err != nil {
		// 如果图数据库初始化失败，需要确保已经启动的 vecX 服务被关闭
		vecService.Close()
		return nil, fmt.Errorf("初始化 graphDB 服务失败: %w", err)
	}

	return &managerImpl{
		vecService:   vecService,
		graphService: graphService,
	}, nil
}

// Close 优雅地关闭所有底层服务
func (m *managerImpl) Close() {
	log.Println("--- 正在关闭 Memory Manager ---")
	m.vecService.Close()
	m.graphService.Close(context.Background())
}

// AddMemory 是高层业务逻辑，它调用底层 SDK 完成组合操作
func (m *managerImpl) AddMemory(ctx context.Context, memoryText string) (string, error) {
	log.Printf("MemoryManager: 正在添加记忆: \"%s\"", memoryText)

	// 1. 生成向量
	vector, err := m.vecService.Encode(memoryText)
	if err != nil {
		return "", err
	}

	// 2. 存入向量库
	memoryId := uuid.New().String()
	timestamp := time.Now().Unix()
	payload := map[string]interface{}{
		"text":      memoryText,
		"timestamp": timestamp,
	}
	if err := m.vecService.AddVector(ctx, memoryId, vector, payload); err != nil {
		return "", err
	}
	log.Printf("MemoryManager: 记忆已存入 Qdrant, ID: %s", memoryId)

	// 3. 提取实体并存入图数据库
	entities := extractEntities(memoryText)
	if len(entities) > 0 {
		if err := m.graphService.AddMemoryNode(ctx, memoryId, memoryText, timestamp, entities); err != nil {
			// 生产环境中可能需要加入补偿/回滚逻辑
			return "", err
		}
		log.Printf("MemoryManager: 记忆已存入 Neo4j, 关联到 %d 个实体", len(entities))
	}

	return memoryId, nil
}

// HybridSearch 实现了您设计的混合搜索流程
func (m *managerImpl) HybridSearch(ctx context.Context, queryText string, topK uint64) ([]string, error) {
	log.Printf("\n--- MemoryManager: 开始混合搜索, 查询: \"%s\" ---", queryText)

	// 1. 向量化查询
	queryVector, err := m.vecService.Encode(queryText)
	if err != nil {
		return nil, err
	}

	// 2. 向量搜索 (语义召回)
	log.Println("--- 2a: 向量搜索 (语义召回) ---")
	vectorSearchResults, err := m.vecService.SearchSimilarVectors(ctx, queryVector, topK)
	if err != nil {
		return nil, err
	}

	var recalledEntities []string
	var uniqueMemoryTexts = make(map[string]bool)

	log.Printf("向量搜索找到 %d 个直接相似的结果:", len(vectorSearchResults))
	for i, point := range vectorSearchResults {
		text := point.GetPayload()["text"].GetStringValue()
		fmt.Printf("  %d. Score: %.4f, Text: %s\n", i+1, point.GetScore(), text)
		uniqueMemoryTexts[text] = true
		recalledEntities = append(recalledEntities, extractEntities(text)...)
	}

	// 3. 图谱搜索 (关联扩展)
	log.Println("--- 2b: 图谱搜索 (关联扩展) ---")
	if len(recalledEntities) > 0 {
		graphSearchResults, err := m.graphService.FindMemoriesByEntities(ctx, recalledEntities)
		if err != nil {
			return nil, err
		}
		log.Printf("通过实体 %v，在图谱中找到 %d 条相关联的记忆:", recalledEntities, len(graphSearchResults))
		for i, text := range graphSearchResults {
			fmt.Printf("  %d. %s\n", i+1, text)
			uniqueMemoryTexts[text] = true
		}
	} else {
		log.Println("未从向量搜索结果中提取到实体，跳过图谱搜索。")
	}

	// 4. 合并并去重所有召回的记忆
	var finalContext []string
	for text := range uniqueMemoryTexts {
		finalContext = append(finalContext, text)
	}

	return finalContext, nil
}

// extractEntities 是一个内部帮助函数
func extractEntities(text string) []string {
	re := regexp.MustCompile(`[\p{Han}A-Za-z0-9]+`)
	matches := re.FindAllString(text, -1)
	stopwords := map[string]bool{"的": true, "是": true, "了": true, "我": true}
	var entities []string
	for _, match := range matches {
		if !stopwords[match] {
			entities = append(entities, strings.ToLower(match))
		}
	}
	return entities
}

// (在 managerImpl 的方法区域)

// UpdateMemory 负责在两个数据库中同步更新一条记忆
func (m *managerImpl) UpdateMemory(ctx context.Context, id, newMemoryText string) error {
	log.Printf("MemoryManager: 正在更新记忆 ID: %s", id)

	// 1. 为新文本生成新向量
	newVector, err := m.vecService.Encode(newMemoryText)
	if err != nil {
		return err
	}

	// 2. 更新 Qdrant (使用 Upsert)
	newTimestamp := time.Now().Unix()
	payload := map[string]interface{}{
		"text":      newMemoryText,
		"timestamp": newTimestamp,
	}
	if err := m.vecService.AddVector(ctx, id, newVector, payload); err != nil {
		return err
	}

	// 3. 更新 Neo4j
	newEntities := extractEntities(newMemoryText)
	if err := m.graphService.UpdateMemoryNode(ctx, id, newMemoryText, newTimestamp, newEntities); err != nil {
		// 在生产环境中，这里需要加入补偿逻辑，将 Qdrant 的修改回滚
		return err
	}

	log.Printf("MemoryManager: 成功更新记忆 ID: %s", id)
	return nil
}

// DeleteMemory 负责在两个数据库中同步删除一条记忆
func (m *managerImpl) DeleteMemory(ctx context.Context, id string) error {
	log.Printf("MemoryManager: 正在删除记忆 ID: %s", id)

	// 我们通常采用“先删主数据，再删向量”的策略
	// 1. 删除 Neo4j 中的图节点
	if err := m.graphService.DeleteMemoryNode(ctx, id); err != nil {
		return err // 如果图删除失败，就不继续删除向量
	}

	// 2. 删除 Qdrant 中的向量
	if err := m.vecService.DeleteVectors(ctx, []string{id}); err != nil {
		// 在生产环境中，这里需要加入补偿逻辑，将 Neo4j 的删除回滚
		return err
	}

	log.Printf("MemoryManager: 成功删除记忆 ID: %s", id)
	return nil
}
