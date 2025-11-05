package api

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// GetStatsHandler fetches various counts for the dashboard.
func (s *Server) GetStatsHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	ctx := c.Request.Context()

	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get email service: " + err.Error()})
		return
	}

	var syncedCount, trashCount, archivedCount int
	var wg sync.WaitGroup
	wg.Add(3) // We are running three concurrent operations

	// Goroutine to get synced email count from our database
	go func() {
		defer wg.Done()
		count, err := s.store.CountEmails(ctx, userEmail)
		if err != nil {
			log.Errorf("Failed to count synced emails: %v", err)
			// Don't fail the whole request, just return 0 for this stat
		}
		syncedCount = count
	}()

	// Goroutine to get trash count from Gmail API
	go func() {
		defer wg.Done()
		count, err := emailService.GetLabelMessageCount("me", "TRASH")
		if err != nil {
			log.Errorf("Failed to get trash count: %v", err)
		}
		trashCount = count
	}()

	// Goroutine to get archived count from Gmail API
	go func() {
		defer wg.Done()
		count, err := emailService.CountArchivedMessages("me")
		if err != nil {
			log.Errorf("Failed to get archived count: %v", err)
		}
		archivedCount = count
	}()

	wg.Wait() // Wait for all three operations to complete

	c.JSON(http.StatusOK, gin.H{
		"synced":   syncedCount,
		"trash":    trashCount,
		"archived": archivedCount,
	})
}
