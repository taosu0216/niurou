package types

type BaseResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Reply     string `json:"reply"`
	Timestamp string `json:"timestamp"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

type AddPersonNodeRequest struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Status      string   `json:"status,omitempty"`
	ContactInfo []string `json:"contact_info,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

type AddPersonNodeResponse struct {
	BaseResponse
}
