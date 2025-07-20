package server

import (
	"log"
	"net/http"

	"niurou/internal/app/agent/agents"
	"niurou/internal/data/graphDB"
	"niurou/internal/data/memManager"
	"niurou/types"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleAddPersonNode(c *gin.Context) {
	var req types.AddPersonNodeRequest
	log.Println("req111 ", req)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.AddPersonNodeResponse{
			BaseResponse: types.BaseResponse{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid request format",
			},
		})
		return
	}
	log.Println("req222 ", req)

	memoryManager, err := memManager.InitMemClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.AddPersonNodeResponse{
			BaseResponse: types.BaseResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    "Failed to initialize memory manager",
			},
		})
	}

	personNode := graphDB.Person{
		Name:        req.Name,
		Aliases:     req.Aliases,
		Roles:       req.Roles,
		Status:      req.Status,
		ContactInfo: req.ContactInfo,
		Notes:       req.Notes,
	}

	err = memoryManager.AddPersonNode(c.Request.Context(), &personNode, req.Labels)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.AddPersonNodeResponse{
			BaseResponse: types.BaseResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    "Failed to add person node",
			},
		})
	}
	c.JSON(http.StatusOK, types.AddPersonNodeResponse{
		BaseResponse: types.BaseResponse{
			StatusCode: http.StatusOK,
			Message:    "Person node added successfully",
		},
	})
}

func (s *Server) handleAddPersonNodeByAgent(c *gin.Context) {
	agent := agents.GetAddPersonNodeAgent()
	resp, err := agent.AddPersonNodeFn(c.Request.Context(), c.Query("input"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.AddPersonNodeResponse{
			BaseResponse: types.BaseResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    "Failed to add person node",
			},
		})
	}
	c.JSON(http.StatusOK, types.AddPersonNodeResponse{
		BaseResponse: types.BaseResponse{
			StatusCode: http.StatusOK,
			Message:    resp,
		},
	})
}
