// internal/memManager/sdk.go
package memManager

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"niurou/internal/data/graphDB"
	"niurou/internal/data/vecX" // 只依赖 vecX 和 graphDB
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var MemoryManager Manager

// KnowledgeFragment 是 HybridSearch 的新返回类型，一个结构化的知识片段。
// 它将图谱的直接答案和向量的上下文统一起来。
type KnowledgeFragment struct {
	ID        string  // 记忆的唯一标识符，用于更新和删除操作
	Source    string  // "graph" or "vector"
	Content   string  // 直接答案 或 原始记忆文本
	Certainty float32 // 对于向量搜索，这是Score；对于图谱，可以是固定值
}

// Manager 接口被更新，以反映其新的、更纯粹的职责。
type Manager interface {
	// AddMemory 现在接收由上层（Agent）提取好的知识图谱进行存储。
	AddMemory(ctx context.Context, knowledge *graphDB.KnowledgeGraph, originalText string) (string, error)
	// HybridSearch 现在只返回知识片段，不再负责最终的答案合成。
	HybridSearch(ctx context.Context, queryText string, topK uint64) ([]*KnowledgeFragment, error)
	Close()
	// Update 和 Delete 暂时保留旧签名，未来可以升级
	UpdateMemory(ctx context.Context, id, newMemoryText string) error
	DeleteMemory(ctx context.Context, id string) error
	// ClearAllData 一键清空Neo4j和向量库中的所有数据
	ClearAllData(ctx context.Context) error

	AddPersonNode(ctx context.Context, personNode *graphDB.Person, labels []string) error

	WarmUp(ctx context.Context) (*graphDB.WarmUpResult, error)
}

// managerImpl 不再包含 llmClient。
type managerImpl struct {
	vecService   vecX.Service
	graphService graphDB.Service
}

// InitMemClient 初始化 MemoryManager。
func InitMemClient() (Manager, error) {
	if MemoryManager != nil {
		return MemoryManager, nil
	}

	log.Println("--- Memory Manager (Pure) 初始化开始 ---")
	vecService, err := vecX.New()
	if err != nil {
		return nil, fmt.Errorf("初始化 vecX 服务失败: %w", err)
	}

	graphService, err := graphDB.InitGraphDbService()
	if err != nil {
		vecService.Close()
		return nil, fmt.Errorf("初始化 graphDB 服务失败: %w", err)
	}
	MemoryManager = &managerImpl{
		vecService:   vecService,
		graphService: graphService,
	}
	return MemoryManager, nil
}

func (m *managerImpl) WarmUp(ctx context.Context) (*graphDB.WarmUpResult, error) {
	return m.graphService.WarmUp(ctx)
}

// checkDuplicateBySimilarity 检查是否存在语义相似的记忆
func (m *managerImpl) checkDuplicateBySimilarity(ctx context.Context, text string, threshold float32) (bool, []*KnowledgeFragment, error) {
	// 使用现有的向量搜索功能检查相似度
	results, err := m.HybridSearch(ctx, text, 3) // 搜索最相似的3个
	if err != nil {
		return false, nil, err
	}

	// 检查是否有超过阈值的相似记忆
	for _, result := range results {
		if result.Certainty >= threshold {
			contentPreview := result.Content
			if len(contentPreview) > 50 {
				contentPreview = contentPreview[:50] + "..."
			}
			log.Printf("MemoryManager: 发现高相似度记忆 (相似度: %.3f): %s", result.Certainty, contentPreview)
			return true, results, nil
		}
	}

	return false, results, nil
}

// AddMemory 的签名和实现已改变，它现在接收由 Agent 提取好的知识（带去重检查）。
func (m *managerImpl) AddMemory(ctx context.Context, knowledge *graphDB.KnowledgeGraph, originalText string) (string, error) {
	log.Printf("MemoryManager: 开始存储记忆，先进行去重检查...")

	// 1. 检查语义相似度重复
	isDuplicateSimilar, similarResults, err := m.checkDuplicateBySimilarity(ctx, originalText, 0.95) // 95%相似度阈值
	if err != nil {
		log.Printf("MemoryManager: 相似度重复检查失败，继续存储: %v", err)
	} else if isDuplicateSimilar {
		log.Printf("MemoryManager: 发现高度相似的记忆，跳过存储")
		log.Printf("MemoryManager: 相似记忆数量: %d", len(similarResults))
		// 返回一个特殊的标识表示跳过存储
		return "DUPLICATE_SKIPPED", nil
	}

	// 2. 没有重复，继续正常存储流程
	memoryId := uuid.New().String()
	timestamp := time.Now().Unix()

	// 1. 向量化并存入Qdrant
	vector, err := m.vecService.Encode(originalText)
	if err != nil {
		return "", fmt.Errorf("生成向量失败: %w", err)
	}
	payload := map[string]interface{}{"text": originalText, "timestamp": timestamp}
	if err := m.vecService.AddVector(ctx, memoryId, vector, payload); err != nil {
		return "", fmt.Errorf("存入向量库失败: %w", err)
	}
	log.Printf("MemoryManager: 原始文本已存入 Qdrant, ID: %s", memoryId)

	// 2. 将结构化知识存入Neo4j
	if knowledge != nil && len(knowledge.Nodes) > 0 {
		err := m.graphService.StoreKnowledgeGraph(ctx, memoryId, originalText, timestamp, knowledge)
		if err != nil {
			return "", fmt.Errorf("存入知识图谱失败: %w", err)
		}
		log.Printf("MemoryManager: 结构化知识已存入 Neo4j, 包含 %d 个节点和 %d 条关系。", len(knowledge.Nodes), len(knowledge.Edges))
	} else {
		log.Println("MemoryManager: 未提供结构化知识，跳过图数据库写入。")
	}
	return memoryId, nil
}

// HybridSearch V3: 只负责搜索并返回知识片段，不再合成答案。
func (m *managerImpl) HybridSearch(ctx context.Context, queryText string, topK uint64) ([]*KnowledgeFragment, error) {
	var fragments []*KnowledgeFragment
	var errVector, errGraph error

	done := make(chan bool, 2)

	// Goroutine 1: 向量搜索，召回上下文
	go func() {
		defer func() { done <- true }()
		vec, err := m.vecService.Encode(queryText)
		if err != nil {
			errVector = err
			return
		}
		results, err := m.vecService.SearchSimilarVectors(ctx, vec, topK)
		if err != nil {
			errVector = err
			return
		}
		for _, point := range results {
			// 从Qdrant的point中提取UUID
			pointID := ""
			if point.GetId() != nil {
				if uuid := point.GetId().GetUuid(); uuid != "" {
					pointID = uuid
				}
			}

			fragments = append(fragments, &KnowledgeFragment{
				ID:        pointID,
				Source:    "vector",
				Content:   point.GetPayload()["text"].GetStringValue(),
				Certainty: point.GetScore(),
			})
		}
	}()

	// Goroutine 2: 图谱关键词搜索 (这是一个简化的V3实现)
	go func() {
		defer func() { done <- true }()
		// 注意：这仍然是一个简化的、基于关键词的图谱搜索。
		// 在一个更高级的Agent中，Agent层会自己生成精确的Cypher并调用一个不同的graphDB方法。
		// 但对于工具来说，返回相关的记忆文本也是一种有效的策略。
		keywords := extractEntities(queryText)
		if len(keywords) > 0 {
			results, err := m.graphService.FindMemoriesByEntities(ctx, keywords)
			if err != nil {
				errGraph = err
				return
			}
			for _, text := range results {
				// 为图谱结果生成临时ID（基于内容哈希）
				// TODO: 未来应该从GraphDB直接返回真正的Memory节点ID
				hash := sha256.Sum256([]byte(text))
				tempID := fmt.Sprintf("%x", hash)[:32] // 取前32位作为临时ID

				fragments = append(fragments, &KnowledgeFragment{
					ID:        tempID,
					Source:    "graph",
					Content:   text,
					Certainty: 0.9, // 图谱结果可以给一个较高的置信度
				})
			}
		}
	}()

	<-done
	<-done

	if errVector != nil {
		log.Printf("警告: HybridSearch 中的向量搜索部分失败: %v", errVector)
	}
	if errGraph != nil {
		log.Printf("警告: HybridSearch 中的图谱搜索部分失败: %v", errGraph)
	}

	return fragments, nil
}

func (m *managerImpl) Close() {
	m.vecService.Close()
	m.graphService.Close(context.Background())
}

// --- 以下方法暂时保留旧实现 ---

// extractEntities 是一个遗留的内部帮助函数，仅供简化的HybridSearch使用。
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

// UpdateMemory 负责在两个数据库中同步更新一条记忆
// TODO: 升级此方法以支持对知识图谱的结构化更新
func (m *managerImpl) UpdateMemory(ctx context.Context, id, newMemoryText string) error {
	log.Printf("MemoryManager: 正在更新记忆 (V1) ID: %s", id)

	vector, err := m.vecService.Encode(newMemoryText)
	if err != nil {
		return err
	}

	newTimestamp := time.Now().Unix()
	payload := map[string]interface{}{
		"text":      newMemoryText,
		"timestamp": newTimestamp,
	}
	if err := m.vecService.AddVector(ctx, id, vector, payload); err != nil {
		return err
	}

	newEntities := extractEntities(newMemoryText)
	// 注意：这里调用的还是旧的graphDB接口
	if err := m.graphService.UpdateMemoryNode(ctx, id, newMemoryText, newTimestamp, newEntities); err != nil {
		return err
	}

	log.Printf("MemoryManager: 成功更新记忆 ID: %s", id)
	return nil
}

// DeleteMemory 负责在两个数据库中同步删除一条记忆
func (m *managerImpl) DeleteMemory(ctx context.Context, id string) error {
	log.Printf("MemoryManager: 正在删除记忆 ID: %s", id)

	// 注意：这里调用的还是旧的graphDB接口
	if err := m.graphService.DeleteMemoryNode(ctx, id); err != nil {
		return err
	}

	if err := m.vecService.DeleteVectors(ctx, []string{id}); err != nil {
		return err
	}

	log.Printf("MemoryManager: 成功删除记忆 ID: %s", id)
	return nil
}

// ClearAllData 一键清空Neo4j和向量库中的所有数据
func (m *managerImpl) ClearAllData(ctx context.Context) error {
	log.Println("⚠️ MemoryManager: 开始清空所有数据...")

	var errors []string

	// 1. 清空Neo4j数据库
	log.Println("🗑️ 正在清空Neo4j数据库...")
	if err := m.clearNeo4jData(ctx); err != nil {
		errorMsg := fmt.Sprintf("清空Neo4j失败: %v", err)
		log.Printf("❗️ %s", errorMsg)
		errors = append(errors, errorMsg)
	} else {
		log.Println("✅ Neo4j数据库已清空")
	}

	// 2. 清空Qdrant向量库
	log.Println("🗑️ 正在清空Qdrant向量库...")
	if err := m.clearQdrantData(ctx); err != nil {
		errorMsg := fmt.Sprintf("清空Qdrant失败: %v", err)
		log.Printf("❗️ %s", errorMsg)
		errors = append(errors, errorMsg)
	} else {
		log.Println("✅ Qdrant向量库已清空")
	}

	// 3. 汇总结果
	if len(errors) > 0 {
		return fmt.Errorf("清空数据时发生错误: %s", strings.Join(errors, "; "))
	}

	log.Println("🎉 所有数据已成功清空！")
	return nil
}

// clearNeo4jData 清空Neo4j数据库中的所有节点和关系
func (m *managerImpl) clearNeo4jData(ctx context.Context) error {
	// 使用Neo4j HTTP API执行清空操作
	neo4jURL := "http://localhost:7474/db/neo4j/tx/commit"

	// 构建Cypher查询
	query := map[string]interface{}{
		"statements": []map[string]interface{}{
			{
				"statement": "MATCH (n) DETACH DELETE n",
			},
		},
	}

	jsonData, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("构建Neo4j查询失败: %w", err)
	}

	// 发送HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", neo4jURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建Neo4j请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic bmVvNGo6cGFzc3dvcmQ=") // neo4j:password base64编码

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送Neo4j请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Neo4j返回错误状态码: %d", resp.StatusCode)
	}

	log.Println("✅ Neo4j数据库已通过HTTP API清空")
	return nil
}

// clearQdrantData 清空Qdrant向量库中的所有数据
func (m *managerImpl) clearQdrantData(ctx context.Context) error {
	// 使用Qdrant HTTP API删除并重新创建集合
	qdrantURL := "http://localhost:6333"
	collectionName := "agent_memory"

	// 1. 删除现有集合
	deleteURL := fmt.Sprintf("%s/collections/%s", qdrantURL, collectionName)
	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("创建Qdrant删除请求失败: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送Qdrant删除请求失败: %w", err)
	}
	resp.Body.Close()

	// 删除可能返回404（集合不存在），这是正常的
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		return fmt.Errorf("Qdrant删除集合返回错误状态码: %d", resp.StatusCode)
	}

	// 2. 重新创建集合
	createURL := fmt.Sprintf("%s/collections/%s", qdrantURL, collectionName)
	createBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     768, // 向量维度，匹配实际模型输出
			"distance": "Cosine",
		},
	}

	jsonData, err := json.Marshal(createBody)
	if err != nil {
		return fmt.Errorf("构建Qdrant创建请求失败: %w", err)
	}

	req, err = http.NewRequestWithContext(ctx, "PUT", createURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建Qdrant创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("发送Qdrant创建请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Qdrant创建集合返回错误状态码: %d", resp.StatusCode)
	}

	log.Println("✅ Qdrant向量库已通过HTTP API清空并重新创建")
	return nil
}
