// internal/graphDB/edge.go

// Package graphDB anages all interactions with the Neo4j graph database.
// This file defines the schema for all RELATIONSHIPS (edges) in the knowledge graph.
package graphDB

// --- 关系类型 (Relationship Types) ---
// 该常量块定义了连接图中“节点”的“边”的所有类型。
// 清晰的分类和一致的命名有助于理解图谱中实体间的动态联系。
const (
	// === 1. 核心结构关系 (Core Structural Relationships) ===
	// 这些关系定义了现实世界中实体间最基础、最核心的结构性联系。

	// --- 人与人的关系 ---
	RelHasRelationshipWith = "HAS_RELATIONSHIP_WITH" // e.g., (:Person)-[r:HAS_RELATIONSHIP_WITH {type:"朋友"}]->(:Person)

	// --- 人与组织的关系 ---
	RelPartOf = "PART_OF" // e.g., (:Person)-[r:PART_OF {role:"开发者"}]->(:Organization)

	// --- 组织与产品的关系 ---
	RelDevelops = "DEVELOPS" // e.g., (:Organization)-[r:DEVELOPS {type:"所有者"}]->(:Product)

	// === 2. 行为与交互关系 (Behavioral & Interactional Relationships) ===
	// 这些关系描述了实体（主要是人）与其他实体发生的动态行为。

	// --- 人与产品的交互 ---
	RelUses = "USES" // e.g., (:Person)-[r:USES {frequency:"每天"}]->(:Product)

	// --- 人与项目的交互 ---
	RelWorksOn = "WORKS_ON" // e.g., (:Person)-[r:WORKS_ON {roles:["后端"]}]->(:Project)
	RelManages = "MANAGES"  // e.g., (:Person)-[:MANAGES]->(:Project)

	// --- 人与技术的交互 ---
	RelUsesTech = "USES_TECH" // e.g., (:Person)-[r:USES_TECH {proficiency:"熟练"}]->(:Technology)

	// --- 人与内容的交互 ---
	RelInterestedIn = "INTERESTED_IN" // e.g., (:Person)-[:INTERESTED_IN]->(:Content)
	RelConsumed     = "CONSUMED"      // e.g., (:Person)-[r:CONSUMED {status:"已看完"}]->(:Content)

	// === 3. 技术与产品/项目的实现关系 (Implementation Relationships) ===
	// 这些关系描述了“事物”是如何被构建出来的。

	RelBuiltWith = "BUILT_WITH" // e.g., (:Project)-[r:BUILT_WITH {is_core:true}]->(:Technology)
	RelPoweredBy = "POWERED_BY" // e.g., (:Product)-[r:POWERED_BY]->(:Technology)

	// === 4. 事件与来源关系 (Event & Provenance Relationships) ===
	// 这些关系用于构建事件的上下文，并追溯知识的来源。

	RelInvolves   = "INVOLVES"    // 描述一次交互事件涉及了哪些人。 e.g., (:Interaction)-[r:INVOLVES {role:"发起者"}]->(:Person)
	RelOccurredOn = "OCCURRED_ON" // 描述一次交互事件发生在哪个产品/平台上。 e.g., (:Interaction)-[:OCCURRED_ON]->(:Product {name:"Lark"})
	RelDiscussed  = "DISCUSSED"   // 描述一次交互事件讨论了什么。 e.g., (:Interaction)-[r:DISCUSSED {sentiment:"积极"}]->(:Project)
	RelCaptures   = "CAPTURES"    // 描述一条原始记忆捕获了哪次交互事件。 e.g., (:Memory)-[:CAPTURES]->(:Interaction)

	// === 5. 通用与备用关系 (Generic & Fallback Relationships) ===

	// RelMentions 是一个通用的“提及”关系，作为无法确定更具体关系时的备用方案。
	RelMentions = "MENTIONS" // e.g., (:Memory)-[:MENTIONS]->(:Person)

	// RelAdoptsTech 在之前的讨论中是“组织采纳技术”，这里我建议将其并入更通用的 :USES_TECH 或通过事件来表达，
	// 以保持关系类型的简洁。如果需要，我们可以随时将其加回。
	// (暂时注释掉以简化模型)
	// RelAdoptsTech = "ADOPTS_TECH"
)

// --- 关系属性结构体 (Relationship Property Structs) ---
//
// 下方的这些结构体，为我们图谱中一些重要的“关系(边)”定义了其可以携带的属性的数据模型。
// 这为代码提供了类型安全，并让整个Schema定义更加清晰和完整。
// 可以将它们理解为特定关系的“属性清单”或“合同模板”。

// HasRelationshipWithProperties 定义了 :HAS_RELATIONSHIP_WITH (拥有关系) 这条边的属性。
// 它用来描述人与人之间社交关系的具体细节。
type HasRelationshipWithProperties struct {
	// Type 描述了关系的性质。
	// 例如: "朋友", "女友", "创业伙伴", "同事"。
	Type string `json:"type,omitempty"`

	// Since 记录了这段关系是何时开始的。
	// 使用字符串类型以增加灵活性，可以存储一个完整的日期，也可以只存一个年份。
	// 例如: "2022-10-01"。
	Since string `json:"since,omitempty"`
}

// PartOfProperties 定义了 :PART_OF (是...的一部分) 这条边的属性。
// 它用来描述一个人加入一个组织的具体情况。
type PartOfProperties struct {
	// Roles 列出了这个人在这个组织中所担任的具体角色/职位。
	// 这比Person节点上总的Roles更具体。例如，一个人总体上是个“开发者”，
	// 但在这个团队里，他的具体角色可能是["前端组长"]。
	Roles []string `json:"roles,omitempty"`

	// StartDate 记录了这个人加入该组织的日期。
	StartDate string `json:"start_date,omitempty"`

	// EndDate 记录了这个人离开该组织的日期 (如果适用)。
	EndDate string `json:"end_date,omitempty"`
}

// UsesProperties 定义了 :USES (使用) 这条边的属性。
// 它用来描述一个用户（Person）使用一个产品（Product）的情况。
type UsesProperties struct {
	// LastUsedDate 记录了Agent最后一次注意到该产品被使用的日期。
	LastUsedDate string `json:"last_used_date,omitempty"`

	// UsageFrequency 提供了对使用频率的一个定性描述。
	// 例如: "每天", "每周", "偶尔"。
	UsageFrequency string `json:"usage_frequency,omitempty"`

	// SubscriptionStatus 描述了用户对于该产品的订阅状态或会员等级。
	// 例如: "免费版", "专业版用户", "企业版"。
	SubscriptionStatus string `json:"subscription_status,omitempty"`
}

// DevelopsProperties 定义了 :DEVELOPS (开发) 这条边的属性。
// 它用来描述一个组织和一个产品之间开发关系的具体性质。
type DevelopsProperties struct {
	// Type 描述了开发关系的类型。
	// 这对于区分合作开发、开源贡献等复杂场景至关重要。
	// 例如: "所有者 (Owner)", "主要贡献者 (Contributor)", "发布者 (Publisher)"。
	Type string `json:"type,omitempty"`

	// IsActive 表明这个开发关系当前是否处于活跃状态。
	// 例如，一个产品可能已经进入仅维护阶段，其开发关系就可以标记为不活跃。
	IsActive bool `json:"is_active,omitempty"`

	// StartDate 记录了开发关系的起始时间。
	StartDate string `json:"start_date,omitempty"`
}

// WorksOnProperties 定义了 :WORKS_ON 这条边的属性。
// 它用来描述一个人在一个特定项目中的具体贡献和角色。
type WorksOnProperties struct {
	// Roles 列出了这个人在这个特定项目中所扮演的角色。
	// 这比 Person 节点上的通用 Roles 更加精确。
	// 例如: 一个人可能是个全能开发者，但在A项目中他只负责["后端开发", "数据库设计"]。
	Roles []string `json:"roles,omitempty"`

	// ContributionHours 可以用来记录一个大致的工时或贡献度。
	ContributionHours int `json:"contribution_hours,omitempty"`
}

// Technology 代表任何带有 :Technology 或其子标签的节点的属性数据模型。
type Technology struct {
	// Name 是技术/概念的名称，作为其唯一的标识符。
	// 例如: "Go", "Neo4j", "微服务架构"。
	Name string `json:"name"`

	// Type 可以用来做一个简单的、非层级的类型说明。
	// 虽然我们有子标签，但有时一个额外的类型字段可以提供更多灵活性。
	// 例如，对于一个数据库，Type可以是 "Graph DB", "Relational DB"。
	Type string `json:"type,omitempty"`

	// Version 记录了技术的具体版本号。
	// 这对于描述项目依赖和兼容性非常重要。
	// 例如: "1.21", "5.0"。
	Version string `json:"version,omitempty"`

	// Description 对该技术或概念进行简要描述。
	// 例如: 对于"RAG"，描述可以是"检索增强生成模型"。
	Description string `json:"description,omitempty"`

	// URL 指向该技术的官方文档、教程或代码仓库。
	// 这是一个 string 数组，以容纳多个相关链接。
	URL []string `json:"url,omitempty"`
}

// UsesTechProperties 定义了 :USES_TECH (个人使用技术) 这条边的属性。
type UsesTechProperties struct {
	Proficiency       string `json:"proficiency,omitempty"` // e.g., "熟练", "精通", "初学者"
	YearsOfExperience int    `json:"years_of_experience,omitempty"`
}

// BuiltWithProperties 定义了 :BUILT_WITH (项目使用技术) 这条边的属性。
type BuiltWithProperties struct {
	// VersionLock 记录了项目在构建时锁定的具体技术版本。
	// e.g., "1.21.0", "~5.7"
	VersionLock string `json:"version_lock,omitempty"`
	// IsCoreComponent 标记该技术是否为项目的核心组件。
	IsCoreComponent bool `json:"is_core_component,omitempty"`
}

// PoweredByProperties 定义了 :POWERED_BY (产品由技术驱动) 这条边的属性。
type PoweredByProperties struct {
	// ImplementationDetails 提供一些关于如何使用的简短说明。
	// e.g., "用于后端服务", "作为主数据库"
	ImplementationDetails string `json:"implementation_details,omitempty"`
}

// AdoptsTechProperties 定义了 :ADOPTS_TECH (组织采纳技术) 这条边的属性。
type AdoptsTechProperties struct {
	// AdoptionLevel 描述了在组织内的推广程度。
	// e.g., "官方标准", "推荐使用", "个别团队使用"
	AdoptionLevel string `json:"adoption_level,omitempty"`
	// Since 记录了组织从何时开始采纳该技术。
	Since string `json:"since,omitempty"`
}

// ConsumedProperties 定义了 :CONSUMED 这条边的属性。
// 它用来描述一次内容消费事件的具体细节。
type ConsumedProperties struct {
	// Status 描述了消费的状态。
	// 例如: "正在看", "已看完", "看到一半", "已收藏待看"。
	Status string `json:"status,omitempty"`

	// OnDate 记录了消费行为发生的大致日期。
	OnDate string `json:"on_date,omitempty"`

	// Rating 可以用来记录您对内容的个人评分（1-5星）或简单评价。
	// 例如: 5, "推荐"。
	Rating int `json:"rating,omitempty"`

	// Notes 记录了您在消费内容时产生的任何想法、笔记或评论。
	// 例如: "这本书关于黑暗森林法则的探讨非常深刻"。
	Notes string `json:"notes,omitempty"`
}

// InvolvesProperties 定义了 :INVOLVES 这条边的属性。
// 它用来描述一个参与者在交互中的具体角色或行为。
type InvolvesProperties struct {
	// Role 描述了参与者在此次交互中的角色。
	// 例如: "发起者", "提问者", "回答者"。
	Role string `json:"role,omitempty"`
}

// DiscussedProperties 定义了 :DISCUSSED 这条边的属性。
type DiscussedProperties struct {
	// Sentiment 描述了在讨论该主题时的情感色彩。
	// 可以由LLM进行分析得出。
	// 例如: "积极", "中立", "存在分歧"。
	Sentiment string `json:"sentiment,omitempty"`
}
