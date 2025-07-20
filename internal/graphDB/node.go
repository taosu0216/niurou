// internal/graphDB/node.go

// Package graphDB anages all interactions with the Neo4j graph database.
// This file defines the schema for all NODES in the knowledge graph.
package graphDB

// --- 节点标签 (Node Labels) ---
// 该常量块定义了图中所有可能的实体(节点)类型。
// 清晰的分类和一致的命名是构建可维护知识图谱的基础。
const (
	// === 1. 核心实体 (Core Entities) ===
	// 这些是图谱中最核心的、代表现实世界主要事物的“名词”。

	LabelPerson       = "Person"       // 代表一个人类。图谱的中心。
	LabelOrganization = "Organization" // 代表一个组织。作为公司、团队等的父标签。
	LabelProject      = "Project"      // 代表一个有明确目标的事业或项目。
	LabelProduct      = "Product"      // 代表一个具体的产品，无论是软件还是硬件。
	LabelTechnology   = "Technology"   // 代表一项技术。作为编程语言、数据库等的父标签。
	LabelContent      = "Content"      // 代表一份信息或媒体内容。作为新闻、小说等的父标签。
	LabelLocation     = "Location"     // 代表一个地理位置。

	// === 2. 特征与分类标签 (Feature & Classification Labels) ===
	// 这些标签通常与其他核心标签组合使用(多标签)，以提供更精细的分类或描述实体的特性。
	// 它们更像是“形容词”。

	// --- Person 的特殊分类 ---
	LabelUser = "User" // 特指您自己，用于快速查询。用法: (:Person:User)

	// --- Organization 的子分类 ---
	LabelCompany    = "Company"    // 商业公司。用法: (:Organization:Company)
	LabelUniversity = "University" // 大学等教育机构。用法: (:Organization:University)
	LabelTeam       = "Team"       // 通用团队。用法: (:Organization:Team)
	LabelStartup    = "Startup"    // 创业公司/团队，是Team/Company的一种特殊形式。用法: (:Organization:Startup)

	// --- Technology 的子分类 ---
	LabelProgrammingLanguage = "ProgrammingLanguage" // 编程语言。用法: (:Technology:ProgrammingLanguage)
	LabelDatabase            = "Database"            // 数据库。用法: (:Technology:Database)
	LabelFramework           = "Framework"           // 软件框架。用法: (:Technology:Framework)
	LabelSystemDesign        = "SystemDesign"        // 系统设计思想。用法: (:Technology:SystemDesign)
	LabelTechConcept         = "TechConcept"         // 通用技术概念。用法: (:Technology:TechConcept)

	// --- Content 的子分类 ---
	LabelNews  = "News"  // 新闻。用法: (:Content:News)
	LabelNovel = "Novel" // 小说。用法: (:Content:Novel)
	LabelVideo = "Video" // 视频。用法: (:Content:Video)

	// --- AI 特征标签 (可应用于多种核心实体) ---
	LabelAI        = "AI"        // 表明一个实体具备AI特性。用法: (:Product:AI), (:Technology:AI)
	LabelAIConcept = "AIConcept" // 表明一个技术概念属于AI领域。用法: (:Technology:AIConcept)

	// === 3. 事件与系统标签 (Event & System Labels) ===
	// 这些标签代表系统内部的、用于记录事件和数据来源的实体。

	// LabelInteraction 代表一次具体的、有时间戳的交互事件，是连接不同实体的“事件枢纽”。
	LabelInteraction = "Interaction"

	// LabelMemory 代表进入系统的最原始、未经处理的文本记录，是所有知识的“证据”和来源。
	LabelMemory = "Memory"
)

// --- 节点属性结构体 (Node Property Structs) ---

type Person struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Status      string   `json:"status,omitempty"`
	ContactInfo string   `json:"contact_info,omitempty"`
	Notes       string   `json:"notes,omitempty"`
}

type Organization struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Industry     string `json:"industry,omitempty"`
	Website      string `json:"website,omitempty"`
	LocationName string `json:"location_name,omitempty"`
}

type Product struct {
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	LaunchDate  string   `json:"launch_date,omitempty"`
	Description string   `json:"description,omitempty"`
	URL         []string `json:"url,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Project 代表一个 :Project 节点的属性数据模型。
type Project struct {
	// Name 是项目的正式名称，作为其主要的唯一标识符。
	// 例如: "Agent记忆系统开发"。
	Name string `json:"name"`

	// Description 对项目的目标、范围或愿景进行简要的文字描述。
	// 例如: "构建一个结合向量与图数据库的长期记忆AI Agent"。
	Description string `json:"description,omitempty"`

	// Status 描述了项目当前的进展状态。
	// 这是一个非常重要的字段，可以帮助Agent了解任务的优先级和进展。
	// 例如: "规划中", "开发中", "测试阶段", "已完成", "已搁置"。
	Status string `json:"status,omitempty"`

	// StartDate 记录了项目的启动日期。
	// 使用字符串以增加灵活性，例如 "2024-Q3" 或 "2024-07-01"。
	StartDate string `json:"start_date,omitempty"`

	// EndDate 记录了项目的计划完成日期或实际完成日期。
	EndDate string `json:"end_date,omitempty"`

	// URL 可以链接到项目的代码仓库(如GitHub)、项目管理工具(如Jira)或相关的文档页面。
	URL []string `json:"url,omitempty"`

	Scale string `json:"scale,omitempty"`
}

// Content 代表任何带有 :Content 或其子标签的节点的属性数据模型。
type Content struct {
	// Name 是内容作品的标题，作为其主要的唯一标识符。
	// 例如: "《三体》", "一篇关于AI Agent的深度报道"。
	Name string `json:"name"`

	// Author 或 Creator，指内容的作者、创作者或发布者。
	// 例如: "刘慈欣", "InfoQ"。
	Author string `json:"author,omitempty"`

	// URL 指向该内容的网络链接。
	// 使用数组以容纳多个相关链接（例如原文链接、转载链接、讨论区链接）。
	URL []string `json:"url,omitempty"`

	// Genre 或 Category，用于对内容进行分类。
	// 例如: "科幻", "技术新闻", "教程"。
	Genre string `json:"genre,omitempty"`

	// Summary 是对内容的一个简短摘要或简介。
	Summary string `json:"summary,omitempty"`
}

// Memory 代表 :Memory 节点的属性数据模型。
type Memory struct {
	// ID 是记忆的唯一标识符。
	ID string `json:"id"`

	// Text 是原始的记忆文本内容。
	Text string `json:"text"`

	// Timestamp 记录了记忆创建的时间戳。
	Timestamp int64 `json:"timestamp"`

	// Source 记录了记忆的来源。
	// 例如: "conversation", "document", "manual_input"。
	Source string `json:"source,omitempty"`

	// Tags 用于对记忆进行分类标记。
	Tags []string `json:"tags,omitempty"`
}

// Interaction 结构体保持不变 (SourceID被保留)
type Interaction struct {
	Timestamp int64  `json:"timestamp"`
	Summary   string `json:"summary,omitempty"`
	SourceID  string `json:"source_id,omitempty"` // For idempotency
}
