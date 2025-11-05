package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSyncProgressHandler returns the current sync progress for the authenticated user
func (s *Server) GetSyncProgressHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	progress := s.getSyncProgress(userEmail)
	c.JSON(http.StatusOK, progress)
}
