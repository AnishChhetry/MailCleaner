package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (s *Server) GetSenderAnalyticsHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	analytics, err := s.store.GetTopSenders(c.Request.Context(), userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch analytics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

// GetSubscribedSendersHandler fetches senders with List-Unsubscribe headers (newsletters, marketing emails)
func (s *Server) GetSubscribedSendersHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get email service"})
		return
	}

	// Fetch all inbox emails (limited to recent ones for performance)
	ids, err := emailService.ListMessageIDs("me", "", []string{"INBOX"}, 500)
	if err != nil {
		log.Errorf("Failed to list messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list messages"})
		return
	}

	// Get message details to check for List-Unsubscribe header
	messages, err := emailService.GetMessageDetails("me", ids)
	if err != nil {
		log.Errorf("Failed to get message details: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get message details"})
		return
	}

	// Track unique senders with List-Unsubscribe headers
	type SubscribedSender struct {
		Sender            string `json:"sender"`
		Count             int    `json:"count"`
		UnsubscribeHeader string `json:"unsubscribe_header"`
		SampleSubject     string `json:"sample_subject"`
	}

	senderMap := make(map[string]*SubscribedSender)

	for _, msg := range messages {
		var sender, subject, unsubscribeHeader string
		
		if msg.Payload != nil && msg.Payload.Headers != nil {
			for _, header := range msg.Payload.Headers {
				switch header.Name {
				case "From":
					sender = header.Value
				case "Subject":
					subject = header.Value
				case "List-Unsubscribe":
					unsubscribeHeader = header.Value
				}
			}
		}

		// Only include senders with List-Unsubscribe header
		if sender != "" && unsubscribeHeader != "" {
			// Clean sender email (extract email from "Name <email>" format)
			cleanSender := extractEmail(sender)
			if cleanSender == "" {
				cleanSender = sender
			}

			if existing, ok := senderMap[cleanSender]; ok {
				existing.Count++
			} else {
				senderMap[cleanSender] = &SubscribedSender{
					Sender:            cleanSender,
					Count:             1,
					UnsubscribeHeader: unsubscribeHeader,
					SampleSubject:     subject,
				}
			}
		}
	}

	// Convert map to slice
	var senders []SubscribedSender
	for _, sender := range senderMap {
		senders = append(senders, *sender)
	}

	c.JSON(http.StatusOK, gin.H{
		"subscribed_senders": senders,
		"total":              len(senders),
	})
}

// extractEmail extracts email address from "Name <email>" format
func extractEmail(from string) string {
	// Look for email in angle brackets
	start := strings.Index(from, "<")
	end := strings.Index(from, ">")
	if start != -1 && end != -1 && end > start {
		return from[start+1 : end]
	}
	// If no angle brackets, return as is (might already be just an email)
	return from
}
