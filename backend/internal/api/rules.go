package api

import (
	"net/http"

	"backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetRulesHandler fetches rules from the database.
func (s *Server) GetRulesHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	rules, err := s.store.ListRules(c.Request.Context(), userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// CreateRuleHandler creates a new rule in the database.
func (s *Server) CreateRuleHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	var req struct {
		Type    string `json:"type" binding:"required"`
		Value   string `json:"value" binding:"required"`
		Action  string `json:"action"`
		AgeDays int    `json:"age_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule data: " + err.Error()})
		return
	}
	// Default action if not provided
	if req.Action == "" {
		req.Action = "DELETE"
	}

	params := database.CreateRuleParams{
		ID:      uuid.NewString(),
		UserID:  userEmail,
		Type:    req.Type,
		Value:   req.Value,
		Action:  req.Action,
		AgeDays: req.AgeDays,
	}

	rule, err := s.store.CreateRule(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Rule created", "rule": rule})
}

// DeleteRuleHandler deletes a rule from the database.
func (s *Server) DeleteRuleHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	ruleID := c.Param("id")

	err := s.store.DeleteRule(c.Request.Context(), ruleID, userEmail)
	if err != nil {
		if err.Error() == "rule not found or not owned by user" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted"})
}

// UpdateRuleHandler updates a rule in the database.
func (s *Server) UpdateRuleHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	ruleID := c.Param("id")
	var req struct {
		Type    string `json:"type" binding:"required"`
		Value   string `json:"value" binding:"required"`
		Action  string `json:"action"`
		AgeDays int    `json:"age_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule data"})
		return
	}
	if req.Action == "" {
		req.Action = "DELETE"
	}

	rule, err := s.store.UpdateRule(c.Request.Context(), ruleID, userEmail, req.Type, req.Value, req.Action, req.AgeDays)
	if err != nil {
		if err.Error() == "rule not found or not owned by user" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule updated", "rule": rule})
}
