package graphDB

import (
	"context"
	"fmt"
	"log"
	"strings"

	"niurou/internal/configger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// KnowledgeGraph 是一个容器，用于封装从LLM提取并经过处理的结构化知识。
type KnowledgeGraph struct {
	Nodes []Node
	Edges []Edge
}

// Node 代表一个图节点。
type Node struct {
	Name       string
	Labels     []string
	Properties map[string]interface{}
}

// Edge 代表一条图的边。
type Edge struct {
	FromNodeName string
	ToNodeName   string
	Type         string
	Properties   map[string]interface{}
}

// Relationship 代表一个具体的关系实例，包含ID和详细信息。
type Relationship struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	FromNodeID string                 `json:"from_node_id"`
	ToNodeID   string                 `json:"to_node_id"`
	Properties map[string]interface{} `json:"properties"`
}

// QueryRequest mirrors the structure from the LLM's query analysis prompt.
type QueryRequest struct {
	StartNode struct {
		Name   string   `json:"name"`
		Labels []string `json:"labels"`
	} `json:"start_node"`
	QueryPattern []struct {
		RelationshipType      string   `json:"relationship_type"`
		RelationshipDirection string   `json:"relationship_direction"`
		TargetNodeLabels      []string `json:"target_node_labels"`
	} `json:"query_pattern"`
	ReturnNode struct {
		Labels   []string `json:"labels"`
		Property string   `json:"property"`
	} `json:"return_node"`
	Filters []struct {
		OnEntity   string `json:"on_entity"`
		EntityName string `json:"entity_name"`
		Property   string `json:"property"`
		Operator   string `json:"operator"`
		Value      string `json:"value"`
	} `json:"filters"`
}

// Service 是与 Neo4j 数据库交互的接口。
type Service interface {
	// StoreKnowledgeGraph 是新的核心方法，用于将结构化知识写入数据库。
	StoreKnowledgeGraph(ctx context.Context, memoryID string, memoryText string, timestamp int64, graph *KnowledgeGraph) error
	Close(ctx context.Context)
	// 保留旧的FindMemoriesByEntities用于HybridSearch的V1版本，未来可以升级
	FindMemoriesByEntities(ctx context.Context, entities []string) ([]string, error)
	DeleteMemoryNode(ctx context.Context, id string) error
	UpdateMemoryNode(ctx context.Context, id, newMemoryText string, newTimestamp int64, newEntities []string) error // 稍后可以升级
	ExecuteStructuredQuery(ctx context.Context, req *QueryRequest) ([]string, error)                                // <-- 新增

}

// serviceImpl 实现了 Service 接口。
type serviceImpl struct {
	driver neo4j.DriverWithContext
}

// New 是 graphDB 服务的构造函数。
func New() (Service, error) {
	// 注意：请将URI、用户名和密码替换为您的Neo4j Aura或本地实例的凭证
	driver, err := neo4j.NewDriverWithContext(configger.GraphDBURI, neo4j.BasicAuth(configger.GraphDBUser, configger.GraphDBPassword, ""))
	if err != nil {
		return nil, fmt.Errorf("创建Neo4j驱动失败: %w", err)
	}

	// 验证连接
	err = driver.VerifyConnectivity(context.Background())
	if err != nil {
		return nil, fmt.Errorf("无法连接到Neo4j: %w", err)
	}
	log.Println("✅ 成功连接到 Neo4j 数据库")
	return &serviceImpl{driver: driver}, nil
}

func (s *serviceImpl) Close(ctx context.Context) {
	s.driver.Close(ctx)
}

// StoreKnowledgeGraph 实现了将结构化知识写入Neo4j的核心逻辑。
func (s *serviceImpl) StoreKnowledgeGraph(ctx context.Context, memoryID string, memoryText string, timestamp int64, graph *KnowledgeGraph) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 1. 创建或合并 Memory 节点
		_, err := tx.Run(ctx,
			"MERGE (m:Memory {id: $id}) SET m.text = $text, m.timestamp = $timestamp",
			map[string]interface{}{
				"id":        memoryID,
				"text":      memoryText,
				"timestamp": timestamp,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("创建Memory节点失败: %w", err)
		}

		// 2. 批量创建或合并所有实体节点
		// 我们使用 UNWIND 来高效处理批量数据
		cypherNodes := []map[string]interface{}{}
		for _, node := range graph.Nodes {
			cypherNodes = append(cypherNodes, map[string]interface{}{
				"name":       node.Name,
				"labels":     node.Labels,
				"properties": node.Properties,
			})
		}

		// APOC 库对于动态设置标签非常有用。确保您的Neo4j实例安装了APOC。
		// MERGE (n {name: node.name}) CALL apoc.create.addLabels(n, node.labels) YIELD node as n2 SET n2 += node.properties
		query := `
		UNWIND $nodes AS node_data
		MERGE (n {name: node_data.name})
		WITH n, node_data // <-- 在这里插入 WITH 子句
		CALL apoc.create.addLabels(id(n), node_data.labels) YIELD node AS ignored
		SET n += node_data.properties
		WITH n
		MATCH (m:Memory {id: $memoryId})
		MERGE (m)-[:MENTIONS]->(n)
		`
		_, err = tx.Run(ctx, query, map[string]interface{}{"nodes": cypherNodes, "memoryId": memoryID})
		if err != nil {
			return nil, fmt.Errorf("批量合并实体节点失败: %w", err)
		}

		// 3. 批量创建关系
		cypherEdges := []map[string]interface{}{}
		for _, edge := range graph.Edges {
			cypherEdges = append(cypherEdges, map[string]interface{}{
				"from":       edge.FromNodeName,
				"to":         edge.ToNodeName,
				"type":       edge.Type,
				"properties": edge.Properties,
			})
		}

		if len(cypherEdges) > 0 {
			query = `
			UNWIND $edges AS edge_data
			MATCH (from {name: edge_data.from})
			MATCH (to {name: edge_data.to})
			CALL apoc.create.relationship(from, edge_data.type, edge_data.properties, to) YIELD rel
			RETURN count(rel)
			`
			_, err = tx.Run(ctx, query, map[string]interface{}{"edges": cypherEdges})
			if err != nil {
				return nil, fmt.Errorf("批量创建关系失败: %w", err)
			}
		}

		return nil, nil
	})

	return err
}

// FindMemoriesByEntities 通过实体在图谱中查找关联的记忆
func (s *serviceImpl) FindMemoriesByEntities(ctx context.Context, entityNames []string) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MATCH (e)
			WHERE e.name IN $entityNames
			MATCH (m:Memory)-[:MENTIONS]->(e)
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

// ExecuteStructuredQuery 将结构化的请求翻译成Cypher并执行
func (s *serviceImpl) ExecuteStructuredQuery(ctx context.Context, req *QueryRequest) ([]string, error) {
	if req.StartNode.Name == "" {
		return nil, fmt.Errorf("查询请求缺少起点")
	}

	// --- 动态构建Cypher查询 ---
	var sb strings.Builder
	params := make(map[string]interface{})

	// 1. 构建 MATCH 子句
	sb.WriteString(fmt.Sprintf("MATCH (a:%s {name: $start_node_name})", strings.Join(req.StartNode.Labels, ":")))
	params["start_node_name"] = req.StartNode.Name

	// 2. 构建查询模式
	lastNodeVar := "a"
	for i, pattern := range req.QueryPattern {
		currentNodeVar := fmt.Sprintf("n%d", i)
		relVar := fmt.Sprintf("r%d", i+1)

		var directionArrowLeft, directionArrowRight string
		switch pattern.RelationshipDirection {
		case "in":
			directionArrowLeft = "<-"
			directionArrowRight = "-"
		case "out":
			directionArrowLeft = "-"
			directionArrowRight = "->"
		default:
			directionArrowLeft = "-"
			directionArrowRight = "-"
		}

		sb.WriteString(fmt.Sprintf("%s[%s:%s]%s(%s:%s)",
			directionArrowLeft, relVar, pattern.RelationshipType, directionArrowRight,
			currentNodeVar, strings.Join(pattern.TargetNodeLabels, ":")))
		lastNodeVar = currentNodeVar
	}

	// 3. 构建 WHERE 子句
	if len(req.Filters) > 0 {
		sb.WriteString(" WHERE ")
		for i, filter := range req.Filters {
			if i > 0 {
				sb.WriteString(" AND ")
			}
			paramName := fmt.Sprintf("filter_value_%d", i)
			sb.WriteString(fmt.Sprintf("%s.%s = $%s", filter.EntityName, filter.Property, paramName))
			params[paramName] = filter.Value
		}
	}

	// 4. 构建 RETURN 子句
	returnProperty := req.ReturnNode.Property
	if returnProperty == "node" {
		returnProperty = "" // 返回整个节点
	} else {
		returnProperty = "." + returnProperty
	}
	sb.WriteString(fmt.Sprintf(" RETURN %s%s", lastNodeVar, returnProperty))

	generatedCypher := sb.String()
	log.Printf("动态生成的Cypher查询: %s", generatedCypher)
	log.Printf("查询参数: %v", params)

	// --- 执行查询 ---
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	results, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		res, err := tx.Run(ctx, generatedCypher, params)
		if err != nil {
			return nil, err
		}

		var records []string
		for res.Next(ctx) {
			record := res.Record()
			if record.Values[0] != nil {
				records = append(records, fmt.Sprintf("%v", record.Values[0]))
			}
		}
		return records, res.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("执行动态Cypher查询失败: %w", err)
	}

	return results.([]string), nil
}
