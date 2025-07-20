// internal/llm/contract.go
package llm

import "encoding/json"

// ExtractedKnowledge 是我们期望LLM返回的JSON的根结构。
// 它作为LLM和我们系统之间进行数据交换的主要对象(DTO)。
type ExtractedKnowledge struct {
	// Entities 是从文本中识别出的所有唯一实体(节点)的列表。
	Entities []Entity `json:"entities"`

	// Relations 是识别出的、连接这些实体的关系(边)的列表。
	Relations []Relation `json:"relations"`
}

// Entity 代表一个图谱中的节点。
type Entity struct {
	// Name 是实体的唯一规范名称，是我们后续代码中关联实体的“主键”。
	// 示例: "飞龙项目", "李雷"。
	Name string `json:"name"`

	// Labels 是一个字符串数组，代表该节点在Neo4j中的标签，支持多标签。
	// 示例: ["Technology", "ProgrammingLanguage"]。
	Labels []string `json:"labels"`

	// Properties 包含了该实体的所有其他属性。
	// 我们使用 json.RawMessage 来延迟解析此字段。这使我们能够根据实体的
	// 'Labels'，在稍后将其解析为 graphDB 包中定义的具体结构体
	// (例如 graphDB.Person, graphDB.Project)。
	Properties json.RawMessage `json:"properties"`
}

// Relation 代表一个图谱中的边，连接两个实体。
// 它遵循“主语-谓语-宾语” (Subject-Predicate-Object) 的模型。
type Relation struct {
	// Subject 是关系起始节点的名称。
	// 它必须与 Entities 列表中的某个实体的 'Name' 完全匹配。
	Subject string `json:"subject"`

	// Predicate 是关系的类型。
	// 它必须是我们 graphDB/edge.go 中定义的常量之一。
	// 示例: "WORKS_ON", "USES_TECH"。
	Predicate string `json:"predicate"`

	// Object 是关系结束节点的名称。
	// 它必须与 Entities 列表中的某个实体的 'Name' 完全匹配。
	Object string `json:"object"`

	// Properties 包含了该关系的所有属性。
	// 与 Entity.Properties 类似，我们使用 json.RawMessage 以便后续进行灵活的二次解析。
	Properties json.RawMessage `json:"properties"`
}

// MemoryRecoveryResult 是记忆回收Agent返回的JSON结构
// 用于智能判断对话中哪些内容值得保存，并提取结构化知识
type MemoryRecoveryResult struct {
	// ConversationAnalysis 对整个对话的分析结果
	ConversationAnalysis ConversationAnalysis `json:"conversation_analysis"`

	// WorthySegments 值得保存的对话片段列表
	WorthySegments []WorthySegment `json:"worthy_segments"`
}

// ConversationAnalysis 对话整体分析
type ConversationAnalysis struct {
	// OverallValue 整体价值评分 (0-10)
	OverallValue int `json:"overall_value"`

	// HasPersonalInfo 是否包含个人信息
	HasPersonalInfo bool `json:"has_personal_info"`

	// HasTechnicalInfo 是否包含技术信息
	HasTechnicalInfo bool `json:"has_technical_info"`

	// HasProjectInfo 是否包含项目信息
	HasProjectInfo bool `json:"has_project_info"`

	// ConversationThemes 对话主题标签
	ConversationThemes []string `json:"conversation_themes"`

	// Summary 对话摘要
	Summary string `json:"summary"`
}

// WorthySegment 值得保存的对话片段
type WorthySegment struct {
	// SegmentIndex 片段在原对话中的索引范围 [start, end]
	SegmentIndex [2]int `json:"segment_index"`

	// ValueScore 价值评分 (0-10)
	ValueScore int `json:"value_score"`

	// ValueReason 价值判断理由
	ValueReason string `json:"value_reason"`

	// Categories 分类标签
	Categories []string `json:"categories"`

	// ExtractedText 提取的关键文本
	ExtractedText string `json:"extracted_text"`

	// ExtractedKnowledge 从该片段提取的结构化知识
	ExtractedKnowledge *ExtractedKnowledge `json:"extracted_knowledge,omitempty"`
}
