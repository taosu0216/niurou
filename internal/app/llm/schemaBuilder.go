// internal/llm/schema_builder.go
package llm

import (
	"niurou/internal/data/graphDB" // 导入我们定义好的schema

	"github.com/getkin/kin-openapi/openapi3"
)

// BuildKnowledgeExtractionSchema 使用代码直接构建用于知识提取的OpenAPI v3 Schema。
// 这种方法提供了对最终Schema的完全控制。
func BuildKnowledgeExtractionSchema() *openapi3.Schema {
	// --- 预加载所有合法的枚举值 ---
	validNodeLabels := toStringAnySlice(GetAllNodeLabels())
	validRelationPredicates := toStringAnySlice(GetAllRelationshipTypes())

	// --- 定义Schema ---
	// 注意：顶层Schema应该是一个对象(Object)，而不是字符串(String)
	schema := &openapi3.Schema{
		Type:        openapi3.TypeObject, // <-- 已修正：顶层应为Object
		Description: "从文本中提取的结构化知识图谱信息。",
		Required:    []string{"entities"}, // 'relations' 可能是可选的，因为有些文本可能只有实体没有关系
		Properties: openapi3.Schemas{
			// --- 定义 'entities' 字段 ---
			"entities": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeArray,
					Description: "从文本中识别出的所有实体（节点）列表。",
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:     openapi3.TypeObject,
							Required: []string{"name", "labels"},
							Properties: openapi3.Schemas{
								"name": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeString,
										Description: "实体的唯一规范名称，例如 'Go' 或 '张三'。",
									},
								},
								"labels": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeArray,
										Description: "描述实体类型的标签数组，必须从以下列表中选择。",
										Items: &openapi3.SchemaRef{
											Value: &openapi3.Schema{
												Type: openapi3.TypeString,
												Enum: validNodeLabels,
											},
										},
									},
								},
								"properties": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeObject,
										Description: "一个包含实体其他属性的对象。其结构应遵循对应实体类型的定义。",
										AdditionalProperties: openapi3.AdditionalProperties{
											Has:    &[]bool{true}[0],
											Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{}},
										},
									},
								},
							},
						},
					},
				},
			},
			// --- 定义 'relations' 字段 ---
			"relations": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeArray,
					Description: "识别出的实体之间的关系（边）列表。",
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:     openapi3.TypeObject,
							Required: []string{"subject", "predicate", "object"},
							Properties: openapi3.Schemas{
								"subject": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeString,
										Description: "关系的主语，必须与实体列表中的某个实体名称完全匹配。",
									},
								},
								"predicate": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeString,
										Description: "关系的类型（谓语），必须从以下列表中选择。",
										Enum:        validRelationPredicates,
									},
								},
								"object": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeString,
										Description: "关系的宾语，必须与实体列表中的某个实体名称完全匹配。",
									},
								},
								"properties": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeObject,
										Description: "一个包含关系其他属性的对象。",
										AdditionalProperties: openapi3.AdditionalProperties{
											Has:    &[]bool{true}[0],
											Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return schema
}

func BuildAddPersonToolSchema() *openapi3.Schema {
	schema := &openapi3.Schema{
		Type:        openapi3.TypeObject,
		Description: "添加一个Person节点。",
		Required:    []string{"person", "labels"},
		Properties: openapi3.Schemas{
			"person": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeObject,
					Description: "Person节点的基本信息",
					Required:    []string{"name", "aliases", "roles", "status", "contact_info", "notes"},
					Properties: openapi3.Schemas{
						"name": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeString,
								Description: "Person节点的名称。也就是对应 person 的法定姓名",
							},
						},
						"aliases": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeArray,
								Description: "Person节点的别名列表。也就是对应 person 的称呼，网名等等",
								Items: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: openapi3.TypeString,
									},
								},
							},
						},
						"roles": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeArray,
								Description: "Person节点的角色列表。roles 是角色的本身职位，比如说学生，打工人，创业者，founder，学者，研究生等等",
								Items: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: openapi3.TypeString,
									},
								},
							},
						},
						"status": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeString,
								Description: "Person节点的状态。比如说 毕业，在打工；居家创业 等等",
							},
						},
						"contact_info": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeArray,
								Description: "Person节点的联系信息列表",
								Items: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: openapi3.TypeString,
									},
								},
							},
						},
						"notes": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeString,
								Description: "Person节点的备注信息",
							},
						},
					},
				},
			},
			"labels": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeArray,
					Description: "Person节点的标签列表。用户没说就默认是 Person, 如果用户说了，就按照用户说的来",
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: openapi3.TypeString,
							Enum: toStringAnySlice(graphDB.AllowedPersonLabels),
						},
					},
				},
			},
		},
	}
	return schema
}

// --- Helper Functions (补齐版本) ---

// GetAllNodeLabels 返回在 graphDB 包中定义的所有节点标签常量。
// 注意：这个实现依赖于您将所有相关常量都放在一个文件中，或者使用统一的命名约定。
// 这是一个简单但有效的实现。
func GetAllNodeLabels() []string {
	return []string{
		// 核心实体
		graphDB.LabelPerson, graphDB.LabelOrganization, graphDB.LabelProject,
		graphDB.LabelProduct, graphDB.LabelTechnology, graphDB.LabelContent,
		graphDB.LabelLocation,
		// 特征与分类
		graphDB.LabelUser, graphDB.LabelCompany, graphDB.LabelUniversity,
		graphDB.LabelTeam, graphDB.LabelStartup, graphDB.LabelProgrammingLanguage,
		graphDB.LabelDatabase, graphDB.LabelFramework, graphDB.LabelSystemDesign,
		graphDB.LabelTechConcept, graphDB.LabelNews, graphDB.LabelNovel,
		graphDB.LabelVideo, graphDB.LabelAI, graphDB.LabelAIConcept,
		// 事件与系统
		graphDB.LabelInteraction, graphDB.LabelMemory,
	}
}

// GetAllRelationshipTypes 返回在 graphDB 包中定义的所有关系类型常量。
func GetAllRelationshipTypes() []string {
	return []string{
		// 核心结构关系
		graphDB.RelHasRelationshipWith, graphDB.RelPartOf, graphDB.RelDevelops,
		// 行为与交互关系
		graphDB.RelUses, graphDB.RelWorksOn, graphDB.RelManages,
		graphDB.RelUsesTech, graphDB.RelInterestedIn, graphDB.RelConsumed,
		// 技术实现关系
		graphDB.RelBuiltWith, graphDB.RelPoweredBy,
		// 事件与来源关系
		graphDB.RelInvolves, graphDB.RelOccurredOn, graphDB.RelDiscussed,
		graphDB.RelCaptures,
		// 通用与备用关系
		graphDB.RelMentions,
	}
}

// BuildMemoryRecoverySchema 构建记忆回收Agent的OpenAPI v3 Schema
func BuildMemoryRecoverySchema() *openapi3.Schema {
	// 定义对话主题和分类的枚举值
	conversationThemes := []any{
		"personal", "technical", "project", "work", "learning", "planning",
		"problem_solving", "decision_making", "brainstorming", "review", "other",
	}

	categories := []any{
		"personal_info", "technical_knowledge", "project_details", "work_progress",
		"learning_notes", "decision_record", "problem_solution", "contact_info",
		"schedule", "preference", "goal", "achievement", "other",
	}

	schema := &openapi3.Schema{
		Type:        openapi3.TypeObject,
		Description: "记忆回收Agent分析对话并提取值得保存的内容",
		Required:    []string{"conversation_analysis", "worthy_segments"},
		Properties: openapi3.Schemas{
			"conversation_analysis": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeObject,
					Description: "对整个对话的分析结果",
					Required:    []string{"overall_value", "has_personal_info", "has_technical_info", "has_project_info", "conversation_themes", "summary"},
					Properties: openapi3.Schemas{
						"overall_value": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeInteger,
								Description: "整体价值评分，0-10分，0表示无价值，10表示极高价值",
								Min:         &[]float64{0}[0],
								Max:         &[]float64{10}[0],
							},
						},
						"has_personal_info": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeBoolean,
								Description: "是否包含个人信息（姓名、联系方式、个人偏好等）",
							},
						},
						"has_technical_info": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeBoolean,
								Description: "是否包含技术信息（编程语言、框架、工具等）",
							},
						},
						"has_project_info": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeBoolean,
								Description: "是否包含项目信息（项目名称、进展、计划等）",
							},
						},
						"conversation_themes": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeArray,
								Description: "对话主题标签",
								Items: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: openapi3.TypeString,
										Enum: conversationThemes,
									},
								},
							},
						},
						"summary": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeString,
								Description: "对话的简要摘要，突出关键信息",
							},
						},
					},
				},
			},
			"worthy_segments": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeArray,
					Description: "值得保存的对话片段列表",
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:     openapi3.TypeObject,
							Required: []string{"segment_index", "value_score", "value_reason", "categories", "extracted_text"},
							Properties: openapi3.Schemas{
								"segment_index": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeArray,
										Description: "片段在原对话中的索引范围 [start, end]",
										Items: &openapi3.SchemaRef{
											Value: &openapi3.Schema{
												Type: openapi3.TypeInteger,
											},
										},
										MinItems: 2,
										MaxItems: &[]uint64{2}[0],
									},
								},
								"value_score": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeInteger,
										Description: "价值评分，0-10分",
										Min:         &[]float64{0}[0],
										Max:         &[]float64{10}[0],
									},
								},
								"value_reason": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeString,
										Description: "价值判断的具体理由",
									},
								},
								"categories": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeArray,
										Description: "分类标签",
										Items: &openapi3.SchemaRef{
											Value: &openapi3.Schema{
												Type: openapi3.TypeString,
												Enum: categories,
											},
										},
									},
								},
								"extracted_text": {
									Value: &openapi3.Schema{
										Type:        openapi3.TypeString,
										Description: "提取的关键文本内容",
									},
								},
								"extracted_knowledge": {
									Value: BuildKnowledgeExtractionSchema(),
								},
							},
						},
					},
				},
			},
		},
	}

	return schema
}

// BuildUpdateMemorySchema 构建用于记忆更新工具的OpenAPI v3 Schema
func BuildUpdateMemorySchema() *openapi3.Schema {
	schema := &openapi3.Schema{
		Type:        openapi3.TypeObject,
		Description: "更新或修正已存储的记忆信息。",
		Required:    []string{"query", "action", "new_content"},
		Properties: openapi3.Schemas{
			// 查询字符串，用于搜索要更新的记忆
			"query": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeString,
					Description: "用于搜索要更新的记忆的查询字符串。应包含关键词或实体名称，如'张三'、'飞龙项目'、'技术栈'等。",
					Example:     "张三 创业伙伴",
				},
			},
			// 更新动作类型
			"action": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeString,
					Description: "要执行的更新动作类型。",
					Enum: []any{
						"update",  // 完全更新/替换信息
						"append",  // 追加新信息
						"correct", // 修正错误信息
						"delete",  // 删除信息
					},
					Example: "correct",
				},
			},
			// 新内容
			"new_content": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeString,
					Description: "新的或修正后的信息内容。对于delete动作，此字段可以为空。",
					Example:     "张三是我的技术顾问，不是创业伙伴",
				},
			},
			// 更新原因
			"reason": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeString,
					Description: "更新的原因说明，帮助理解为什么需要这次更新。",
					Example:     "用户修正了张三的角色信息",
				},
			},
		},
	}
	return schema
}

// toStringAnySlice 将 string 切片转换为 any 切片，以符合 openapi3.Schema 的 Enum 字段类型。
func toStringAnySlice(ss []string) []any {
	a := make([]any, len(ss))
	for i, s := range ss {
		a[i] = s
	}
	return a
}
