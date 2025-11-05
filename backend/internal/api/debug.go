package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// ResetDBHandler drops all tables and re-runs migrations.
// This is a destructive operation and should only be used in development.
func (s *Server) ResetDBHandler(c *gin.Context) {
	log.Warn("!! Received request to reset database !!")
	err := s.store.ResetDB(c.Request.Context())
	if err != nil {
		log.Errorf("Failed to reset database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset database: " + err.Error()})
		return
	}

	log.Info("Database has been successfully reset.")
	c.JSON(http.StatusOK, gin.H{"message": "Database has been successfully reset."})
}
