package graphDB

import (
	"context"
	"fmt"
	"log"
	"niurou/internal/configger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// WarmUpResult 包含用户的完整上下文信息
type WarmUpResult struct {
	UserInfo      *Person                    `json:"user_info"`
	Projects      []ProjectWithRelation      `json:"projects"`
	Organizations []OrganizationWithRelation `json:"organizations"`
	People        []PersonWithRelation       `json:"people"`
	Products      []ProductWithRelation      `json:"products"`
	Technologies  []TechnologyWithRelation   `json:"technologies"`
}

// 带关系信息的结构体
type ProjectWithRelation struct {
	Project      *Project               `json:"project"`
	Relationship string                 `json:"relationship"` // WORKS_ON, MANAGES等
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

type OrganizationWithRelation struct {
	Organization *Organization          `json:"organization"`
	Relationship string                 `json:"relationship"` // PART_OF等
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

type PersonWithRelation struct {
	Person       *Person                `json:"person"`
	Relationship string                 `json:"relationship"` // HAS_RELATIONSHIP_WITH等
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

type ProductWithRelation struct {
	Product      *Product               `json:"product"`
	Relationship string                 `json:"relationship"` // USES, DEVELOPS等
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

type TechnologyWithRelation struct {
	Technology   *Technology            `json:"technology"`
	Relationship string                 `json:"relationship"` // USES_TECH等
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

func (s *serviceImpl) WarmUp(ctx context.Context) (*WarmUpResult, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// 1. 根据别名查找真名
	userRealName, err := s.GetPersonNameByAlias(ctx, configger.UserName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user real name: %w", err)
	}

	if userRealName == "" {
		return nil, fmt.Errorf("user not found with alias: %s", configger.UserName)
	}

	// 2. 获取用户的完整上下文信息
	warmUpResult, err := s.getUserCompleteContext(ctx, userRealName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	log.Printf("WarmUp completed for user: %s", userRealName)
	return warmUpResult, nil
}

// getUserCompleteContext 获取用户的完整上下文信息
func (s *serviceImpl) getUserCompleteContext(ctx context.Context, userName string) (*WarmUpResult, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result := &WarmUpResult{
		Projects:      []ProjectWithRelation{},
		Organizations: []OrganizationWithRelation{},
		People:        []PersonWithRelation{},
		Products:      []ProductWithRelation{},
		Technologies:  []TechnologyWithRelation{},
	}

	// 1. 获取用户基本信息
	userInfo, err := s.getUserInfo(ctx, session, userName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	result.UserInfo = userInfo

	// 2. 获取相关的项目信息
	projects, err := s.getUserProjects(ctx, session, userName)
	if err != nil {
		log.Printf("Warning: failed to get user projects: %v", err)
	} else {
		result.Projects = projects
	}

	// 3. 获取相关的组织信息
	organizations, err := s.getUserOrganizations(ctx, session, userName)
	if err != nil {
		log.Printf("Warning: failed to get user organizations: %v", err)
	} else {
		result.Organizations = organizations
	}

	// 4. 获取相关的人员信息
	people, err := s.getUserRelatedPeople(ctx, session, userName)
	if err != nil {
		log.Printf("Warning: failed to get related people: %v", err)
	} else {
		result.People = people
	}

	// 5. 获取相关的产品信息
	products, err := s.getUserProducts(ctx, session, userName)
	if err != nil {
		log.Printf("Warning: failed to get user products: %v", err)
	} else {
		result.Products = products
	}

	// 6. 获取相关的技术信息
	technologies, err := s.getUserTechnologies(ctx, session, userName)
	if err != nil {
		log.Printf("Warning: failed to get user technologies: %v", err)
	} else {
		result.Technologies = technologies
	}

	return result, nil
}

// getUserInfo 获取用户基本信息
func (s *serviceImpl) getUserInfo(ctx context.Context, session neo4j.SessionWithContext, userName string) (*Person, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MATCH (u:Person {name: $userName})
			RETURN u.name as name, u.aliases as aliases, u.roles as roles,
			       u.status as status, u.contact_info as contact_info, u.notes as notes
		`

		result, err := tx.Run(ctx, cypher, map[string]any{
			"userName": userName,
		})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			person := &Person{
				Name:        getStringValue(record, "name"),
				Aliases:     getStringArrayValue(record, "aliases"),
				Roles:       getStringArrayValue(record, "roles"),
				Status:      getStringValue(record, "status"),
				ContactInfo: getStringArrayValue(record, "contact_info"),
				Notes:       getStringValue(record, "notes"),
			}
			return person, nil
		}
		return nil, fmt.Errorf("user not found: %s", userName)
	})

	if err != nil {
		return nil, err
	}

	if person, ok := result.(*Person); ok {
		return person, nil
	}
	return nil, fmt.Errorf("failed to parse user info")
}

// getUserProjects 获取用户相关的项目信息
func (s *serviceImpl) getUserProjects(ctx context.Context, session neo4j.SessionWithContext, userName string) ([]ProjectWithRelation, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MATCH (u:Person {name: $userName})-[r]->(p:Project)
			RETURN p.name as name, p.description as description, p.status as status,
			       p.start_date as start_date, p.end_date as end_date, p.url as url, p.scale as scale,
			       type(r) as relationship, properties(r) as properties
		`

		result, err := tx.Run(ctx, cypher, map[string]any{
			"userName": userName,
		})
		if err != nil {
			return nil, err
		}

		var projects []ProjectWithRelation
		for result.Next(ctx) {
			record := result.Record()
			project := &Project{
				Name:        getStringValue(record, "name"),
				Description: getStringValue(record, "description"),
				Status:      getStringValue(record, "status"),
				StartDate:   getStringValue(record, "start_date"),
				EndDate:     getStringValue(record, "end_date"),
				URL:         getStringArrayValue(record, "url"),
				Scale:       getStringValue(record, "scale"),
			}

			projectWithRel := ProjectWithRelation{
				Project:      project,
				Relationship: getStringValue(record, "relationship"),
				Properties:   getMapValue(record, "properties"),
			}
			projects = append(projects, projectWithRel)
		}
		return projects, nil
	})

	if err != nil {
		return nil, err
	}

	if projects, ok := result.([]ProjectWithRelation); ok {
		return projects, nil
	}
	return []ProjectWithRelation{}, nil
}

// 辅助函数：从Neo4j记录中安全获取字符串值
func getStringValue(record *neo4j.Record, key string) string {
	if value, ok := record.Get(key); ok && value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// 辅助函数：从Neo4j记录中安全获取字符串数组值
func getStringArrayValue(record *neo4j.Record, key string) []string {
	if value, ok := record.Get(key); ok && value != nil {
		if arr, ok := value.([]interface{}); ok {
			var result []string
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

// 辅助函数：从Neo4j记录中安全获取map值
func getMapValue(record *neo4j.Record, key string) map[string]interface{} {
	if value, ok := record.Get(key); ok && value != nil {
		if m, ok := value.(map[string]interface{}); ok {
			return m
		}
	}
	return map[string]interface{}{}
}

// getUserOrganizations 获取用户相关的组织信息
func (s *serviceImpl) getUserOrganizations(ctx context.Context, session neo4j.SessionWithContext, userName string) ([]OrganizationWithRelation, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MATCH (u:Person {name: $userName})-[r]->(o:Organization)
			RETURN o.name as name, o.description as description, o.industry as industry,
			       o.website as website, o.location_name as location_name,
			       type(r) as relationship, properties(r) as properties
		`

		result, err := tx.Run(ctx, cypher, map[string]any{
			"userName": userName,
		})
		if err != nil {
			return nil, err
		}

		var organizations []OrganizationWithRelation
		for result.Next(ctx) {
			record := result.Record()
			org := &Organization{
				Name:         getStringValue(record, "name"),
				Description:  getStringValue(record, "description"),
				Industry:     getStringValue(record, "industry"),
				Website:      getStringValue(record, "website"),
				LocationName: getStringValue(record, "location_name"),
			}

			orgWithRel := OrganizationWithRelation{
				Organization: org,
				Relationship: getStringValue(record, "relationship"),
				Properties:   getMapValue(record, "properties"),
			}
			organizations = append(organizations, orgWithRel)
		}
		return organizations, nil
	})

	if err != nil {
		return nil, err
	}

	if organizations, ok := result.([]OrganizationWithRelation); ok {
		return organizations, nil
	}
	return []OrganizationWithRelation{}, nil
}

// getUserRelatedPeople 获取用户相关的人员信息
func (s *serviceImpl) getUserRelatedPeople(ctx context.Context, session neo4j.SessionWithContext, userName string) ([]PersonWithRelation, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MATCH (u:Person {name: $userName})-[r]-(p:Person)
			WHERE p.name <> $userName
			RETURN p.name as name, p.aliases as aliases, p.roles as roles,
			       p.status as status, p.contact_info as contact_info, p.notes as notes,
			       type(r) as relationship, properties(r) as properties
		`

		result, err := tx.Run(ctx, cypher, map[string]any{
			"userName": userName,
		})
		if err != nil {
			return nil, err
		}

		var people []PersonWithRelation
		for result.Next(ctx) {
			record := result.Record()
			person := &Person{
				Name:        getStringValue(record, "name"),
				Aliases:     getStringArrayValue(record, "aliases"),
				Roles:       getStringArrayValue(record, "roles"),
				Status:      getStringValue(record, "status"),
				ContactInfo: getStringArrayValue(record, "contact_info"),
				Notes:       getStringValue(record, "notes"),
			}

			personWithRel := PersonWithRelation{
				Person:       person,
				Relationship: getStringValue(record, "relationship"),
				Properties:   getMapValue(record, "properties"),
			}
			people = append(people, personWithRel)
		}
		return people, nil
	})

	if err != nil {
		return nil, err
	}

	if people, ok := result.([]PersonWithRelation); ok {
		return people, nil
	}
	return []PersonWithRelation{}, nil
}

// getUserProducts 获取用户相关的产品信息
func (s *serviceImpl) getUserProducts(ctx context.Context, session neo4j.SessionWithContext, userName string) ([]ProductWithRelation, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MATCH (u:Person {name: $userName})-[r]->(p:Product)
			RETURN p.name as name, p.version as version, p.launch_date as launch_date,
			       p.description as description, p.url as url, p.tags as tags,
			       type(r) as relationship, properties(r) as properties
		`

		result, err := tx.Run(ctx, cypher, map[string]any{
			"userName": userName,
		})
		if err != nil {
			return nil, err
		}

		var products []ProductWithRelation
		for result.Next(ctx) {
			record := result.Record()
			product := &Product{
				Name:        getStringValue(record, "name"),
				Version:     getStringValue(record, "version"),
				LaunchDate:  getStringValue(record, "launch_date"),
				Description: getStringValue(record, "description"),
				URL:         getStringArrayValue(record, "url"),
				Tags:        getStringArrayValue(record, "tags"),
			}

			productWithRel := ProductWithRelation{
				Product:      product,
				Relationship: getStringValue(record, "relationship"),
				Properties:   getMapValue(record, "properties"),
			}
			products = append(products, productWithRel)
		}
		return products, nil
	})

	if err != nil {
		return nil, err
	}

	if products, ok := result.([]ProductWithRelation); ok {
		return products, nil
	}
	return []ProductWithRelation{}, nil
}

// getUserTechnologies 获取用户相关的技术信息
func (s *serviceImpl) getUserTechnologies(ctx context.Context, session neo4j.SessionWithContext, userName string) ([]TechnologyWithRelation, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			MATCH (u:Person {name: $userName})-[r]->(t:Technology)
			RETURN t.name as name, t.type as type, t.version as version,
			       t.description as description, t.url as url,
			       type(r) as relationship, properties(r) as properties
		`

		result, err := tx.Run(ctx, cypher, map[string]any{
			"userName": userName,
		})
		if err != nil {
			return nil, err
		}

		var technologies []TechnologyWithRelation
		for result.Next(ctx) {
			record := result.Record()
			tech := &Technology{
				Name:        getStringValue(record, "name"),
				Type:        getStringValue(record, "type"),
				Version:     getStringValue(record, "version"),
				Description: getStringValue(record, "description"),
				URL:         getStringArrayValue(record, "url"),
			}

			techWithRel := TechnologyWithRelation{
				Technology:   tech,
				Relationship: getStringValue(record, "relationship"),
				Properties:   getMapValue(record, "properties"),
			}
			technologies = append(technologies, techWithRel)
		}
		return technologies, nil
	})

	if err != nil {
		return nil, err
	}

	if technologies, ok := result.([]TechnologyWithRelation); ok {
		return technologies, nil
	}
	return []TechnologyWithRelation{}, nil
}
