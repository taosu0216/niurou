// iinternal/graphDB/sdk.go

package graphDB

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Service interface {
	AddMemoryNode(ctx context.Context, id, text string, timestamp int64, entities []string) error
	UpdateMemoryNode(ctx context.Context, id, newText string, newTimestamp int64, newEntities []string) error // <-- 新增
	DeleteMemoryNode(ctx context.Context, id string) error                                                    // <-- 新增
	FindMemoriesByEntities(ctx context.Context, entityNames []string) ([]string, error)
	Close(ctx context.Context)
}

// serviceImpl 实现了图谱数据库服务
type serviceImpl struct {
	driver neo4j.DriverWithContext
}

const (
	neo4jURI      = "neo4j://localhost:7687"
	neo4jUser     = "neo4j"
	neo4jPassword = "password"
)

// New 是 graphDB SDK 的构造函数
func New() (Service, error) {
	log.Println("--- GraphDB SDK 初始化开始 ---")
	neo4jStart := time.Now()

	driver, err := neo4j.NewDriverWithContext(neo4jURI, neo4j.BasicAuth(neo4jUser, neo4jPassword, ""))
	if err != nil {
		return nil, fmt.Errorf("无法创建 Neo4j 驱动: %w", err)
	}

	// 验证连接
	if err = driver.VerifyConnectivity(context.Background()); err != nil {
		return nil, fmt.Errorf("无法连接到 Neo4j: %w", err)
	}

	log.Printf("Neo4j 客户端初始化成功！耗时: %s", time.Since(neo4jStart))
	return &serviceImpl{driver: driver}, nil
}

// Close 关闭 Neo4j 连接
func (s *serviceImpl) Close(ctx context.Context) {
	if s.driver != nil {
		log.Println("--- 正在关闭 GraphDB SDK 服务 ---")
		s.driver.Close(ctx)
	}
}

// AddMemoryNode 在图谱中创建记忆和实体节点及它们的关系
func (s *serviceImpl) AddMemoryNode(ctx context.Context, id, text string, timestamp int64, entities []string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MERGE (m:Memory {id: $id})
			SET m.text = $text, m.timestamp = $timestamp
			WITH m
			UNWIND $entities AS entityName
			MERGE (e:Entity {name: entityName})
			MERGE (m)-[:CONTAINS_ENTITY]->(e)
		`
		params := map[string]any{
			"id":        id,
			"text":      text,
			"timestamp": timestamp,
			"entities":  entities,
		}
		result, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})

	return err
}

// FindMemoriesByEntities 通过实体在图谱中查找关联的记忆
func (s *serviceImpl) FindMemoriesByEntities(ctx context.Context, entityNames []string) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MATCH (e:Entity)
			WHERE e.name IN $entityNames
			MATCH (m:Memory)-[:CONTAINS_ENTITY]->(e)
			RETURN DISTINCT m.text AS memoryText
		`
		params := map[string]any{"entityNames": entityNames}
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		if err != nil {
			return nil, err
		}

		var memoryTexts []string
		for _, record := range records {
			if text, ok := record.Values[0].(string); ok {
				memoryTexts = append(memoryTexts, text)
			}
		}
		return memoryTexts, nil
	})
	if err != nil {
		return nil, fmt.Errorf("在 Neo4j 中查找失败: %w", err)
	}
	return result.([]string), nil
}

// (在 serviceImpl 的方法区域)

// UpdateMemoryNode 更新一个记忆节点及其关联的实体
func (s *serviceImpl) UpdateMemoryNode(ctx context.Context, id, newText string, newTimestamp int64, newEntities []string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// 这个 Cypher 查询比较复杂，它做了几件事：
		// 1. 找到要更新的 Memory 节点。
		// 2. 更新它的属性。
		// 3. 删除它所有旧的 [:CONTAINS_ENTITY] 关系。
		// 4. 为新的实体列表，创建新的关系。
		// 5. (可选) 删除不再被任何记忆关联的“孤儿”实体。
		cypher := `
            MATCH (m:Memory {id: $id})
            SET m.text = $newText, m.timestamp = $newTimestamp
            WITH m
            OPTIONAL MATCH (m)-[r:CONTAINS_ENTITY]->(oldE:Entity)
            DELETE r
            WITH m
            UNWIND $newEntities AS entityName
            MERGE (e:Entity {name: entityName})
            MERGE (m)-[:CONTAINS_ENTITY]->(e)
        `
		params := map[string]any{
			"id":           id,
			"newText":      newText,
			"newTimestamp": newTimestamp,
			"newEntities":  newEntities,
		}
		result, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})
	return err
}

// (在 serviceImpl 的方法区域)

// DeleteMemoryNode 从图谱中删除一个记忆节点及其关系
func (s *serviceImpl) DeleteMemoryNode(ctx context.Context, id string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// DETACH DELETE 会同时删除节点和它所有的关系
		cypher := `MATCH (m:Memory {id: $id}) DETACH DELETE m`
		params := map[string]any{"id": id}
		result, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})
	return err
}
