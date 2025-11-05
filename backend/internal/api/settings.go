package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"backend/internal/database"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func (s *Server) GetSettingsHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	settings, err := s.store.GetUserSettings(c.Request.Context(), userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user settings: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (s *Server) UpdateSettingsHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	var req struct {
		Enabled   bool   `json:"automation_enabled"`
		Frequency string `json:"automation_frequency"`
		TimeOfDay string `json:"automation_time"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settings data"})
		return
	}
    settings, err := s.store.UpdateUserSettings(c.Request.Context(), userEmail, req.Enabled, req.Frequency, req.TimeOfDay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user settings"})
		return
	}
    c.JSON(http.StatusOK, settings)
}

func (s *Server) SyncHistoryHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	ctx := c.Request.Context()
	defer s.clearSyncProgress(userEmail)
	
	s.updateSyncProgress(userEmail, "quick", "Checking for changes", 0, 0)

	settings, err := s.store.GetUserSettings(ctx, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user settings"})
		return
	}

	gmailService, err := s.getGmailService(ctx, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Gmail service"})
		return
	}

	// If no history ID exists or history retrieval fails, do a fallback sync
	if settings.LastHistoryID == 0 {
		log.Infof("No history ID found for user %s, initializing with recent emails", userEmail)
		s.fallbackSync(c, userEmail, ctx, gmailService)
		return
	}

	historyResponse, err := gmailService.Users.History.List(userEmail).StartHistoryId(settings.LastHistoryID).Do()
	if err != nil {
		log.Warnf("Failed to retrieve history for user %s (history ID may be too old): %v", userEmail, err)
		log.Infof("Falling back to query-based sync for user %s", userEmail)
		s.fallbackSync(c, userEmail, ctx, gmailService)
		return
	}

	if len(historyResponse.History) == 0 {
		log.Infof("No new history changes detected for user %s", userEmail)
		s.updateSyncProgress(userEmail, "quick", "Complete", 0, 0)
		c.JSON(http.StatusOK, gin.H{"message": "No new history to sync"})
		return
	}

	log.Infof("Processing %d history records for user %s", len(historyResponse.History), userEmail)
	s.updateSyncProgress(userEmail, "quick", "Processing changes", 0, 0)

	var addedMessages []*gmail.Message
	var removedMessageIds []string
	processedIds := make(map[string]bool) // Track IDs to avoid duplicates

	for _, history := range historyResponse.History {
		// Process new messages
		if history.MessagesAdded != nil {
			for _, msg := range history.MessagesAdded {
				if processedIds[msg.Message.Id] {
					continue
				}
				fullMsg, err := gmailService.Users.Messages.Get(userEmail, msg.Message.Id).Format("full").Do()
				if err != nil {
					log.Errorf("Failed to get message %s: %v", msg.Message.Id, err)
					continue
				}
				addedMessages = append(addedMessages, fullMsg)
				processedIds[msg.Message.Id] = true
			}
		}
		
		// Process label additions (e.g., moved from archive to inbox)
		if history.LabelsAdded != nil {
			for _, labelAdded := range history.LabelsAdded {
				if processedIds[labelAdded.Message.Id] {
					continue
				}
				// Check if INBOX label was added
				hasInbox := false
				for _, labelId := range labelAdded.LabelIds {
					if labelId == "INBOX" {
						hasInbox = true
						break
					}
				}
				if hasInbox {
					fullMsg, err := gmailService.Users.Messages.Get(userEmail, labelAdded.Message.Id).Format("full").Do()
					if err != nil {
						log.Errorf("Failed to get message %s after label addition: %v", labelAdded.Message.Id, err)
						continue
					}
					addedMessages = append(addedMessages, fullMsg)
					processedIds[labelAdded.Message.Id] = true
					log.Infof("Detected INBOX label added to message %s", labelAdded.Message.Id)
				}
			}
		}
		
		// Process label removals (e.g., moved from inbox to archive)
		if history.LabelsRemoved != nil {
			for _, labelRemoved := range history.LabelsRemoved {
				// Check if INBOX label was removed
				hasInbox := false
				for _, labelId := range labelRemoved.LabelIds {
					if labelId == "INBOX" {
						hasInbox = true
						break
					}
				}
				if hasInbox {
					removedMessageIds = append(removedMessageIds, labelRemoved.Message.Id)
					log.Infof("Detected INBOX label removed from message %s", labelRemoved.Message.Id)
				}
			}
		}
		
		// Process deleted messages
		if history.MessagesDeleted != nil {
			for _, msg := range history.MessagesDeleted {
				removedMessageIds = append(removedMessageIds, msg.Message.Id)
			}
		}
	}

	log.Infof("History sync summary: %d messages to add/update, %d messages to remove", len(addedMessages), len(removedMessageIds))
	s.updateSyncProgress(userEmail, "quick", "Updating database", 0, 0)

	if len(addedMessages) > 0 {
		emails := s.gmailMessagesToEmails(addedMessages, userEmail)
		if err := s.store.UpsertEmails(ctx, emails); err != nil {
			log.Errorf("Failed to upsert emails: %v", err)
		} else {
			log.Infof("Successfully upserted %d emails", len(emails))
		}
	}

	if len(removedMessageIds) > 0 {
		for _, msgId := range removedMessageIds {
			if err := s.store.DeleteEmail(ctx, msgId); err != nil {
				log.Errorf("Failed to delete email %s: %v", msgId, err)
			}
		}
		log.Infof("Successfully removed %d emails from inbox", len(removedMessageIds))
	}

	newHistoryID := historyResponse.HistoryId
	if err := s.store.UpdateHistoryID(ctx, userEmail, newHistoryID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update history ID"})
		return
	}

	log.Infof("Updated history ID from %d to %d", settings.LastHistoryID, newHistoryID)
	s.updateSyncProgress(userEmail, "quick", "Complete", 0, 0)

	var message string
	if len(addedMessages) > 0 || len(removedMessageIds) > 0 {
		message = fmt.Sprintf("Quick sync complete: %d added/updated, %d removed", len(addedMessages), len(removedMessageIds))
	} else {
		message = "Quick sync complete: No changes detected"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// fallbackSync performs a query-based sync when history tracking is not available
func (s *Server) fallbackSync(c *gin.Context, userEmail string, ctx context.Context, gmailService *gmail.Service) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create email service"})
		return
	}

	log.Infof("Starting fallback sync for user: %s (quick catchup of recent emails)", userEmail)
	s.updateSyncProgress(userEmail, "quick", "Fetching recent emails", 0, 0)

	// Fetch recent INBOX emails (last 7 days) for quick initialization
	// After this, the History API will track ALL future changes efficiently
	query := "newer_than:7d"
	ids, err := emailService.ListMessageIDs("me", query, []string{"INBOX"}, 500)
	if err != nil {
		log.Errorf("Failed to list recent messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list recent messages"})
		return
	}

	log.Infof("Found %d recent INBOX emails for fallback sync", len(ids))

	if len(ids) == 0 {
		// No emails but we still need to initialize history ID
		// Get a message to extract history ID
		initIds, err := emailService.ListMessageIDs("me", "", []string{"INBOX"}, 1)
		if err == nil && len(initIds) > 0 {
			msg, err := gmailService.Users.Messages.Get(userEmail, initIds[0]).Format("minimal").Do()
			if err == nil && msg.HistoryId > 0 {
				_ = s.store.UpdateHistoryID(ctx, userEmail, msg.HistoryId)
				log.Infof("Initialized history ID to %d for user %s", msg.HistoryId, userEmail)
			}
		}
		c.JSON(http.StatusOK, gin.H{"message": "No recent emails to sync, history tracking initialized"})
		return
	}

	s.updateSyncProgress(userEmail, "quick", "Processing email details", 0, 0)
	
	messages, err := emailService.GetMessageDetails("me", ids)
	if err != nil {
		log.Errorf("Failed to get message details: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get message details"})
		return
	}

	log.Infof("Successfully fetched details for %d messages", len(messages))
	s.updateSyncProgress(userEmail, "quick", "Saving emails", 0, 0)

	emails := s.gmailMessagesToEmails(messages, userEmail)
	if len(emails) > 0 {
		if err := s.store.UpsertEmails(ctx, emails); err != nil {
			log.Errorf("Failed to upsert emails: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save emails"})
			return
		}
	}

	// Initialize history ID from the first message
	if len(messages) > 0 && messages[0].HistoryId > 0 {
		if err := s.store.UpdateHistoryID(ctx, userEmail, messages[0].HistoryId); err != nil {
			log.Errorf("Failed to update history ID: %v", err)
		} else {
			log.Infof("Initialized history ID to %d for user %s", messages[0].HistoryId, userEmail)
		}
	}

	s.updateSyncProgress(userEmail, "quick", "Complete", 0, 0)
	
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Synced %d recent emails, history tracking initialized", len(emails)),
		"total":   len(emails),
	})
}

// getGmailService creates a Gmail service for the given user
func (s *Server) getGmailService(ctx context.Context, userEmail string) (*gmail.Service, error) {
	tok, err := s.tokenStore.Get(ctx, userEmail)
	if err != nil {
		return nil, err
	}
	if tok == nil {
		return nil, errors.New("no token for user")
	}

	srv, err := gmail.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(tok)))
	if err != nil {
		return nil, err
	}
	return srv, nil
}

// gmailMessagesToEmails converts Gmail messages to database Email objects
func (s *Server) gmailMessagesToEmails(messages []*gmail.Message, userEmail string) []database.Email {
	var emails []database.Email
	for _, msg := range messages {
		var sender, subject, snippet string
		var date time.Time

		// Extract headers
		if msg.Payload != nil && msg.Payload.Headers != nil {
			for _, header := range msg.Payload.Headers {
				switch header.Name {
				case "From":
					sender = header.Value
				case "Subject":
					subject = header.Value
				case "Date":
					if parsedTime, err := time.Parse(time.RFC1123, header.Value); err == nil {
						date = parsedTime
					} else if parsedTime, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
						date = parsedTime
					} else {
						date = time.Unix(msg.InternalDate/1000, 0)
					}
				}
			}
		}

		// Use fallback values if headers are missing
		if sender == "" && msg.Payload != nil && msg.Payload.Headers != nil {
			for _, header := range msg.Payload.Headers {
				if header.Name == "From" {
					sender = header.Value
					break
				}
			}
		}
		if subject == "" {
			subject = "(No Subject)"
		}
		if snippet == "" {
			snippet = msg.Snippet
		}
		if date.IsZero() {
			date = time.Unix(msg.InternalDate/1000, 0)
		}

		// Determine read status
		isRead := true
		for _, labelID := range msg.LabelIds {
			if labelID == "UNREAD" {
				isRead = false
				break
			}
		}

		emails = append(emails, database.Email{
			ID:      msg.Id,
			UserID:  userEmail,
			Sender:  sender,
			Subject: subject,
			Snippet: snippet,
			Date:    date,
			Read:    isRead,
		})
	}
	return emails
}
